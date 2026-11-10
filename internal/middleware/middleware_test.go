package middleware_test

import (
	"errors"
	"testing"
	"time"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/middleware"
	"be-ayaka/internal/testingutils"
	"be-ayaka/internal/testingutils/mocks"
	"be-ayaka/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ok(c *fiber.Ctx) error { return c.SendString("ok") }

// ============================================================ RequireAuth / RequireRole

func newAuthApp(tokens *mocks.AccessTokenProvider, roles ...entity.Role) *fiber.App {
	app := testingutils.NewTestApp()
	chain := []fiber.Handler{middleware.RequireAuth(tokens)}
	if len(roles) > 0 {
		chain = append(chain, middleware.RequireRole(roles...))
	}
	chain = append(chain, func(c *fiber.Ctx) error {
		id, _ := middleware.UserID(c)
		return c.SendString(id)
	})
	app.Get("/protected", chain...)
	return app
}

func TestRequireAuth(t *testing.T) {
	tokens := new(mocks.AccessTokenProvider)
	tokens.On("Parse", "good").
		Return(&model.TokenClaims{UserID: "USR-1", Role: entity.RoleUser}, nil)
	tokens.On("Parse", "expired").
		Return(nil, apperror.Unauthorized("token expired"))

	app := newAuthApp(tokens)

	tests := []struct {
		name   string
		header string
		status int
	}{
		{"missing header", "", fiber.StatusUnauthorized},
		{"wrong scheme", "Basic abc", fiber.StatusUnauthorized},
		{"empty token", "Bearer ", fiber.StatusUnauthorized},
		{"expired token", "Bearer expired", fiber.StatusUnauthorized},
		{"valid token", "Bearer good", fiber.StatusOK},
		{"scheme is case-insensitive", "bearer good", fiber.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testingutils.MakeJSONRequest(fiber.MethodGet, "/protected", nil)
			if tt.header != "" {
				req.Header.Set(fiber.HeaderAuthorization, tt.header)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}

func TestRequireRole(t *testing.T) {
	tokens := new(mocks.AccessTokenProvider)
	tokens.On("Parse", "user").Return(&model.TokenClaims{UserID: "U", Role: entity.RoleUser}, nil)
	tokens.On("Parse", "admin").Return(&model.TokenClaims{UserID: "A", Role: entity.RoleAdmin}, nil)

	app := newAuthApp(tokens, entity.RoleAdmin)

	req := testingutils.MakeJSONRequest(fiber.MethodGet, "/protected", nil, testingutils.WithAuth("user"))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	req = testingutils.MakeJSONRequest(fiber.MethodGet, "/protected", nil, testingutils.WithAuth("admin"))
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ============================================================ ErrorHandler / Recover

func TestErrorHandler_Mapping(t *testing.T) {
	tests := []struct {
		err    error
		status int
		code   response.Status
	}{
		{apperror.BadRequest("x"), fiber.StatusBadRequest, response.BadRequest},
		{apperror.Validation(map[string]string{"f": "bad"}), fiber.StatusUnprocessableEntity, response.UnprocessableEntity},
		{apperror.Unauthorized("x"), fiber.StatusUnauthorized, response.Unauthorized},
		{apperror.Forbidden("x"), fiber.StatusForbidden, response.Forbidden},
		{apperror.NotFound("x"), fiber.StatusNotFound, response.DataNotFound},
		{apperror.Conflict("x"), fiber.StatusConflict, response.DataConflict},
		{apperror.TooManyRequests("x"), fiber.StatusTooManyRequests, response.TooManyRequests},
		{errors.New("raw error"), fiber.StatusInternalServerError, response.InternalServerError},
		{fiber.ErrMethodNotAllowed, fiber.StatusMethodNotAllowed, response.MethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			app := testingutils.NewTestApp()
			app.Get("/", func(*fiber.Ctx) error { return tt.err })

			resp, err := app.Test(testingutils.MakeJSONRequest(fiber.MethodGet, "/", nil))
			require.NoError(t, err)

			assert.Equal(t, tt.status, resp.StatusCode)
			assert.Equal(t, tt.code.String(), testingutils.DecodeResponse(t, resp).Status)
		})
	}
}

func TestUnknownRoute_Returns404Envelope(t *testing.T) {
	app := testingutils.NewTestApp()

	resp, err := app.Test(testingutils.MakeJSONRequest(fiber.MethodGet, "/does-not-exist", nil))
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.Equal(t, response.DataNotFound.String(), testingutils.DecodeResponse(t, resp).Status)
}

func TestRecover_PanicBecomes500(t *testing.T) {
	app := testingutils.NewTestApp()
	app.Get("/panic", func(*fiber.Ctx) error { panic("kaboom") })

	resp, err := app.Test(testingutils.MakeJSONRequest(fiber.MethodGet, "/panic", nil))
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	body := testingutils.DecodeResponse(t, resp)
	assert.Equal(t, "internal server error", body.Message)
	assert.Equal(t, "TEST-REQUEST-ID", body.RequestID)
}

func TestRequestID_GeneratedWhenMissing(t *testing.T) {
	app := testingutils.NewTestApp()
	app.Get("/", ok)

	req := testingutils.MakeJSONRequest(fiber.MethodGet, "/", nil)
	req.Header.Del(fiber.HeaderXRequestID)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Regexp(t, `^REQ-[0-9a-f]{32}$`, resp.Header.Get(fiber.HeaderXRequestID))
}

func TestRateLimit(t *testing.T) {
	app := testingutils.NewTestApp()
	app.Get("/", middleware.RateLimit(2, time.Minute), ok)

	for i := 0; i < 2; i++ {
		resp, err := app.Test(testingutils.MakeJSONRequest(fiber.MethodGet, "/", nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)
	}

	resp, err := app.Test(testingutils.MakeJSONRequest(fiber.MethodGet, "/", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
}
