package repository

import (
	"context"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound = apperror.NotFound("user not found")
	ErrUserConflict = apperror.Conflict("username or email already exists")
)

// UserRepo implements port.UserRepository with GORM.
type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*entity.User, error) {
	var user entity.User
	if err := conn(ctx, r.db).First(&user, "id = ?", id).Error; err != nil {
		return nil, translate(err, ErrUserNotFound, ErrUserConflict)
	}
	return &user, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	if err := conn(ctx, r.db).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, translate(err, ErrUserNotFound, ErrUserConflict)
	}
	return &user, nil
}

func (r *UserRepo) List(ctx context.Context, page model.PageRequest) ([]entity.User, int64, error) {
	var (
		users []entity.User
		total int64
	)
	// Session() makes the base query safely reusable for both statements.
	q := conn(ctx, r.db).Model(&entity.User{}).Session(&gorm.Session{})

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, translate(err, ErrUserNotFound, ErrUserConflict)
	}
	if total == 0 {
		return []entity.User{}, 0, nil
	}

	err := q.Order("created_at DESC, id DESC").
		Offset(page.Offset()).
		Limit(page.Limit()).
		Find(&users).Error
	if err != nil {
		return nil, 0, translate(err, ErrUserNotFound, ErrUserConflict)
	}
	return users, total, nil
}

func (r *UserRepo) Create(ctx context.Context, user *entity.User) error {
	return translate(conn(ctx, r.db).Create(user).Error, ErrUserNotFound, ErrUserConflict)
}
