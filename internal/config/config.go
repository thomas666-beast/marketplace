package config

import (
    "fmt"
    "os"
    "strconv"

    "github.com/joho/godotenv"
)

type Config struct {
    AppEnv        string
    AppPort       string
    PostgresHost  string
    PostgresPort  string
    PostgresUser  string
    PostgresPass  string
    PostgresDB    string
    RedisHost     string
    RedisPort     string
    RedisPass     string
    DefaultLocale string
    JWTSecret string
}

func Load() (*Config, error) {
    _ = godotenv.Load()

    cfg := &Config{
        AppEnv:        getEnv("APP_ENV", "development"),
        AppPort:       getEnv("APP_PORT", "8080"),
        PostgresHost:  getEnv("POSTGRES_HOST", "localhost"),
        PostgresPort:  getEnv("POSTGRES_PORT", "5432"),
        PostgresUser:  getEnv("POSTGRES_USER", "marketplace"),
        PostgresPass:  getEnv("POSTGRES_PASSWORD", ""),
        PostgresDB:    getEnv("POSTGRES_DB", "marketplace"),
        RedisHost:     getEnv("REDIS_HOST", "localhost"),
        RedisPort:     getEnv("REDIS_PORT", "6379"),
        RedisPass:     getEnv("REDIS_PASSWORD", ""),
        DefaultLocale: getEnv("DEFAULT_LOCALE", "en"),
        JWTSecret: getEnv("JWT_SECRET", ""),
    }

    if cfg.PostgresPass == "" {
        return nil, fmt.Errorf("POSTGRES_PASSWORD is required")
    }

    if len(cfg.JWTSecret) < 32 {
        return nil, fmt.Errorf("JWT_SECRET must be at least 32 bytes")
    }
    return cfg, nil
}

func (c *Config) PostgresDSN() string {
    return fmt.Sprintf(
        "postgres://%s:%s@%s:%s/%s?sslmode=disable",
        c.PostgresUser, c.PostgresPass, c.PostgresHost, c.PostgresPort, c.PostgresDB,
    )
}

func getEnv(key, fallback string) string {
    if v, ok := os.LookupEnv(key); ok && v != "" {
        return v
    }
    return fallback
}

// helper kept for future numeric config values
func getEnvInt(key string, fallback int) int {
    if v, ok := os.LookupEnv(key); ok {
        if n, err := strconv.Atoi(v); err == nil {
            return n
        }
    }
    return fallback
}
