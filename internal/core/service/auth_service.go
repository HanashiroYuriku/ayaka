package service

import (
	"context"
	"errors"
	"strings"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/core/port"
	"be-ayaka/pkg/generateid"
)

var (
	// ErrInvalidCredentials is deliberately generic so it does not reveal
	// whether the email exists.
	ErrInvalidCredentials = apperror.Unauthorized("invalid email or password")
	ErrInvalidToken       = apperror.Unauthorized("invalid or expired token")
)

// AuthService handles registration and authentication use cases.
type AuthService interface {
	Register(ctx context.Context, in model.RegisterInput) (*model.UserProfile, error)
	Login(ctx context.Context, in model.LoginInput) (*model.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (*model.AuthTokens, error)
}

type authService struct {
	users  port.UserRepository
	hasher port.PasswordHasher
	tokens port.TokenProvider
	tx     port.TxManager
	// dummyHash is compared when the user does not exist so that login takes
	// roughly the same time either way (mitigates user enumeration by timing).
	dummyHash string
}

func NewAuthService(users port.UserRepository, hasher port.PasswordHasher, tokens port.TokenProvider, tx port.TxManager) AuthService {
	dummy, _ := hasher.Hash("ayaka-dummy-password")
	return &authService{users: users, hasher: hasher, tokens: tokens, tx: tx, dummyHash: dummy}
}

func (s *authService) Register(ctx context.Context, in model.RegisterInput) (*model.UserProfile, error) {
	hashed, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	user := &entity.User{
		Username:    strings.TrimSpace(in.Username),
		Email:       normalizeEmail(in.Email),
		DisplayName: strings.TrimSpace(in.DisplayName),
		Password:    hashed,
		Role:        entity.RoleUser,
	}
	user.ID = generateid.New("USR")

	// The transaction is not strictly needed for a single insert; it shows where
	// to put multi-step writes (e.g. create user + profile + audit log).
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		return s.users.Create(ctx, user)
	})
	if err != nil {
		return nil, err
	}

	profile := model.NewUserProfile(user)
	return &profile, nil
}

func (s *authService) Login(ctx context.Context, in model.LoginInput) (*model.AuthTokens, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(in.Email))
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			_ = s.hasher.Compare(s.dummyHash, in.Password)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := s.hasher.Compare(user.Password, in.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issue(user)
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (*model.AuthTokens, error) {
	claims, err := s.tokens.Parse(refreshToken, model.TokenTypeRefresh)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Reload the user so deleted users cannot refresh and role changes apply.
	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	return s.issue(user)
}

func (s *authService) issue(user *entity.User) (*model.AuthTokens, error) {
	tokens, err := s.tokens.Issue(user.ID, user.Role)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return tokens, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
