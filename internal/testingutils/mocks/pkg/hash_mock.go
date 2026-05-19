package mocks

import (
	"github.com/stretchr/testify/mock"
)

// mock hash service
type MockHashService struct {
	mock.Mock
}

func (m *MockHashService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockHashService) ComparePassword(password, hash string) error {
	args := m.Called(password, hash)
	return args.Error(0)
}
