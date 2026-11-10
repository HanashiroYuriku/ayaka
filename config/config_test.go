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
auth:
  refresh_ttl: 168h
  require_email_verification: true
  max_failed_logins: 5
  lockout_duration: 15m
  password_reset_ttl: 30m
  email_verification_ttl: 24h
  email_cooldown: 60s
email:
  driver: ${EMAIL_DRIVER:log}
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

	cfg, err := config.Load(writeConfig(t))
	require.NoError(t, err)

	assert.Equal(t, "Ayaka Test", cfg.App.Name)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, 5*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 15*time.Minute, cfg.JWT.Expired)
	assert.Equal(t, 168*time.Hour, cfg.Auth.RefreshTTL)
	assert.True(t, cfg.Auth.RequireEmailVerification)
	assert.Equal(t, "log", cfg.Email.Driver)
	assert.Equal(t, []string{"http://a.test", "http://b.test"}, cfg.Server.CORSAllowedOrigins)
	assert.Empty(t, cfg.Database.Password)
}

func TestLoad_MissingRequiredEnv(t *testing.T) {
	os.Unsetenv("JWT_ACCESS_SECRET")

	_, err := config.Load(writeConfig(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_ACCESS_SECRET")
}

func TestLoad_ProductionRejectsWeakSecretAndLogMailer(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_ACCESS_SECRET", "short")

	_, err := config.Load(writeConfig(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 32 characters")
	assert.Contains(t, err.Error(), "email.driver=log is not allowed in production")
}

func TestValidate_RejectsInvalidValues(t *testing.T) {
	cfg := &config.Config{
		App:    config.AppConfig{Env: "prod"},
		Server: config.ServerConfig{Port: 8000},
		JWT:    config.JwtConfig{Secret: "x", Expired: time.Minute},
		Auth:   config.AuthConfig{RefreshTTL: -time.Hour},
		Email:  config.EmailConfig{Driver: "smtp"},
	}

	err := cfg.Validate()

	require.Error(t, err)
	for _, want := range []string{"app.env", "auth.refresh_ttl", "auth.max_failed_logins", "smtp driver"} {
		assert.Contains(t, err.Error(), want)
	}
}
