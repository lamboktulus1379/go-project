package configuration

import (
	"errors"
	"os"
)

// YouTubeConfig represents YouTube API configuration
type YouTubeConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	AccessToken  string `mapstructure:"access_token"`
	RefreshToken string `mapstructure:"refresh_token"`
	ChannelID    string `mapstructure:"channel_id"`
}

// GetYouTubeConfig returns YouTube configuration from environment variables or config file
func GetYouTubeConfig() (*YouTubeConfig, error) {
	config := &YouTubeConfig{
		ClientID:     getEnv("YOUTUBE_CLIENT_ID", ""),
		ClientSecret: getEnv("YOUTUBE_CLIENT_SECRET", ""),
		RedirectURL:  getEnv("YOUTUBE_REDIRECT_URL", "http://localhost:8080/auth/youtube/callback"),
		AccessToken:  getEnv("YOUTUBE_ACCESS_TOKEN", ""),
		RefreshToken: getEnv("YOUTUBE_REFRESH_TOKEN", ""),
		ChannelID:    getEnv("YOUTUBE_CHANNEL_ID", ""),
	}

	// Validate required fields
	if config.ClientID == "" {
		return nil, errors.New("YOUTUBE_CLIENT_ID is required")
	}
	if config.ClientSecret == "" {
		return nil, errors.New("YOUTUBE_CLIENT_SECRET is required")
	}

	return config, nil
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
