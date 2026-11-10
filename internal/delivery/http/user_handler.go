package http

import (
	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/service"
	"be-ayaka/internal/delivery/http/dto"
	"be-ayaka/internal/middleware"
	"be-ayaka/pkg/response"
	"be-ayaka/pkg/validator"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	users     service.UserService
	validator validator.Validator
}

func NewUserHandler(users service.UserService, v validator.Validator) *UserHandler {
	return &UserHandler{users: users, validator: v}
}

// GetMe returns the profile of the authenticated user.
//
//	@Summary		Current user
//	@Description	Get the profile of the authenticated user.
//	@Tags			Users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response{data=dto.UserResponse}
//	@Failure		401	{object}	response.Response	"Missing or invalid token"
//	@Failure		404	{object}	response.Response	"User not found"
//	@Router			/api/v1/users/me [get]
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	userID, ok := middleware.UserID(c)
	if !ok {
		return apperror.ErrUnauthorized
	}

	profile, err := h.users.GetProfile(c.UserContext(), userID)
	if err != nil {
		return err
	}

	return c.JSON(response.Success("profile retrieved", dto.NewUserResponse(*profile), middleware.RequestID(c)))
}

// List returns a paginated list of users (admin only).
//
//	@Summary		List users
//	@Description	Paginated list of users, newest first. Requires the admin role.
//	@Tags			Users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number"	minimum(1)	default(1)
//	@Param			pageSize	query		int	false	"Page size"		minimum(1)	maximum(100)	default(20)
//	@Success		200			{object}	response.Response{data=[]dto.UserResponse,meta=dto.PaginationMeta}
//	@Failure		401			{object}	response.Response	"Missing or invalid token"
//	@Failure		403			{object}	response.Response	"Not an admin"
//	@Failure		422			{object}	response.Response	"Invalid pagination parameters"
//	@Router			/api/v1/users [get]
func (h *UserHandler) List(c *fiber.Ctx) error {
	var q dto.PaginationQuery
	if err := bindQuery(c, h.validator, &q); err != nil {
		return err
	}

	page, err := h.users.List(c.UserContext(), q.ToPageRequest())
	if err != nil {
		return err
	}

	items, meta := dto.NewPaginated(page, dto.NewUserResponse)
	return c.JSON(response.SuccessWithMeta("users retrieved", items, meta, middleware.RequestID(c)))
}
