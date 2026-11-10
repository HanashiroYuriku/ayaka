// Package model contains the input/output types of the core use cases.
// They are independent from transport (HTTP DTOs) and persistence (entities).
package model

import (
	"time"

	"be-ayaka/internal/core/entity"
)

type RegisterInput struct {
	Username    string
	Email       string
	DisplayName string
	Password    string
}

type LoginInput struct {
	Email    string
	Password string
}

type UserProfile struct {
	ID          string
	Username    string
	Email       string
	DisplayName string
	Role        entity.Role
	CreatedAt   time.Time
}

// NewUserProfile maps a user entity to its public representation.
func NewUserProfile(u *entity.User) UserProfile {
	return UserProfile{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		CreatedAt:   u.CreatedAt,
	}
}
