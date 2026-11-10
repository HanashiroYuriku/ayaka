package repository

import (
	"context"
	"time"

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
		return nil, r.translate(err)
	}
	return &user, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	if err := conn(ctx, r.db).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, r.translate(err)
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
		return nil, 0, r.translate(err)
	}
	if total == 0 {
		return []entity.User{}, 0, nil
	}

	err := q.Order("created_at DESC, id DESC").
		Offset(page.Offset()).
		Limit(page.Limit()).
		Find(&users).Error
	if err != nil {
		return nil, 0, r.translate(err)
	}
	return users, total, nil
}

func (r *UserRepo) Create(ctx context.Context, user *entity.User) error {
	return r.translate(conn(ctx, r.db).Create(user).Error)
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	return r.updateOne(ctx, id, map[string]any{
		"password":              passwordHash,
		"failed_login_attempts": 0,
		"locked_until":          nil,
	})
}

func (r *UserRepo) MarkEmailVerified(ctx context.Context, id string, at time.Time) error {
	return r.updateOne(ctx, id, map[string]any{"email_verified_at": at})
}

func (r *UserRepo) RegisterFailedLogin(ctx context.Context, id string, maxAttempts int, lockUntil time.Time) error {
	// Single atomic UPDATE: safe under concurrent failed logins.
	return r.updateOne(ctx, id, map[string]any{
		"failed_login_attempts": gorm.Expr(
			"CASE WHEN failed_login_attempts + 1 >= ? THEN 0 ELSE failed_login_attempts + 1 END", maxAttempts),
		"locked_until": gorm.Expr(
			"CASE WHEN failed_login_attempts + 1 >= ? THEN ? ELSE locked_until END", maxAttempts, lockUntil),
	})
}

func (r *UserRepo) ResetFailedLogins(ctx context.Context, id string) error {
	return r.updateOne(ctx, id, map[string]any{"failed_login_attempts": 0, "locked_until": nil})
}

// updateOne updates columns of one user and reports ErrUserNotFound when no row matched.
func (r *UserRepo) updateOne(ctx context.Context, id string, values map[string]any) error {
	res := conn(ctx, r.db).Model(&entity.User{}).Where("id = ?", id).Updates(values)
	if res.Error != nil {
		return r.translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) translate(err error) error {
	return translate(err, ErrUserNotFound, ErrUserConflict)
}
