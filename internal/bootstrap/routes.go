package bootstrap

import (
	"be-ayaka/config"
	"be-ayaka/internal/delivery/http"
	"be-ayaka/internal/middleware"
	"be-ayaka/pkg/generateid"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"gorm.io/gorm"
	"github.com/gofiber/swagger"
)

func SetupRoutes(app *fiber.App, cfg *config.Config, db *gorm.DB) {
	// middleware request
	app.Use(requestid.New(requestid.Config{
		Header:     fiber.HeaderXRequestID,
		ContextKey: "requestid",
		Generator: func() string {
			return generateid.GenerateID("REQUEST")
		},
	}))

	// === Swagger UI
	app.Get("/swagger/*", swagger.HandlerDefault)

	// === Health Check
	healthHandler := http.NewHealthCheckHandler(cfg, db)
	// --- health route
	app.Get("/health", healthHandler.Check)
	// ---

	// === Handler ===
	handler := BuildAllDependencies(db, cfg)

	// === api app group
	apiApp := app.Group("/api")

	// === version 1
	apiV1 := apiApp.Group("/v1")

	// === auth group
	authGroup := apiV1.Group("/", middleware.RequireAuth(cfg))

	// === User and Admin routes
	userGroup := authGroup.Group("/user", middleware.OnlyRole("user", "admin"))

	userGroup.Get("/profile", handler.User.GetProfile)
}
