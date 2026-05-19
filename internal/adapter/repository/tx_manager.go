package repository

import (
	"context"

	"be-ayaka/internal/core/port"
	"gorm.io/gorm"
)

type TxKey struct{}

type gormTxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) port.TxManager {
	return &gormTxManager{db: db}
}

func (tm *gormTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, TxKey{}, tx)
		return fn(txCtx)		
	})
}

func ExtractTx(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(TxKey{}).(*gorm.DB); ok {
		return tx
	}
	return defaultDB
}
