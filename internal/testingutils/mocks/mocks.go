// Package mocks contains testify mocks and fakes for the core ports and services.
//
// Each mock has a compile-time assertion so it breaks loudly when an interface changes.
package mocks

import (
	"context"

	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/core/port"
	"be-ayaka/internal/core/service"
	"be-ayaka/pkg/validator"

	"github.com/stretchr/testify/mock"
)

var (
	_ port.UserRepository = (*UserRepo)(nil)
	_ port.PasswordHasher = (*Hasher)(nil)
	_ port.TokenProvider  = (*TokenProvider)(nil)
	_ port.TxManager      = (*FakeTx)(nil)
	_ port.EmailSender    = (*EmailSender)(nil)
	_ validator.Validator = (*Validator)(nil)
	_ service.AuthService = (*AuthService)(nil)
	_ service.UserService = (*UserService)(nil)
)

// getOrNil returns args.Get(i) as T, or the zero value when it is nil.
func getOrNil[T any](args mock.Arguments, i int) T {
	var zero T
	if v := args.Get(i); v != nil {
		return v.(T)
	}
	return zero
}

// ---------------------------------------------------------------- ports

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

type Hasher struct{ mock.Mock }

func (m *Hasher) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *Hasher) Compare(hash, password string) error {
	return m.Called(hash, password).Error(0)
}

type TokenProvider struct{ mock.Mock }

func (m *TokenProvider) Issue(userID string, role entity.Role) (*model.AuthTokens, error) {
	args := m.Called(userID, role)
	return getOrNil[*model.AuthTokens](args, 0), args.Error(1)
}

func (m *TokenProvider) Parse(token string, typ model.TokenType) (*model.TokenClaims, error) {
	args := m.Called(token, typ)
	return getOrNil[*model.TokenClaims](args, 0), args.Error(1)
}

// FakeTx runs the function directly, without a real transaction.
// Count lets tests assert that a transaction boundary was used.
type FakeTx struct{ Count int }

func (f *FakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.Count++
	return fn(ctx)
}

type EmailSender struct{ mock.Mock }

func (m *EmailSender) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	return m.Called(ctx, to, subject, htmlBody).Error(0)
}

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

func (m *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.AuthTokens, error) {
	args := m.Called(ctx, refreshToken)
	return getOrNil[*model.AuthTokens](args, 0), args.Error(1)
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
