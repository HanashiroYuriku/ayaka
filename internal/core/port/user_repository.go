package port

import (
	"context"

	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
)

// UserRepository is the persistence contract for users.
//
// Implementations must return apperror.ErrNotFound (or an error matching it)
// when a record does not exist, and apperror.ErrConflict on unique violations.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	List(ctx context.Context, page model.PageRequest) ([]entity.User, int64, error)
	Create(ctx context.Context, user *entity.User) error
}
