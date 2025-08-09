package http

import (
    "net/http"
    "my-project/usecase"
    "github.com/gin-gonic/gin"
)

type IShareHandler interface {
    ShareVideo(ctx *gin.Context)
    GetShareStatus(ctx *gin.Context)
    GetPlatforms(ctx *gin.Context)
}

type ShareHandler struct {
    shareUsecase usecase.IShareUsecase
    platforms    []string
}

func NewShareHandler(uc usecase.IShareUsecase, platforms []string) IShareHandler {
    return &ShareHandler{shareUsecase: uc, platforms: platforms}
}

type shareRequest struct {
    Platforms []string `json:"platforms"`
    Mode      string   `json:"mode"`
}

func (h *ShareHandler) ShareVideo(ctx *gin.Context) {
    videoID := ctx.Param("videoId")
    userID := ctx.GetString("user_id")
    var req shareRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"}); return
    }
    mode := usecase.ShareMode(req.Mode)
    if mode == "" { mode = usecase.ShareModeTrackOnly }
    results, err := h.shareUsecase.Share(ctx.Request.Context(), videoID, userID, req.Platforms, mode)
    if err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    ctx.JSON(http.StatusOK, gin.H{"video_id": videoID, "results": results})
}

func (h *ShareHandler) GetShareStatus(ctx *gin.Context) {
    videoID := ctx.Param("videoId")
    userID := ctx.GetString("user_id")
    list, err := h.shareUsecase.GetStatus(ctx.Request.Context(), videoID, userID)
    if err != nil { ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    ctx.JSON(http.StatusOK, gin.H{"video_id": videoID, "records": list})
}

func (h *ShareHandler) GetPlatforms(ctx *gin.Context) {
    caps := make([]gin.H, 0, len(h.platforms))
    for _, p := range h.platforms {
        caps = append(caps, gin.H{"platform": p, "server_post_supported": p=="twitter" || p=="facebook", "implemented": false})
    }
    ctx.JSON(http.StatusOK, gin.H{"platforms": caps})
}
