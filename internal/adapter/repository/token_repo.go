package repository

import (
	"context"
	"time"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrTokenNotFound = apperror.NotFound("token not found")
	errTokenConflict = apperror.Conflict("token already exists")
)

// ---------------------------------------------------------------- refresh tokens

// RefreshTokenRepo implements port.RefreshTokenRepository with GORM.
type RefreshTokenRepo struct {
	db *gorm.DB
}

func NewRefreshTokenRepo(db *gorm.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, t *entity.RefreshToken) error {
	return translate(conn(ctx, r.db).Create(t).Error, ErrTokenNotFound, errTokenConflict)
}

func (r *RefreshTokenRepo) FindByHash(ctx context.Context, hash string, forUpdate bool) (*entity.RefreshToken, error) {
	q := conn(ctx, r.db)
	if forUpdate {
		q = q.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate})
	}
	var t entity.RefreshToken
	if err := q.Where("token_hash = ?", hash).First(&t).Error; err != nil {
		return nil, translate(err, ErrTokenNotFound, errTokenConflict)
	}
	return &t, nil
}

func (r *RefreshTokenRepo) Revoke(ctx context.Context, id string, replacedBy *string, at time.Time) error {
	err := conn(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]any{"revoked_at": at, "replaced_by": replacedBy}).Error
	return translate(err, ErrTokenNotFound, errTokenConflict)
}

func (r *RefreshTokenRepo) RevokeFamily(ctx context.Context, familyID string, at time.Time) error {
	return r.revokeWhere(ctx, at, "family_id = ?", familyID)
}

func (r *RefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string, at time.Time) error {
	return r.revokeWhere(ctx, at, "user_id = ?", userID)
}

func (r *RefreshTokenRepo) DeleteStale(ctx context.Context, before time.Time) (int64, error) {
	res := conn(ctx, r.db).
		Where("expires_at < ? OR revoked_at < ?", before, before).
		Delete(&entity.RefreshToken{})
	return res.RowsAffected, translate(res.Error, ErrTokenNotFound, errTokenConflict)
}

func (r *RefreshTokenRepo) revokeWhere(ctx context.Context, at time.Time, query string, args ...any) error {
	err := conn(ctx, r.db).Model(&entity.RefreshToken{}).
		Where(query, args...).
		Where("revoked_at IS NULL").
		Update("revoked_at", at).Error
	return translate(err, ErrTokenNotFound, errTokenConflict)
}

// ---------------------------------------------------------------- one-time user tokens

// UserTokenRepo implements port.UserTokenRepository with GORM.
type UserTokenRepo struct {
	db *gorm.DB
}

func NewUserTokenRepo(db *gorm.DB) *UserTokenRepo {
	return &UserTokenRepo{db: db}
}

func (r *UserTokenRepo) Create(ctx context.Context, t *entity.UserToken) error {
	return translate(conn(ctx, r.db).Create(t).Error, ErrTokenNotFound, errTokenConflict)
}

func (r *UserTokenRepo) FindByHash(ctx context.Context, purpose entity.TokenPurpose, hash string) (*entity.UserToken, error) {
	var t entity.UserToken
	err := conn(ctx, r.db).Where("purpose = ? AND token_hash = ?", purpose, hash).First(&t).Error
	if err != nil {
		return nil, translate(err, ErrTokenNotFound, errTokenConflict)
	}
	return &t, nil
}

func (r *UserTokenRepo) MarkUsed(ctx context.Context, id string, at time.Time) error {
	// The used_at IS NULL guard makes redemption atomic: of two concurrent
	// requests with the same token, only one updates a row.
	res := conn(ctx, r.db).Model(&entity.UserToken{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", at)
	if res.Error != nil {
		return translate(res.Error, ErrTokenNotFound, errTokenConflict)
	}
	if res.RowsAffected == 0 {
		return ErrTokenNotFound
	}
	return nil
}

func (r *UserTokenRepo) InvalidateAll(ctx context.Context, userID string, purpose entity.TokenPurpose, at time.Time) error {
	err := conn(ctx, r.db).Model(&entity.UserToken{}).
		Where("user_id = ? AND purpose = ? AND used_at IS NULL", userID, purpose).
		Update("used_at", at).Error
	return translate(err, ErrTokenNotFound, errTokenConflict)
}

func (r *UserTokenRepo) LastCreatedAt(ctx context.Context, userID string, purpose entity.TokenPurpose) (time.Time, bool, error) {
	var t entity.UserToken
	err := conn(ctx, r.db).
		Where("user_id = ? AND purpose = ?", userID, purpose).
		Order("created_at DESC").
		First(&t).Error
	if err != nil {
		err = translate(err, ErrTokenNotFound, errTokenConflict)
		if apperror.As(err).Kind == apperror.KindNotFound {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return t.CreatedAt, true, nil
}

func (r *UserTokenRepo) DeleteStale(ctx context.Context, before time.Time) (int64, error) {
	res := conn(ctx, r.db).
		Where("expires_at < ? OR used_at < ?", before, before).
		Delete(&entity.UserToken{})
	return res.RowsAffected, translate(res.Error, ErrTokenNotFound, errTokenConflict)
}
