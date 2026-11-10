package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"be-ayaka/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleYAML = `
app:
  name: ${APP_NAME:"Ayaka Test"}
  version: v0.0.1
  env: ${APP_ENV:development}
server:
  port: ${SERVER_PORT:8000}
  read_timeout: 5s
  cors_allowed_origins: ${CORS_ALLOWED_ORIGINS:"http://a.test, http://b.test"}
database:
  pass: ${DATABASE_PASS:""}
jwt:
  secret: ${JWT_ACCESS_SECRET}
  expired: ${JWT_ACCESS_EXPIRED:15m}
  refresh_secret: ${JWT_REFRESH_SECRET}
  refresh_expired: 168h
`

func writeConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAML), 0o600))
	return path
}

func TestLoad_InterpolatesEnvAndDefaults(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("JWT_ACCESS_SECRET", "access")
	t.Setenv("JWT_REFRESH_SECRET", "refresh")

	cfg, err := config.Load(writeConfig(t))
	require.NoError(t, err)

	assert.Equal(t, "Ayaka Test", cfg.App.Name)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, 5*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 15*time.Minute, cfg.JWT.Expired)
	assert.Equal(t, 168*time.Hour, cfg.JWT.RefreshExpired)
	assert.Equal(t, []string{"http://a.test", "http://b.test"}, cfg.Server.CORSAllowedOrigins)
	assert.Empty(t, cfg.Database.Password)
}

func TestLoad_MissingRequiredEnv(t *testing.T) {
	t.Setenv("JWT_REFRESH_SECRET", "refresh")
	os.Unsetenv("JWT_ACCESS_SECRET")

	_, err := config.Load(writeConfig(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_ACCESS_SECRET")
}

func TestLoad_ProductionRejectsWeakSecrets(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_ACCESS_SECRET", "short")
	t.Setenv("JWT_REFRESH_SECRET", "also-short")

	_, err := config.Load(writeConfig(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 32 characters")
}

func TestValidate_RejectsSameSecretsAndBadEnv(t *testing.T) {
	cfg := &config.Config{
		App:    config.AppConfig{Env: "prod"},
		Server: config.ServerConfig{Port: 8000},
		JWT:    config.JwtConfig{Secret: "same", RefreshSecret: "same", Expired: time.Minute, RefreshExpired: time.Hour},
	}

	err := cfg.Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.env")
	assert.Contains(t, err.Error(), "must be different")
}
