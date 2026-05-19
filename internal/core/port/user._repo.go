package port

import "be-ayaka/internal/core/entity"

type UserRepository interface {
	FindByID(id string) (*entity.User, error)
	FindByEmail(email string) (*entity.User, error)
	Create(user *entity.User) error
}