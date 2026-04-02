package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {

	err := godotenv.Load()

	if err != nil {
		log.Println("Note: No .env file found, relying on system environment variables")
	}

}

func GetEnv(key string) string {
	return os.Getenv(key)
}
