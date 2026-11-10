package bootstrap

import (
	"database/sql"

	"be-ayaka/config"
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
	_ port.UserRepository = (*repository.UserRepo)(nil)
	_ port.TxManager      = (*repository.TxManager)(nil)
	_ port.PasswordHasher = (*hash.Bcrypt)(nil)
	_ port.TokenProvider  = (*token.JWTProvider)(nil)
)

// Container is the composition root: the only place that knows concrete types.
// To add a feature: repository -> service -> handler here, then a route in routes.go.
type Container struct {
	Tokens port.TokenProvider

	Health *http.HealthHandler
	Auth   *http.AuthHandler
	User   *http.UserHandler
}

func NewContainer(cfg *config.Config, db *gorm.DB, sqlDB *sql.DB) *Container {
	// shared infrastructure
	v := validator.New(db)
	tx := repository.NewTxManager(db)
	hasher := hash.NewBcrypt(hash.DefaultCost)
	tokens := token.NewJWTProvider(token.Options{
		Issuer:        cfg.App.Name,
		AccessSecret:  cfg.JWT.Secret,
		AccessTTL:     cfg.JWT.Expired,
		RefreshSecret: cfg.JWT.RefreshSecret,
		RefreshTTL:    cfg.JWT.RefreshExpired,
	})

	// repositories
	userRepo := repository.NewUserRepo(db)

	// services
	authSvc := service.NewAuthService(userRepo, hasher, tokens, tx)
	userSvc := service.NewUserService(userRepo)

	return &Container{
		Tokens: tokens,
		Health: http.NewHealthHandler(cfg.App.Name, cfg.App.Version, map[string]http.Pinger{"database": sqlDB}),
		Auth:   http.NewAuthHandler(authSvc, v),
		User:   http.NewUserHandler(userSvc, v),
	}
}
