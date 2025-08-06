package configuration

import (
	"errors"
	"os"
	"strings"
)

// YouTubeConfig represents YouTube API configuration
type YouTubeConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	AccessToken  string `mapstructure:"access_token"`
	RefreshToken string `mapstructure:"refresh_token"`
	ChannelID    string `mapstructure:"channel_id"`
	APIKey       string `mapstructure:"api_key"`
}

// GetYouTubeConfig returns YouTube configuration from JSON config with environment variable fallback
func GetYouTubeConfig() (*YouTubeConfig, error) {
	config := &YouTubeConfig{
		ClientID:     getConfigValue(C.YouTube.ClientID, "YOUTUBE_CLIENT_ID", ""),
		ClientSecret: getConfigValue(C.YouTube.ClientSecret, "YOUTUBE_CLIENT_SECRET", ""),
		RedirectURL:  getConfigValue(C.YouTube.RedirectURI, "YOUTUBE_REDIRECT_URL", "http://localhost:10001/auth/youtube/callback"),
		AccessToken:  getEnv("YOUTUBE_ACCESS_TOKEN", ""),
		RefreshToken: getEnv("YOUTUBE_REFRESH_TOKEN", ""),
		ChannelID:    getEnv("YOUTUBE_CHANNEL_ID", ""),
		APIKey:       getConfigValue(C.YouTube.APIKey, "YOUTUBE_API_KEY", ""),
	}

	// Validate required fields
	if config.ClientID == "" || config.ClientID == "YOUR_YOUTUBE_CLIENT_ID" {
		return nil, errors.New("YOUTUBE_CLIENT_ID is required. Please set it in config.json or environment variable")
	}
	if config.ClientSecret == "" || config.ClientSecret == "YOUR_YOUTUBE_CLIENT_SECRET" {
		return nil, errors.New("YOUTUBE_CLIENT_SECRET is required. Please set it in config.json or environment variable")
	}
	if config.APIKey == "" || config.APIKey == "YOUR_YOUTUBE_API_KEY" {
		return nil, errors.New("YOUTUBE_API_KEY is required. Please set it in config.json or environment variable")
	}

	return config, nil
}

// getConfigValue gets value from config first, then environment variable, then default
func getConfigValue(configValue, envKey, defaultValue string) string {
	// If config value is set and not a placeholder, use it
	if configValue != "" && !strings.HasPrefix(configValue, "YOUR_") {
		return configValue
	}
	// Otherwise fall back to environment variable
	return getEnv(envKey, defaultValue)
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
