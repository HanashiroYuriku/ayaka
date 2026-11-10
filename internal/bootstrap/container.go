package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"be-ayaka/config"
	"be-ayaka/internal/adapter/email"
	"be-ayaka/internal/adapter/repository"
	"be-ayaka/internal/adapter/token"
	"be-ayaka/internal/core/port"
	"be-ayaka/internal/core/service"
	"be-ayaka/internal/delivery/http"
	"be-ayaka/pkg/hash"
	"be-ayaka/pkg/validator"

	"gorm.io/gorm"
)

// Compile-time checks that adapters satisfy the core ports.
var (
	_ port.UserRepository         = (*repository.UserRepo)(nil)
	_ port.RefreshTokenRepository = (*repository.RefreshTokenRepo)(nil)
	_ port.UserTokenRepository    = (*repository.UserTokenRepo)(nil)
	_ port.TxManager              = (*repository.TxManager)(nil)
	_ port.PasswordHasher         = (*hash.Bcrypt)(nil)
	_ port.AccessTokenProvider    = (*token.JWTProvider)(nil)
	_ port.EmailSender            = (*email.SMTPSender)(nil)
	_ port.EmailSender            = (*email.LogSender)(nil)
	_ port.EmailSender            = (*email.AsyncSender)(nil)
	_ port.AuthMailer             = (*email.TemplateMailer)(nil)
)

const emailSendTimeout = 30 * time.Second

// Container is the composition root: the only place that knows concrete types.
// To add a feature: repository -> service -> handler here, then a route in routes.go.
type Container struct {
	Validator     validator.Validator
	AccessTokens  port.AccessTokenProvider
	AuthService   service.AuthService
	UserService   service.UserService
	RefreshTokens port.RefreshTokenRepository
	UserTokens    port.UserTokenRepository

	Health *http.HealthHandler
	Auth   *http.AuthHandler
	User   *http.UserHandler

	mail *email.AsyncSender
}

func NewContainer(cfg *config.Config, db *gorm.DB, sqlDB *sql.DB, log *slog.Logger) (*Container, error) {
	// shared infrastructure
	v := validator.New(db)
	tx := repository.NewTxManager(db)
	hasher := hash.NewBcrypt(hash.DefaultCost)
	accessTokens := token.NewJWTProvider(token.Options{
		Issuer: cfg.App.Name,
		Secret: cfg.JWT.Secret,
		TTL:    cfg.JWT.Expired,
	})

	// email: templates -> async -> smtp | log
	var transport port.EmailSender = email.NewLogSender(log)
	if cfg.Email.Driver == "smtp" {
		transport = email.NewSMTPSender(cfg.Email.Host, cfg.Email.Port, cfg.Email.User, cfg.Email.Pass, cfg.Email.From)
	} else if !cfg.App.IsDevelopment() {
		log.Warn("email.driver=log: emails (including reset links) are written to the log, not sent")
	}
	asyncMail := email.NewAsyncSender(transport, emailSendTimeout)
	mailer, err := email.NewTemplateMailer(cfg.App.Name, asyncMail)
	if err != nil {
		return nil, err
	}

	// repositories
	userRepo := repository.NewUserRepo(db)
	refreshRepo := repository.NewRefreshTokenRepo(db)
	userTokenRepo := repository.NewUserTokenRepo(db)

	// services
	authSvc := service.NewAuthService(service.AuthDeps{
		Users:         userRepo,
		RefreshTokens: refreshRepo,
		UserTokens:    userTokenRepo,
		Hasher:        hasher,
		AccessTokens:  accessTokens,
		Mailer:        mailer,
		Tx:            tx,
	}, service.AuthConfig{
		RefreshTTL:               cfg.Auth.RefreshTTL,
		RequireEmailVerification: cfg.Auth.RequireEmailVerification,
		MaxFailedLogins:          cfg.Auth.MaxFailedLogins,
		LockoutDuration:          cfg.Auth.LockoutDuration,
		PasswordResetTTL:         cfg.Auth.PasswordResetTTL,
		EmailVerificationTTL:     cfg.Auth.EmailVerificationTTL,
		EmailCooldown:            cfg.Auth.EmailCooldown,
		ResetPasswordURL:         frontendURL(cfg.Frontend, cfg.Frontend.ResetPasswordPath),
		VerifyEmailURL:           frontendURL(cfg.Frontend, cfg.Frontend.VerifyEmailPath),
	})
	userSvc := service.NewUserService(userRepo)

	return &Container{
		Validator:     v,
		AccessTokens:  accessTokens,
		AuthService:   authSvc,
		UserService:   userSvc,
		RefreshTokens: refreshRepo,
		UserTokens:    userTokenRepo,

		Health: http.NewHealthHandler(cfg.App.Name, cfg.App.Version, map[string]http.Pinger{"database": sqlDB}),
		Auth:   http.NewAuthHandler(authSvc, v),
		User:   http.NewUserHandler(userSvc, v),

		mail: asyncMail,
	}, nil
}

// Close waits for background work (pending emails) to finish.
func (c *Container) Close(ctx context.Context) error {
	if err := c.mail.Wait(ctx); err != nil {
		return fmt.Errorf("close container: %w", err)
	}
	return nil
}

func frontendURL(f config.FrontendConfig, path string) string {
	return strings.TrimRight(f.URL, "/") + "/" + strings.TrimLeft(path, "/")
}
