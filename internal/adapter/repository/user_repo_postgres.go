package repository

import (
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/port"
	"errors"

	"gorm.io/gorm"
)

type userRepoPostgres struct {
	db *gorm.DB
}

func NewUserRepoPostgres(db *gorm.DB) port.UserRepository {
	return &userRepoPostgres{
		db: db,
	}
}

func (r *userRepoPostgres) FindByID(id string) (*entity.User, error) {
	var user entity.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepoPostgres) FindByEmail(email string) (*entity.User, error) {
	var user entity.User
	err := r.db.Where("email = ?" , email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepoPostgres) Create(user *entity.User) error {
	return r.db.Create(user).Error
}