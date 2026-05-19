package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Log      LogConfig      `mapstructure:"log"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JwtConfig      `mapstructure:"jwt"`
	Frontend FrontendConfig `mapstructure:"frontend"`
	Email    EmailConfig    `mapstructure:"email"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type LogConfig struct {
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"pass"`
	Name     string `mapstructure:"name"`
}

type JwtConfig struct {
	Secret  string `mapstructure:"secret"`
	Expired int    `mapstructure:"expired"` // minutes
	RefreshSecret  string `mapstructure:"refresh_secret"`
	RefreshExpired int    `mapstructure:"refresh_expired"` // minutes
}

type FrontendConfig struct {
	URL string `mapstructure:"url"`
}

type EmailConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
	User string `mapstructure:"user"`
	Pass string `mapstructure:"pass"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	// load .env file
	_ = godotenv.Load(".env")

	// create new viper instance
	v := viper.New()

	// read config from file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	for _, k := range v.AllKeys() {
		value := v.GetString(k)
		// Cek apakah value memiliki format ${ENV_VAR}
		if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
			// Ekstrak nama environment variable dari value
			envQuery := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")

			// Ganti teks di Viper dengan nilai asli dari OS/Env
			v.Set(k, getEnvOrPanic(envQuery))
		}
	}

	// unmarshal config to struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}

// getEnvOrPanic is a helper function to get environment variable or panic if not found
func getEnvOrPanic(env string) string {
	// Split env by ":" to separate env name and default value (if any)
	split := strings.SplitN(env, ":", 2)

	envName := split[0]
	res := os.Getenv(envName) // Cari di sistem

	// if not found in system, check if there's a default value in the yaml (split[1])
	if len(res) == 0 {
		if len(split) > 1 {
			res = split[1]

			// remove surrounding quotes if any (untuk kasus default value yang berupa string dengan spasi, exmp: "default value")
			res = strings.Trim(res, "\"")
		}

		// if still empty (and not a password since passwords can be empty)
		if len(res) == 0 && !strings.Contains(strings.ToLower(envName), "pass") {
			panic(fmt.Sprintf("Environment Variable Not Found: %s", envName))
		}
	}
	return res
}
