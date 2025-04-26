package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	LogPath           string
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	ServerPort        string
	Environment       string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	RedisHost         string
	RedisPort         int
	RedisPassword     string
	RedisDB           int
}

func LoadConfig() *Config {
	return &Config{
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "wallet_user"),
		DBPassword:        getEnv("DB_PASSWORD", "wallet_pass"),
		DBName:            getEnv("DB_NAME", "wallet_db"),
		DBSSLMode:         getEnv("DB_SSL_MODE", "disable"),
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		Environment:       getEnv("ENVIRONMENT", "development"),
		DBMaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME", 300)) * time.Second,
		RedisHost:         getEnv("REDIS_HOST", "localhost"),
		RedisPort:         getEnvAsInt("REDIS_PORT", 6379),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvAsInt("REDIS_DB", 0),
		LogPath:           "./logs/app.log",
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
