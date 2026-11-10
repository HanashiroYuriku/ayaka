// Package mocks contains testify mocks and fakes for the core ports and services.
//
// Each mock has a compile-time assertion so it breaks loudly when an interface changes.
package mocks

import (
	"context"
	"time"

	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/core/port"
	"be-ayaka/internal/core/service"
	"be-ayaka/pkg/validator"

	"github.com/stretchr/testify/mock"
)

var (
	_ port.UserRepository         = (*UserRepo)(nil)
	_ port.RefreshTokenRepository = (*RefreshTokenRepo)(nil)
	_ port.UserTokenRepository    = (*UserTokenRepo)(nil)
	_ port.PasswordHasher         = (*Hasher)(nil)
	_ port.AccessTokenProvider    = (*AccessTokenProvider)(nil)
	_ port.TxManager              = (*FakeTx)(nil)
	_ port.EmailSender            = (*EmailSender)(nil)
	_ port.AuthMailer             = (*AuthMailer)(nil)
	_ validator.Validator         = (*Validator)(nil)
	_ service.AuthService         = (*AuthService)(nil)
	_ service.UserService         = (*UserService)(nil)
)

// getOrNil returns args.Get(i) as T, or the zero value when it is nil.
func getOrNil[T any](args mock.Arguments, i int) T {
	var zero T
	if v := args.Get(i); v != nil {
		return v.(T)
	}
	return zero
}

// ---------------------------------------------------------------- repositories

type UserRepo struct{ mock.Mock }

func (m *UserRepo) FindByID(ctx context.Context, id string) (*entity.User, error) {
	args := m.Called(ctx, id)
	return getOrNil[*entity.User](args, 0), args.Error(1)
}

func (m *UserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	return getOrNil[*entity.User](args, 0), args.Error(1)
}

func (m *UserRepo) List(ctx context.Context, page model.PageRequest) ([]entity.User, int64, error) {
	args := m.Called(ctx, page)
	return getOrNil[[]entity.User](args, 0), args.Get(1).(int64), args.Error(2)
}

func (m *UserRepo) Create(ctx context.Context, user *entity.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *UserRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	return m.Called(ctx, id, passwordHash).Error(0)
}

func (m *UserRepo) MarkEmailVerified(ctx context.Context, id string, at time.Time) error {
	return m.Called(ctx, id, at).Error(0)
}

func (m *UserRepo) RegisterFailedLogin(ctx context.Context, id string, maxAttempts int, lockUntil time.Time) error {
	return m.Called(ctx, id, maxAttempts, lockUntil).Error(0)
}

func (m *UserRepo) ResetFailedLogins(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

type RefreshTokenRepo struct{ mock.Mock }

func (m *RefreshTokenRepo) Create(ctx context.Context, t *entity.RefreshToken) error {
	return m.Called(ctx, t).Error(0)
}

func (m *RefreshTokenRepo) FindByHash(ctx context.Context, hash string, forUpdate bool) (*entity.RefreshToken, error) {
	args := m.Called(ctx, hash, forUpdate)
	return getOrNil[*entity.RefreshToken](args, 0), args.Error(1)
}

func (m *RefreshTokenRepo) Revoke(ctx context.Context, id string, replacedBy *string, at time.Time) error {
	return m.Called(ctx, id, replacedBy, at).Error(0)
}

func (m *RefreshTokenRepo) RevokeFamily(ctx context.Context, familyID string, at time.Time) error {
	return m.Called(ctx, familyID, at).Error(0)
}

func (m *RefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string, at time.Time) error {
	return m.Called(ctx, userID, at).Error(0)
}

func (m *RefreshTokenRepo) DeleteStale(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

type UserTokenRepo struct{ mock.Mock }

func (m *UserTokenRepo) Create(ctx context.Context, t *entity.UserToken) error {
	return m.Called(ctx, t).Error(0)
}

func (m *UserTokenRepo) FindByHash(ctx context.Context, purpose entity.TokenPurpose, hash string) (*entity.UserToken, error) {
	args := m.Called(ctx, purpose, hash)
	return getOrNil[*entity.UserToken](args, 0), args.Error(1)
}

func (m *UserTokenRepo) MarkUsed(ctx context.Context, id string, at time.Time) error {
	return m.Called(ctx, id, at).Error(0)
}

func (m *UserTokenRepo) InvalidateAll(ctx context.Context, userID string, purpose entity.TokenPurpose, at time.Time) error {
	return m.Called(ctx, userID, purpose, at).Error(0)
}

func (m *UserTokenRepo) LastCreatedAt(ctx context.Context, userID string, purpose entity.TokenPurpose) (time.Time, bool, error) {
	args := m.Called(ctx, userID, purpose)
	return args.Get(0).(time.Time), args.Bool(1), args.Error(2)
}

func (m *UserTokenRepo) DeleteStale(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

// ---------------------------------------------------------------- security

type Hasher struct{ mock.Mock }

func (m *Hasher) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *Hasher) Compare(hash, password string) error {
	return m.Called(hash, password).Error(0)
}

type AccessTokenProvider struct{ mock.Mock }

func (m *AccessTokenProvider) Issue(userID string, role entity.Role, sessionID string) (string, time.Time, error) {
	args := m.Called(userID, role, sessionID)
	return args.String(0), args.Get(1).(time.Time), args.Error(2)
}

func (m *AccessTokenProvider) Parse(token string) (*model.TokenClaims, error) {
	args := m.Called(token)
	return getOrNil[*model.TokenClaims](args, 0), args.Error(1)
}

// FakeTx runs the function directly, without a real transaction.
// Count lets tests assert that a transaction boundary was used.
type FakeTx struct{ Count int }

func (f *FakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.Count++
	return fn(ctx)
}

// ---------------------------------------------------------------- email

type EmailSender struct{ mock.Mock }

func (m *EmailSender) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	return m.Called(ctx, to, subject, htmlBody).Error(0)
}

type AuthMailer struct{ mock.Mock }

func (m *AuthMailer) SendEmailVerification(ctx context.Context, to, name, link string, validFor time.Duration) error {
	return m.Called(ctx, to, name, link, validFor).Error(0)
}

func (m *AuthMailer) SendPasswordReset(ctx context.Context, to, name, link string, validFor time.Duration) error {
	return m.Called(ctx, to, name, link, validFor).Error(0)
}

// ---------------------------------------------------------------- delivery

type Validator struct{ mock.Mock }

func (m *Validator) Validate(ctx context.Context, data any) error {
	return m.Called(ctx, data).Error(0)
}

// ---------------------------------------------------------------- services

type AuthService struct{ mock.Mock }

func (m *AuthService) Register(ctx context.Context, in model.RegisterInput) (*model.UserProfile, error) {
	args := m.Called(ctx, in)
	return getOrNil[*model.UserProfile](args, 0), args.Error(1)
}

func (m *AuthService) Login(ctx context.Context, in model.LoginInput) (*model.AuthTokens, error) {
	args := m.Called(ctx, in)
	return getOrNil[*model.AuthTokens](args, 0), args.Error(1)
}

func (m *AuthService) Refresh(ctx context.Context, refreshToken string, client model.ClientInfo) (*model.AuthTokens, error) {
	args := m.Called(ctx, refreshToken, client)
	return getOrNil[*model.AuthTokens](args, 0), args.Error(1)
}

func (m *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return m.Called(ctx, refreshToken).Error(0)
}

func (m *AuthService) LogoutAll(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *AuthService) ChangePassword(ctx context.Context, in model.ChangePasswordInput) error {
	return m.Called(ctx, in).Error(0)
}

func (m *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	return m.Called(ctx, email).Error(0)
}

func (m *AuthService) ResetPassword(ctx context.Context, in model.ResetPasswordInput) error {
	return m.Called(ctx, in).Error(0)
}

func (m *AuthService) VerifyEmail(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func (m *AuthService) ResendVerification(ctx context.Context, email string) error {
	return m.Called(ctx, email).Error(0)
}

func (m *AuthService) CreateAdmin(ctx context.Context, in model.RegisterInput) (*model.UserProfile, error) {
	args := m.Called(ctx, in)
	return getOrNil[*model.UserProfile](args, 0), args.Error(1)
}

type UserService struct{ mock.Mock }

func (m *UserService) GetProfile(ctx context.Context, userID string) (*model.UserProfile, error) {
	args := m.Called(ctx, userID)
	return getOrNil[*model.UserProfile](args, 0), args.Error(1)
}

func (m *UserService) List(ctx context.Context, page model.PageRequest) (*model.Page[model.UserProfile], error) {
	args := m.Called(ctx, page)
	return getOrNil[*model.Page[model.UserProfile]](args, 0), args.Error(1)
}
