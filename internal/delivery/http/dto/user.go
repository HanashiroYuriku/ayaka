// Package dto contains the HTTP request/response payloads and their mapping
// to and from the core models.
package dto

import (
	"time"

	"be-ayaka/internal/core/model"
)

type UserResponse struct {
	ID            string    `json:"id" example:"USR-01932c7e6f2a7c3b9f1d2e3a4b5c6d7e"`
	Username      string    `json:"username" example:"hanashiroyuriku"`
	Email         string    `json:"email" example:"hanashiroyuriku@gmail.com"`
	DisplayName   string    `json:"displayName" example:"Hanashiro Yuriku"`
	Role          string    `json:"role" example:"user"`
	EmailVerified bool      `json:"emailVerified" example:"false"`
	CreatedAt     time.Time `json:"createdAt"`
}

func NewUserResponse(p model.UserProfile) UserResponse {
	return UserResponse{
		ID:            p.ID,
		Username:      p.Username,
		Email:         p.Email,
		DisplayName:   p.DisplayName,
		Role:          p.Role.String(),
		EmailVerified: p.EmailVerified,
		CreatedAt:     p.CreatedAt.UTC(),
	}
}
