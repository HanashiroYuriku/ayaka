package repository

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// TxManager implements port.TxManager with GORM transactions.
// Repositories pick the transaction up from the context via conn().
type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

// WithTx runs fn inside a transaction. Nested calls reuse the outer transaction.
func (tm *TxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// ContextWithTx stores tx in ctx. Mainly useful for tests.
func ContextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// conn returns the transaction stored in ctx, or db when there is none,
// already bound to ctx.
func conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}
