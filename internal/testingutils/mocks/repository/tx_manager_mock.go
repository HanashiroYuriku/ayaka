package mocks

import (
	"context"
	"github.com/stretchr/testify/mock"
)

type MockTxManager struct {
	mock.Mock
}

func (m *MockTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	err := fn(ctx)

	if err != nil {
		return err
	}

	return args.Error(0)
}
