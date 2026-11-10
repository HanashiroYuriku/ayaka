package repository

import (
	"errors"

	"be-ayaka/internal/core/apperror"

	"gorm.io/gorm"
)

// translate converts GORM/driver errors into application errors.
// It relies on gorm.Config{TranslateError: true}.
func translate(err error, notFound *apperror.Error, conflict *apperror.Error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return notFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return conflict.Wrap(err)
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return apperror.Conflict("related resource constraint violated").Wrap(err)
	default:
		return apperror.Internal(err)
	}
}
