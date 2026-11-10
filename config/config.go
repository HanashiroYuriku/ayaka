// Package config loads application configuration from a YAML file whose values
// may reference environment variables using the ${ENV_NAME:default} syntax.
package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"

	// minSecretLength is the minimum JWT secret length enforced outside development.
	minSecretLength = 32
)

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Server    ServerConfig    `mapstructure:"server"`
	Log       LogConfig       `mapstructure:"log"`
	Database  DatabaseConfig  `mapstructure:"database"`
	JWT       JwtConfig       `mapstructure:"jwt"`
	Auth      AuthConfig      `mapstructure:"auth"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Frontend  FrontendConfig  `mapstructure:"frontend"`
	Email     EmailConfig     `mapstructure:"email"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"`
}

// IsProduction reports whether the application runs in the production environment.
func (a AppConfig) IsProduction() bool { return a.Env == EnvProduction }

// IsDevelopment reports whether the application runs in the development environment.
func (a AppConfig) IsDevelopment() bool { return a.Env == EnvDevelopment }

type ServerConfig struct {
	Port               int           `mapstructure:"port"`
	ReadTimeout        time.Duration `mapstructure:"read_timeout"`
	WriteTimeout       time.Duration `mapstructure:"write_timeout"`
	IdleTimeout        time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout    time.Duration `mapstructure:"shutdown_timeout"`
	BodyLimitMB        int           `mapstructure:"body_limit_mb"`
	CORSAllowedOrigins []string      `mapstructure:"cors_allowed_origins"`
	EnableSwagger      bool          `mapstructure:"enable_swagger"`
	// TrustedProxies enables reading the client IP from X-Forwarded-For when the
	// request comes from one of these addresses (e.g. a load balancer).
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug | info | warn | error
	Format string `mapstructure:"format"` // json | text
	// FilePath is optional. When empty, logs are written to stdout only (12-factor).
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"pass"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"`
	TimeZone        string        `mapstructure:"timezone"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	SlowQuery       time.Duration `mapstructure:"slow_query_threshold"`
}

// DSN builds the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode, d.TimeZone)
}

type JwtConfig struct {
	Secret  string        `mapstructure:"secret"`
	Expired time.Duration `mapstructure:"expired"` // access token lifetime
}

// AuthConfig controls sessions and account security.
type AuthConfig struct {
	// RefreshTTL is the lifetime of a refresh token (sliding: renewed on every refresh).
	RefreshTTL               time.Duration `mapstructure:"refresh_ttl"`
	RequireEmailVerification bool          `mapstructure:"require_email_verification"`
	MaxFailedLogins          int           `mapstructure:"max_failed_logins"`
	LockoutDuration          time.Duration `mapstructure:"lockout_duration"`
	PasswordResetTTL         time.Duration `mapstructure:"password_reset_ttl"`
	EmailVerificationTTL     time.Duration `mapstructure:"email_verification_ttl"`
	// EmailCooldown is the minimum time between two emails of the same kind to one user.
	EmailCooldown time.Duration `mapstructure:"email_cooldown"`
}

type RateLimitConfig struct {
	// Auth limits requests per client IP on authentication endpoints.
	AuthMax    int           `mapstructure:"auth_max"`
	AuthWindow time.Duration `mapstructure:"auth_window"`
}

type FrontendConfig struct {
	URL string `mapstructure:"url"`
	// Paths of the frontend pages that receive ?token=... from emails.
	ResetPasswordPath string `mapstructure:"reset_password_path"`
	VerifyEmailPath   string `mapstructure:"verify_email_path"`
}

type EmailConfig struct {
	// Driver is "smtp" (real delivery) or "log" (emails are written to the log; development only).
	Driver string `mapstructure:"driver"`
	Host   string `mapstructure:"host"`
	Port   string `mapstructure:"port"`
	User   string `mapstructure:"user"`
	Pass   string `mapstructure:"pass"`
	From   string `mapstructure:"from"`
}

// envPattern matches ${NAME} or ${NAME:default}.
var envPattern = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)(?::(.*))?\}$`)

// Load reads the configuration file at path (defaults to ./config.yaml),
// resolves environment variable placeholders and validates the result.
func Load(path string) (*Config, error) {
	// .env is optional; real environment variables always take precedence.
	_ = godotenv.Load(".env")

	if path == "" {
		path = "config.yaml"
	}

	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var missing []string
	for _, key := range v.AllKeys() {
		raw := v.GetString(key)
		m := envPattern.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		name, def, hasDefault := m[1], m[2], strings.Contains(raw, ":")

		val, ok := os.LookupEnv(name)
		switch {
		case ok:
		case hasDefault:
			val = strings.Trim(def, `"'`)
		default:
			missing = append(missing, name)
			continue
		}
		v.Set(key, val)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Server.CORSAllowedOrigins = splitList(cfg.Server.CORSAllowedOrigins)
	cfg.Server.TrustedProxies = splitList(cfg.Server.TrustedProxies)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks that the configuration is consistent and safe to run with.
func (c *Config) Validate() error {
	var errs []error

	switch c.App.Env {
	case EnvDevelopment, EnvStaging, EnvProduction:
	default:
		errs = append(errs, fmt.Errorf("app.env must be one of development|staging|production, got %q", c.App.Env))
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port))
	}

	if c.JWT.Secret == "" {
		errs = append(errs, errors.New("jwt.secret is required"))
	}
	positive := map[string]time.Duration{
		"jwt.expired":                 c.JWT.Expired,
		"auth.refresh_ttl":            c.Auth.RefreshTTL,
		"auth.lockout_duration":       c.Auth.LockoutDuration,
		"auth.password_reset_ttl":     c.Auth.PasswordResetTTL,
		"auth.email_verification_ttl": c.Auth.EmailVerificationTTL,
	}
	for name, d := range positive {
		if d <= 0 {
			errs = append(errs, fmt.Errorf("%s must be a positive duration", name))
		}
	}
	if c.Auth.MaxFailedLogins < 1 {
		errs = append(errs, errors.New("auth.max_failed_logins must be at least 1"))
	}

	switch c.Email.Driver {
	case "log":
	case "smtp":
		if c.Email.Host == "" || c.Email.Port == "" || (c.Email.From == "" && c.Email.User == "") {
			errs = append(errs, errors.New("email.host, email.port and email.from (or email.user) are required for the smtp driver"))
		}
	default:
		errs = append(errs, fmt.Errorf("email.driver must be smtp|log, got %q", c.Email.Driver))
	}

	if !c.App.IsDevelopment() {
		if len(c.JWT.Secret) < minSecretLength {
			errs = append(errs, fmt.Errorf("jwt.secret must be at least %d characters outside development", minSecretLength))
		}
		if c.App.IsProduction() && c.Email.Driver == "log" {
			errs = append(errs, errors.New("email.driver=log is not allowed in production (it would write reset links to the logs)"))
		}
		for _, o := range c.Server.CORSAllowedOrigins {
			if o == "*" {
				errs = append(errs, errors.New("server.cors_allowed_origins must not contain '*' outside development"))
			}
		}
	}

	return errors.Join(errs...)
}

// splitList normalises list values that may come from a single comma-separated env var.
func splitList(in []string) []string {
	var out []string
	for _, item := range in {
		for _, part := range strings.Split(item, ",") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}
