package service

import (
	"context"

	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/core/port"
)

// UserService handles user related use cases.
type UserService interface {
	GetProfile(ctx context.Context, userID string) (*model.UserProfile, error)
	List(ctx context.Context, page model.PageRequest) (*model.Page[model.UserProfile], error)
}

type userService struct {
	users port.UserRepository
}

func NewUserService(users port.UserRepository) UserService {
	return &userService{users: users}
}

func (s *userService) GetProfile(ctx context.Context, userID string) (*model.UserProfile, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	profile := model.NewUserProfile(user)
	return &profile, nil
}

func (s *userService) List(ctx context.Context, req model.PageRequest) (*model.Page[model.UserProfile], error) {
	req = req.Normalize()

	users, total, err := s.users.List(ctx, req)
	if err != nil {
		return nil, err
	}

	page := model.MapPage(model.NewPage(users, total, req), func(u entity.User) model.UserProfile {
		return model.NewUserProfile(&u)
	})
	return &page, nil
}
