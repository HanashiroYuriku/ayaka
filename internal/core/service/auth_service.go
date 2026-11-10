package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/core/port"
	"be-ayaka/pkg/generateid"
	"be-ayaka/pkg/logger"
	"be-ayaka/pkg/securetoken"
)

var (
	// ErrInvalidCredentials is deliberately generic so it does not reveal
	// whether the email exists.
	ErrInvalidCredentials = apperror.Unauthorized("invalid email or password")
	ErrInvalidToken       = apperror.Unauthorized("invalid or expired token")
	ErrAccountLocked      = apperror.TooManyRequests("too many failed login attempts, please try again later")
	ErrEmailNotVerified   = apperror.Forbidden("email address is not verified")
)

// AuthConfig holds the security settings of the auth flows.
type AuthConfig struct {
	RefreshTTL               time.Duration
	RequireEmailVerification bool
	MaxFailedLogins          int
	LockoutDuration          time.Duration
	PasswordResetTTL         time.Duration
	EmailVerificationTTL     time.Duration
	EmailCooldown            time.Duration
	// Frontend pages receiving ?token=... from emails.
	ResetPasswordURL string
	VerifyEmailURL   string
}

// AuthService handles registration, sessions, passwords and email verification.
type AuthService interface {
	// sessions
	Register(ctx context.Context, in model.RegisterInput) (*model.UserProfile, error)
	Login(ctx context.Context, in model.LoginInput) (*model.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string, client model.ClientInfo) (*model.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID string) error

	// passwords
	ChangePassword(ctx context.Context, in model.ChangePasswordInput) error
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, in model.ResetPasswordInput) error

	// email verification
	VerifyEmail(ctx context.Context, token string) error
	ResendVerification(ctx context.Context, email string) error

	// administration
	CreateAdmin(ctx context.Context, in model.RegisterInput) (*model.UserProfile, error)
}

// AuthDeps groups the ports AuthService depends on.
type AuthDeps struct {
	Users         port.UserRepository
	RefreshTokens port.RefreshTokenRepository
	UserTokens    port.UserTokenRepository
	Hasher        port.PasswordHasher
	AccessTokens  port.AccessTokenProvider
	Mailer        port.AuthMailer
	Tx            port.TxManager
}

type authService struct {
	AuthDeps
	cfg AuthConfig
	now func() time.Time
	// dummyHash is compared when the user does not exist so that login takes
	// roughly the same time either way (mitigates user enumeration by timing).
	dummyHash string
}

func NewAuthService(deps AuthDeps, cfg AuthConfig) AuthService {
	dummy, _ := deps.Hasher.Hash("ayaka-dummy-password")
	return &authService{AuthDeps: deps, cfg: cfg, now: time.Now, dummyHash: dummy}
}

// ============================================================ register

func (s *authService) Register(ctx context.Context, in model.RegisterInput) (*model.UserProfile, error) {
	user, err := s.newUser(in, entity.RoleUser)
	if err != nil {
		return nil, err
	}

	var verifyToken string
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Users.Create(ctx, user); err != nil {
			return err
		}
		verifyToken, err = s.issueUserToken(ctx, user.ID, entity.PurposeEmailVerification, s.cfg.EmailVerificationTTL)
		return err
	})
	if err != nil {
		return nil, err
	}

	// Sent after commit so we never email a link for a user that does not exist.
	s.sendVerification(ctx, user, verifyToken)

	profile := model.NewUserProfile(user)
	return &profile, nil
}

// CreateAdmin creates an already verified administrator (used by the CLI).
func (s *authService) CreateAdmin(ctx context.Context, in model.RegisterInput) (*model.UserProfile, error) {
	user, err := s.newUser(in, entity.RoleAdmin)
	if err != nil {
		return nil, err
	}
	now := s.now()
	user.EmailVerifiedAt = &now

	if err := s.Users.Create(ctx, user); err != nil {
		return nil, err
	}
	profile := model.NewUserProfile(user)
	return &profile, nil
}

func (s *authService) newUser(in model.RegisterInput, role entity.Role) (*entity.User, error) {
	hashed, err := s.Hasher.Hash(in.Password)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	user := &entity.User{
		Username:    strings.TrimSpace(in.Username),
		Email:       normalizeEmail(in.Email),
		DisplayName: strings.TrimSpace(in.DisplayName),
		Password:    hashed,
		Role:        role,
	}
	user.ID = generateid.New("USR")
	return user, nil
}

// ============================================================ login / sessions

func (s *authService) Login(ctx context.Context, in model.LoginInput) (*model.AuthTokens, error) {
	user, err := s.Users.FindByEmail(ctx, normalizeEmail(in.Email))
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			_ = s.Hasher.Compare(s.dummyHash, in.Password)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	now := s.now()
	// Checked before the password so a locked account cannot be brute-forced.
	if user.IsLocked(now) {
		return nil, ErrAccountLocked
	}

	if err := s.Hasher.Compare(user.Password, in.Password); err != nil {
		if err := s.Users.RegisterFailedLogin(ctx, user.ID, s.cfg.MaxFailedLogins, now.Add(s.cfg.LockoutDuration)); err != nil {
			return nil, err
		}
		return nil, ErrInvalidCredentials
	}

	// Checked after the password so it does not leak to someone guessing.
	if s.cfg.RequireEmailVerification && !user.IsEmailVerified() {
		return nil, ErrEmailNotVerified
	}

	if user.FailedLoginAttempts > 0 || user.LockedUntil != nil {
		if err := s.Users.ResetFailedLogins(ctx, user.ID); err != nil {
			return nil, err
		}
	}

	var issued *issuedTokens
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		issued, err = s.issueSession(ctx, user, generateid.New("SES"), in.Client)
		return err
	})
	if err != nil {
		return nil, err
	}
	return issued.AuthTokens, nil
}

// Refresh rotates the refresh token. Reusing an already rotated token revokes
// the whole session, because it means the token was stolen or replayed.
func (s *authService) Refresh(ctx context.Context, refreshToken string, client model.ClientInfo) (*model.AuthTokens, error) {
	var (
		tokens *issuedTokens
		reused bool
	)
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		current, err := s.RefreshTokens.FindByHash(ctx, securetoken.Hash(refreshToken), true)
		if err != nil {
			if errors.Is(err, apperror.ErrNotFound) {
				return ErrInvalidToken
			}
			return err
		}

		now := s.now()
		if current.IsRevoked() && current.ReplacedBy == nil {
			// Revoked by logout / password change: simply no longer valid.
			return ErrInvalidToken
		}
		if current.IsRevoked() {
			// Already rotated: someone is replaying an old token (theft or replay).
			// Commit the family revocation, then report the error outside the tx.
			reused = true
			logger.FromContext(ctx).Warn("refresh token reuse detected, revoking session",
				slog.String("user_id", current.UserID), slog.String("session_id", current.FamilyID))
			return s.RefreshTokens.RevokeFamily(ctx, current.FamilyID, now)
		}
		if current.IsExpired(now) {
			return ErrInvalidToken
		}

		// Reload the user so deleted users cannot refresh and role changes apply.
		user, err := s.Users.FindByID(ctx, current.UserID)
		if err != nil {
			if errors.Is(err, apperror.ErrNotFound) {
				return ErrInvalidToken
			}
			return err
		}

		tokens, err = s.issueSession(ctx, user, current.FamilyID, client)
		if err != nil {
			return err
		}
		return s.RefreshTokens.Revoke(ctx, current.ID, &tokens.refreshID, now)
	})
	if err != nil {
		return nil, err
	}
	if reused {
		return nil, ErrInvalidToken
	}
	return tokens.AuthTokens, nil
}

// Logout revokes the session the refresh token belongs to. It is idempotent:
// unknown or already revoked tokens are not an error.
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	current, err := s.RefreshTokens.FindByHash(ctx, securetoken.Hash(refreshToken), false)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil
		}
		return err
	}
	return s.RefreshTokens.RevokeFamily(ctx, current.FamilyID, s.now())
}

// LogoutAll revokes every session of the user (all devices).
func (s *authService) LogoutAll(ctx context.Context, userID string) error {
	return s.RefreshTokens.RevokeAllForUser(ctx, userID, s.now())
}

// issuedTokens carries the new refresh token ID alongside the public result.
type issuedTokens struct {
	*model.AuthTokens
	refreshID string
}

// issueSession creates a refresh token in sessionID and a matching access token.
func (s *authService) issueSession(ctx context.Context, user *entity.User, sessionID string, client model.ClientInfo) (*issuedTokens, error) {
	plain, hash, err := securetoken.New()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	now := s.now()
	rt := &entity.RefreshToken{
		ID:        generateid.New("RT"),
		UserID:    user.ID,
		FamilyID:  sessionID,
		TokenHash: hash,
		ExpiresAt: now.Add(s.cfg.RefreshTTL),
		UserAgent: truncate(client.UserAgent, 255),
		IP:        truncate(client.IP, 64),
		CreatedAt: now,
	}
	if err := s.RefreshTokens.Create(ctx, rt); err != nil {
		return nil, err
	}

	access, accessExp, err := s.AccessTokens.Issue(user.ID, user.Role, sessionID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return &issuedTokens{
		AuthTokens: &model.AuthTokens{
			AccessToken:      access,
			RefreshToken:     plain,
			TokenType:        "Bearer",
			AccessExpiresAt:  accessExp,
			RefreshExpiresAt: rt.ExpiresAt,
		},
		refreshID: rt.ID,
	}, nil
}

// ============================================================ helpers

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
