package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SecretKey          string
	DBName             string
	DBUser             string
	DBPass             string
	DBHost             string
	DBPort             string
	RedisURL           string
	Port               string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

var C Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	C = Config{
		SecretKey:          mustGet("SECRET_KEY"),
		DBName:             mustGet("DB_NAME"),
		DBUser:             mustGet("DB_USER"),
		DBPass:             mustGet("DB_PASS"),
		DBHost:             mustGet("DB_HOST"),
		DBPort:             getOrDefault("DB_PORT", "5432"),
		RedisURL:           getOrDefault("REDIS_URL", "redis://127.0.0.1:6379"),
		Port:               getOrDefault("PORT", "8000"),
		GoogleClientID:     mustGet("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: mustGet("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  mustGet("GOOGLE_REDIRECT_URL"),
	}
}

func mustGet(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

func getOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
