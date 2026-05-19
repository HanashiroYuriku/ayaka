package mocks

import "github.com/stretchr/testify/mock"

type MockEmail struct {
	mock.Mock
}

func (m *MockEmail) SendEmail(to, subject, htmlBody string) error {
	args := m.Called(to, subject, htmlBody)
	return args.Error(0)
}
