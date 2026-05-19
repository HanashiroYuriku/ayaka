package mocks

import (
	"be-ayaka/config"
	"be-ayaka/pkg/jwt"

	"github.com/stretchr/testify/mock"
)

// mock hash service
type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateToken(cfg *config.Config, userID string, role string) (*jwt.TokenPair, error) {
	args := m.Called(cfg, userID, role)
	if args.Get(0) != nil {
		return args.Get(0).(*jwt.TokenPair), args.Error(1)
	}
	return nil, args.Error(1)
}
