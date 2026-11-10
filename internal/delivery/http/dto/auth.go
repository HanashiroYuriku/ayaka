package dto

import (
	"strings"
	"time"

	"be-ayaka/internal/core/model"
)

type RegisterRequest struct {
	Username    string `json:"username" validate:"required,min=3,max=20,username,unique=users->username" example:"hanashiroyuriku"`
	Email       string `json:"email" validate:"required,max=100,email,unique=users->email" example:"hanashiroyuriku@gmail.com"`
	DisplayName string `json:"displayName" validate:"omitempty,max=100" example:"Hanashiro Yuriku"`
	Password    string `json:"password" validate:"required,complexpassword" example:"P4$$word"`
}

// Normalize is called before validation so that rules (e.g. unique) see the
// same value that will be stored.
func (r *RegisterRequest) Normalize() {
	r.Username = strings.TrimSpace(r.Username)
	r.Email = normalizeEmail(r.Email)
	r.DisplayName = strings.TrimSpace(r.DisplayName)
}

func (r RegisterRequest) ToInput() model.RegisterInput {
	return model.RegisterInput{
		Username:    r.Username,
		Email:       r.Email,
		DisplayName: r.DisplayName,
		Password:    r.Password,
	}
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"hanashiroyuriku@gmail.com"`
	Password string `json:"password" validate:"required,maxbytes=72" example:"P4$$word"`
}

func (r *LoginRequest) Normalize() { r.Email = normalizeEmail(r.Email) }

func (r LoginRequest) ToInput(client model.ClientInfo) model.LoginInput {
	return model.LoginInput{Email: r.Email, Password: r.Password, Client: client}
}

// RefreshTokenRequest is used by /auth/refresh and /auth/logout.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required,max=128"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required,maxbytes=72"`
	NewPassword     string `json:"newPassword" validate:"required,complexpassword" example:"N3w$ecret"`
}

type EmailRequest struct {
	Email string `json:"email" validate:"required,email,max=100" example:"hanashiroyuriku@gmail.com"`
}

func (r *EmailRequest) Normalize() { r.Email = normalizeEmail(r.Email) }

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required,max=128"`
	NewPassword string `json:"newPassword" validate:"required,complexpassword" example:"N3w$ecret"`
}

type TokenRequest struct {
	Token string `json:"token" validate:"required,max=128"`
}

type TokenResponse struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	TokenType        string    `json:"tokenType" example:"Bearer"`
	ExpiresIn        int64     `json:"expiresIn" example:"900"` // seconds until the access token expires
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
}

func NewTokenResponse(t *model.AuthTokens) TokenResponse {
	return TokenResponse{
		AccessToken:      t.AccessToken,
		RefreshToken:     t.RefreshToken,
		TokenType:        t.TokenType,
		ExpiresIn:        int64(time.Until(t.AccessExpiresAt).Seconds()),
		AccessExpiresAt:  t.AccessExpiresAt.UTC(),
		RefreshExpiresAt: t.RefreshExpiresAt.UTC(),
	}
}

func normalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
