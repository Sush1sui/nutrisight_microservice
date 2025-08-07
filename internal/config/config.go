package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PORT                string
	ServerURL           string
	USDA_API_KEY        string
	HUGGINGFACE_API_KEY string
	NUTRITIONIX_API_KEY string
	NUTRITIONIX_APP_ID  string
}

var Global *Config

func New() (*Config, error) {
	if err := godotenv.Load(); err != nil { fmt.Println("Error loading .env file") }

	port := os.Getenv("PORT")
	if port == "" {
		port = "1169" // Default port if not set
	}

	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		return nil, fmt.Errorf("SERVER_URL is not set in the environment variables")
	}

	usdaAPIKey := os.Getenv("USDA_API_KEY")
	if usdaAPIKey == "" {
		return nil, fmt.Errorf("USDA_API_KEY is not set in the environment variables")
	}

	huggingfaceAPIKey := os.Getenv("HUGGINGFACE_API_KEY")
	if huggingfaceAPIKey == "" {
		return nil, fmt.Errorf("HUGGINGFACE_API_KEY is not set in the environment variables")
	}

	nutritionixAPIKey := os.Getenv("NUTRITIONIX_API_KEY")
	if nutritionixAPIKey == "" {
		return nil, fmt.Errorf("NUTRITIONIX_API_KEY is not set in the environment variables")
	}

	nutritionixAppID := os.Getenv("NUTRITIONIX_APP_ID")
	if nutritionixAppID == "" {
		return nil, fmt.Errorf("NUTRITIONIX_APP_ID is not set in the environment variables")
	}

	return &Config{
		PORT:                port,
		ServerURL:           serverURL,
		USDA_API_KEY:        usdaAPIKey,
		HUGGINGFACE_API_KEY: huggingfaceAPIKey,
		NUTRITIONIX_API_KEY: nutritionixAPIKey,
		NUTRITIONIX_APP_ID:  nutritionixAppID,
	}, nil
}