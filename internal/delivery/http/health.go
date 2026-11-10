package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Pinger is satisfied by *sql.DB (and anything else worth checking).
type Pinger interface {
	PingContext(ctx context.Context) error
}

type HealthHandler struct {
	name    string
	version string
	checks  map[string]Pinger
}

func NewHealthHandler(name, version string, checks map[string]Pinger) *HealthHandler {
	return &HealthHandler{name: name, version: version, checks: checks}
}

type HealthResponse struct {
	Name       string            `json:"name" example:"Ayaka Backend"`
	Version    string            `json:"version" example:"v1.0.0"`
	Status     string            `json:"status" example:"UP"`
	Time       time.Time         `json:"time"`
	Components map[string]string `json:"components,omitempty"`
}

// Live reports that the process is running (liveness probe). It never touches
// dependencies, so a slow database does not get the container restarted.
//
//	@Summary	Liveness probe
//	@Tags		System
//	@Produce	json
//	@Success	200	{object}	HealthResponse
//	@Router		/livez [get]
func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return c.JSON(HealthResponse{Name: h.name, Version: h.version, Status: "UP", Time: time.Now().UTC()})
}

// Ready reports whether the service can serve traffic (readiness probe).
//
//	@Summary	Readiness probe
//	@Tags		System
//	@Produce	json
//	@Success	200	{object}	HealthResponse
//	@Failure	503	{object}	HealthResponse
//	@Router		/readyz [get]
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	res := HealthResponse{
		Name: h.name, Version: h.version, Status: "UP", Time: time.Now().UTC(),
		Components: make(map[string]string, len(h.checks)),
	}
	code := fiber.StatusOK
	for name, p := range h.checks {
		if err := p.PingContext(ctx); err != nil {
			res.Components[name] = "DOWN"
			res.Status = "DOWN"
			code = fiber.StatusServiceUnavailable
			continue
		}
		res.Components[name] = "UP"
	}
	return c.Status(code).JSON(res)
}
