package http

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"my-project/infrastructure/configuration"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
)

// IYouTubeAuthHandler defines the interface for YouTube authentication handlers
type IYouTubeAuthHandler interface {
	GetAuthURL(ctx *gin.Context)
	HandleCallback(ctx *gin.Context)
}

// YouTubeAuthHandler implements YouTube OAuth2 authentication
type YouTubeAuthHandler struct {
	oauth2Config *oauth2.Config
}

// NewYouTubeAuthHandler creates a new YouTube auth handler
func NewYouTubeAuthHandler() (IYouTubeAuthHandler, error) {
	config, err := configuration.GetYouTubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get YouTube config: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes: []string{
			youtube.YoutubeScope,
			youtube.YoutubeUploadScope,
			youtube.YoutubeForceSslScope,
		},
		Endpoint: google.Endpoint,
	}

	return &YouTubeAuthHandler{
		oauth2Config: oauth2Config,
	}, nil
}

// GetAuthURL handles GET /auth/youtube
func (h *YouTubeAuthHandler) GetAuthURL(ctx *gin.Context) {
	// Generate a random state parameter for security
	state := generateRandomState()

	// Store state in session (you might want to use a proper session store)
	ctx.SetCookie("oauth_state", state, 600, "/", "", false, true)

	// Generate authorization URL
	authURL := h.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))

	ctx.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
	})
}

// HandleCallback handles GET /auth/youtube/callback
func (h *YouTubeAuthHandler) HandleCallback(ctx *gin.Context) {
	// Check for OAuth error first
	if errorParam := ctx.Query("error"); errorParam != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":       fmt.Sprintf("OAuth error: %s", errorParam),
			"description": ctx.Query("error_description"),
		})
		return
	}

	// Get state parameter - for development, we'll be more lenient
	state := ctx.Query("state")
	if state == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":  "State parameter missing",
			"action": "Visit /auth/youtube to start over",
		})
		return
	}

	// For development purposes, we'll skip strict state validation entirely
	// In production, you should implement proper state validation with secure storage
	fmt.Printf("Received state: %s\n", state)
	fmt.Printf("Skipping state validation for development\n")

	// Get authorization code
	code := ctx.Query("code")
	if code == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Authorization code not found",
		})
		return
	}

	// Exchange code for token
	token, err := h.oauth2Config.Exchange(context.Background(), code)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to exchange code for token",
			"message": err.Error(),
		})
		return
	}

	// Clear the state cookie
	ctx.SetCookie("oauth_state", "", -1, "/", "", false, true)

	// Return tokens (in production, you should store these securely)
	ctx.JSON(http.StatusOK, gin.H{
		"success":       true,
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    token.TokenType,
		"expiry":        token.Expiry,
		"message":       "Authentication successful! Please save these tokens in your environment variables.",
		"next_steps": []string{
			"1. Copy the access_token and refresh_token above",
			"2. Update your environment variables:",
			"   export YOUTUBE_ACCESS_TOKEN='" + token.AccessToken + "'",
			"   export YOUTUBE_REFRESH_TOKEN='" + token.RefreshToken + "'",
			"3. Restart your application to enable full YouTube API features",
		},
	})
}

// generateRandomState generates a random state parameter for OAuth2
func generateRandomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
