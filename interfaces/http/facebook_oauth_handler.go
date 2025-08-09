package http

import (
    "crypto/rand"
    "encoding/hex"
    "net/http"
    "net/url"
    "time"

    "my-project/infrastructure/configuration"
    "my-project/infrastructure/logger"
    "my-project/infrastructure/persistence"
    "my-project/domain/model"

    "github.com/gin-gonic/gin"
)

type IFacebookOAuthHandler interface {
    GetAuthURL(ctx *gin.Context)
    Callback(ctx *gin.Context)
}

type facebookOAuthHandler struct {
    tokenRepo *persistence.OAuthTokenRepository
}

func NewFacebookOAuthHandler(tokenRepo *persistence.OAuthTokenRepository) IFacebookOAuthHandler {
    return &facebookOAuthHandler{tokenRepo: tokenRepo}
}

func randomState() string {
    b := make([]byte, 16)
    _, _ = rand.Read(b)
    return hex.EncodeToString(b)
}

// GetAuthURL builds Facebook OAuth URL (user must approve in browser)
func (h *facebookOAuthHandler) GetAuthURL(c *gin.Context) {
    conf := configuration.C.OAuth.Facebook
    if conf.ClientID == "" || conf.RedirectURI == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "facebook oauth not configured"})
        return
    }
    state := randomState()
    // TODO: store 'state' securely (session/cache). For now we return state and expect client to echo it back.
    // Comma-separated scopes; we do not urlencode the commas here because Facebook expects raw list.
    scopes := "pages_show_list,pages_read_engagement,pages_manage_posts,public_profile"
    u := url.URL{Scheme: "https", Host: "www.facebook.com", Path: "/v19.0/dialog/oauth"}
    q := u.Query()
    q.Set("client_id", conf.ClientID)
    q.Set("redirect_uri", conf.RedirectURI)
    q.Set("state", state)
    q.Set("scope", scopes)
    u.RawQuery = q.Encode()
    c.JSON(http.StatusOK, gin.H{"auth_url": u.String(), "state": state})
}

// Callback exchanges code for token(s) (placeholder – real FB token exchange not yet implemented)
func (h *facebookOAuthHandler) Callback(c *gin.Context) {
    code := c.Query("code")
    state := c.Query("state")
    if code == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
        return
    }
    // NOTE: validate state (skipped in placeholder)
    userID := c.GetString("user_id")
    if userID == "" { // fallback for unauthenticated testing
        userID = "demo-user"
    }
    dummy := &model.OAuthToken{
        UserID:      userID,
        Platform:    "facebook",
        AccessToken: "DUMMY_PAGE_TOKEN", // replace after real exchange
        Scopes:      "pages_show_list,pages_read_engagement,pages_manage_posts",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
    }
    if err := h.tokenRepo.UpsertToken(c.Request.Context(), dummy); err != nil {
        logger.GetLogger().WithField("error", err).Error("failed to upsert facebook token")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "store token failed"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"connected": true, "state": state})
}
