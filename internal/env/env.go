package env

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv load env variables from .env file
func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}
}

// GetEnv return a value of an env variable
func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
