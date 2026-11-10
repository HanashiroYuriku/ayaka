package validator_test

import (
	"context"
	"strings"
	"testing"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/pkg/validator"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type registerPayload struct {
	Username string `json:"username" validate:"required,min=3,max=20,username"`
	Email    string `json:"email" validate:"required,email,unique=accounts->email"`
	Password string `json:"password" validate:"required,complexpassword"`
}

type account struct {
	ID        uint
	Email     string
	DeletedAt gorm.DeletedAt
}

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&account{}))
	return db
}

func TestValidate_Success(t *testing.T) {
	v := validator.New(newDB(t))

	err := v.Validate(context.Background(), &registerPayload{
		Username: "riku", Email: "riku@mail.com", Password: "P4$$word",
	})

	assert.NoError(t, err)
}

func TestValidate_FormatErrorsReturn422WithFields(t *testing.T) {
	v := validator.New(newDB(t))

	err := v.Validate(context.Background(), &registerPayload{
		Username: "a b", Email: "not-an-email", Password: "weak",
	})

	require.ErrorIs(t, err, apperror.ErrValidation)
	fields := apperror.As(err).Fields
	assert.Contains(t, fields, "username")
	assert.Contains(t, fields, "email")
	assert.Equal(t, "Password must be 8-64 characters (max 72 bytes) and contain uppercase, lowercase, number and special character", fields["password"])
}

func TestValidate_UniqueReturnsConflict_IgnoringSoftDeleted(t *testing.T) {
	db := newDB(t)
	v := validator.New(db)
	require.NoError(t, db.Create(&account{Email: "taken@mail.com"}).Error)
	deleted := &account{Email: "deleted@mail.com"}
	require.NoError(t, db.Create(deleted).Error)
	require.NoError(t, db.Delete(deleted).Error)

	err := v.Validate(context.Background(), &registerPayload{
		Username: "riku", Email: "taken@mail.com", Password: "P4$$word",
	})
	require.ErrorIs(t, err, apperror.ErrConflict)
	assert.Equal(t, "Email already exists", apperror.As(err).Fields["email"])

	err = v.Validate(context.Background(), &registerPayload{
		Username: "riku", Email: "deleted@mail.com", Password: "P4$$word",
	})
	assert.NoError(t, err)
}

// Regression: 64 characters can exceed bcrypt's 72-byte limit with multi-byte
// characters, which used to fail at hashing time with a 500.
func TestValidate_PasswordByteLimit(t *testing.T) {
	v := validator.New(newDB(t))
	multiByte := "Aa1!" + strings.Repeat("é", 60) // 64 characters, 124 bytes

	err := v.Validate(context.Background(), &registerPayload{
		Username: "riku", Email: "riku@mail.com", Password: multiByte,
	})
	require.ErrorIs(t, err, apperror.ErrValidation)
	assert.Contains(t, apperror.As(err).Fields, "password")

	type login struct {
		Password string `json:"password" validate:"required,maxbytes=72"`
	}
	err = v.Validate(context.Background(), &login{Password: multiByte})
	require.ErrorIs(t, err, apperror.ErrValidation)
	assert.Equal(t, "Password must be at most 72 bytes long", apperror.As(err).Fields["password"])

	assert.NoError(t, v.Validate(context.Background(), &login{Password: strings.Repeat("a", 72)}))
}
