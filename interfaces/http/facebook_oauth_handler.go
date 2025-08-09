package http

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "sync"
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
    Status(ctx *gin.Context)
}

type facebookOAuthHandler struct {
    tokenRepo *persistence.OAuthTokenRepository
    stateMu   sync.Mutex
    states    map[string]time.Time // state -> expiry
}

func NewFacebookOAuthHandler(tokenRepo *persistence.OAuthTokenRepository) IFacebookOAuthHandler {
    return &facebookOAuthHandler{tokenRepo: tokenRepo, states: map[string]time.Time{}}
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
    // store state with 10 minute expiry
    h.stateMu.Lock()
    h.states[state] = time.Now().Add(10 * time.Minute)
    h.stateMu.Unlock()
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
    lg := logger.GetLogger()
    conf := configuration.C.OAuth.Facebook
    code := c.Query("code")
    state := c.Query("state")
    if code == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
        return
    }
    // validate state
    h.stateMu.Lock()
    exp, ok := h.states[state]
    if ok && time.Now().After(exp) { // expired
        ok = false
    }
    if ok {
        delete(h.states, state)
    }
    h.stateMu.Unlock()
    if !ok {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state"})
        return
    }

    userID := c.GetString("user_id")
    if userID == "" { // fallback for dev
        userID = "demo-user"
    }

    // 1. Exchange code for short-lived user access token
    tokenURL := fmt.Sprintf("https://graph.facebook.com/v19.0/oauth/access_token?client_id=%s&redirect_uri=%s&client_secret=%s&code=%s",
        url.QueryEscape(conf.ClientID), url.QueryEscape(conf.RedirectURI), url.QueryEscape(conf.ClientSecret), url.QueryEscape(code))
    shortTokResp, err := http.Get(tokenURL)
    if err != nil {
        lg.Errorf("facebook token exchange request error: %v", err)
        c.JSON(http.StatusBadGateway, gin.H{"error": "token_request_failed"})
        return
    }
    body, _ := io.ReadAll(shortTokResp.Body)
    shortTokResp.Body.Close()
    if shortTokResp.StatusCode != 200 {
        lg.WithField("body", string(body)).Error("facebook token exchange failed")
        c.JSON(http.StatusBadGateway, gin.H{"error": "token_exchange_failed"})
        return
    }
    var shortData struct {
        AccessToken string `json:"access_token"`
        TokenType   string `json:"token_type"`
        ExpiresIn   int    `json:"expires_in"`
    }
    if err := json.Unmarshal(body, &shortData); err != nil {
        lg.WithField("err", err).Error("unmarshal short token")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "parse_token_failed"})
        return
    }
    // 2. Exchange short-lived for long-lived token
    llURL := fmt.Sprintf("https://graph.facebook.com/v19.0/oauth/access_token?grant_type=fb_exchange_token&client_id=%s&client_secret=%s&fb_exchange_token=%s",
        url.QueryEscape(conf.ClientID), url.QueryEscape(conf.ClientSecret), url.QueryEscape(shortData.AccessToken))
    llResp, err := http.Get(llURL)
    if err != nil {
        lg.Errorf("facebook long-lived exchange error: %v", err)
        c.JSON(http.StatusBadGateway, gin.H{"error": "long_lived_exchange_failed"})
        return
    }
    llBody, _ := io.ReadAll(llResp.Body)
    llResp.Body.Close()
    if llResp.StatusCode != 200 {
        lg.WithField("body", string(llBody)).Error("long lived token exchange failed")
        c.JSON(http.StatusBadGateway, gin.H{"error": "long_lived_token_failed"})
        return
    }
    var llData struct {
        AccessToken string `json:"access_token"`
        TokenType   string `json:"token_type"`
        ExpiresIn   int    `json:"expires_in"`
    }
    if err := json.Unmarshal(llBody, &llData); err != nil {
        lg.WithField("err", err).Error("unmarshal long lived token")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "parse_long_token_failed"})
        return
    }
    expiresAt := time.Now().Add(time.Duration(llData.ExpiresIn) * time.Second).UTC()

    // 3. Get pages list using long-lived user token
    pagesURL := fmt.Sprintf("https://graph.facebook.com/v19.0/me/accounts?access_token=%s", url.QueryEscape(llData.AccessToken))
    pagesResp, err := http.Get(pagesURL)
    if err != nil {
        lg.Errorf("facebook pages request error: %v", err)
        c.JSON(http.StatusBadGateway, gin.H{"error": "pages_request_failed"})
        return
    }
    pagesBody, _ := io.ReadAll(pagesResp.Body)
    pagesResp.Body.Close()
    if pagesResp.StatusCode != 200 {
        lg.WithField("body", string(pagesBody)).Error("pages fetch failed")
        c.JSON(http.StatusBadGateway, gin.H{"error": "pages_fetch_failed"})
        return
    }
    var pages struct {
        Data []struct {
            Name        string `json:"name"`
            ID          string `json:"id"`
            AccessToken string `json:"access_token"`
        } `json:"data"`
    }
    if err := json.Unmarshal(pagesBody, &pages); err != nil {
        lg.WithField("err", err).Error("unmarshal pages list")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "parse_pages_failed"})
        return
    }
    if len(pages.Data) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "no_pages_available"})
        return
    }
    // For now auto-select first page; later expose selection UI
    selected := pages.Data[0]
    tokenType := "page"
    scopes := "pages_show_list,pages_read_engagement,pages_manage_posts,public_profile"

    tok := &model.OAuthToken{
        UserID:      userID,
        Platform:    "facebook",
        AccessToken: selected.AccessToken, // page token used for posting
        RefreshToken: "", // facebook page tokens typically long-lived; refresh not used
        ExpiresAt:   &expiresAt,
        Scopes:      scopes,
        PageID:      &selected.ID,
        PageName:    &selected.Name,
        TokenType:   &tokenType,
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
    }
    if err := h.tokenRepo.UpsertToken(c.Request.Context(), tok); err != nil {
        lg.WithField("error", err).Error("failed to upsert facebook token")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "store_token_failed"})
        return
    }
    if c.Query("frontend") == "1" {
        c.Header("Content-Type", "text/html; charset=utf-8")
        _, _ = c.Writer.Write([]byte(fmt.Sprintf(`<!DOCTYPE html><html><head><title>Facebook Connected</title></head><body><script>if (window.opener){window.opener.postMessage({source:'facebook-oauth',connected:true,page_id:'%s',page_name:%q},'*');window.close();}else{document.write('Facebook connected: %s');}</script></body></html>`, selected.ID, selected.Name, selected.Name)))
        return
    }
    c.JSON(http.StatusOK, gin.H{"connected": true, "page_id": selected.ID, "page_name": selected.Name})
}

// Status returns whether a facebook page token is stored
func (h *facebookOAuthHandler) Status(c *gin.Context) {
    userID := c.GetString("user_id")
    if userID == "" { userID = "demo-user" }
    tok, err := h.tokenRepo.GetToken(c.Request.Context(), userID, "facebook")
    if err != nil || tok == nil || tok.AccessToken == "" {
        c.JSON(http.StatusOK, gin.H{"connected": false})
        return
    }
    resp := gin.H{"connected": true}
    if tok.PageID != nil { resp["page_id"] = *tok.PageID }
    if tok.PageName != nil { resp["page_name"] = *tok.PageName }
    c.JSON(http.StatusOK, resp)
}
