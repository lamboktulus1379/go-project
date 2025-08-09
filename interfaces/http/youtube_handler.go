package http

import (
	"net/http"
	"strconv"

	"my-project/domain/dto"
	"my-project/usecase"

	"github.com/gin-gonic/gin"
)

// IYouTubeHandler defines the interface for YouTube HTTP handlers
type IYouTubeHandler interface {
	// Video operations
	GetMyVideos(ctx *gin.Context)
	GetVideoDetails(ctx *gin.Context)
	UploadVideo(ctx *gin.Context)
	SearchVideos(ctx *gin.Context)

	// Comment operations
	GetVideoComments(ctx *gin.Context)
	AddComment(ctx *gin.Context)
	UpdateComment(ctx *gin.Context)
	DeleteComment(ctx *gin.Context)

	// Like operations
	LikeVideo(ctx *gin.Context)
	DislikeVideo(ctx *gin.Context)
	RemoveVideoRating(ctx *gin.Context)
	LikeComment(ctx *gin.Context)

	// Channel operations
	GetMyChannel(ctx *gin.Context)
	GetChannelDetails(ctx *gin.Context)

	// Playlist operations
	GetMyPlaylists(ctx *gin.Context)
	CreatePlaylist(ctx *gin.Context)
}

// YouTubeHandler implements the YouTube HTTP handlers
type YouTubeHandler struct {
	youtubeUseCase usecase.IYouTubeUseCase
}

// NewYouTubeHandler creates a new YouTube handler instance
func NewYouTubeHandler(youtubeUseCase usecase.IYouTubeUseCase) IYouTubeHandler {
	return &YouTubeHandler{
		youtubeUseCase: youtubeUseCase,
	}
}

// GetMyVideos handles GET /api/youtube/videos
func (h *YouTubeHandler) GetMyVideos(ctx *gin.Context) {
	req := &dto.YouTubeVideoListRequest{}

	// Support both snake_case and camelCase query params from frontend
	maxResultsRaw := ctx.Query("max_results")
	if maxResultsRaw == "" {
		maxResultsRaw = ctx.Query("maxResults")
	}
	if maxResultsRaw != "" {
		if val, err := strconv.ParseInt(maxResultsRaw, 10, 64); err == nil {
			req.MaxResults = val
		}
	}
	pageToken := ctx.Query("page_token")
	if pageToken == "" {
		pageToken = ctx.Query("pageToken")
	}
	req.PageToken = pageToken
	req.Order = ctx.Query("order")
	req.Q = ctx.Query("q")
	publishedAfter := ctx.Query("published_after")
	if publishedAfter == "" {
		publishedAfter = ctx.Query("publishedAfter")
	}
	req.PublishedAfter = publishedAfter
	publishedBefore := ctx.Query("published_before")
	if publishedBefore == "" {
		publishedBefore = ctx.Query("publishedBefore")
	}
	req.PublishedBefore = publishedBefore
	channelID := ctx.Query("channel_id")
	if channelID == "" {
		channelID = ctx.Query("channelId")
	}
	req.ChannelID = channelID

	response, err := h.youtubeUseCase.GetMyVideos(ctx.Request.Context(), req)
	if err != nil {
		// Provide fallback mock data so FE can still render something and display error
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get videos",
			"message": err.Error(),
			"data": []gin.H{
				{"id": "mock-error-1", "title": "YouTube fetch failed", "description": err.Error()},
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true, "data": response})
}

// GetVideoDetails handles GET /api/youtube/videos/:videoId
func (h *YouTubeHandler) GetVideoDetails(ctx *gin.Context) {
	videoID := ctx.Param("videoId")
	if videoID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Video ID is required",
		})
		return
	}

	video, err := h.youtubeUseCase.GetVideoDetails(ctx.Request.Context(), videoID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get video details",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    video,
	})
}

// UploadVideo handles POST /api/youtube/videos/upload
func (h *YouTubeHandler) UploadVideo(ctx *gin.Context) {
	var req dto.YouTubeVideoUploadRequest

	// Parse form data
	req.Title = ctx.PostForm("title")
	req.Description = ctx.PostForm("description")
	req.CategoryID = ctx.PostForm("category_id")
	req.Privacy = ctx.PostForm("privacy")

	// Parse tags (comma-separated)
	if tags := ctx.PostForm("tags"); tags != "" {
		// You might want to implement proper tag parsing here
		req.Tags = []string{tags}
	}

	// Get file
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "File is required",
		})
		return
	}
	req.File = file

	// Validate required fields
	if req.Title == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Title is required",
		})
		return
	}

	video, err := h.youtubeUseCase.UploadVideo(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to upload video",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    video,
	})
}

// SearchVideos handles GET /api/youtube/search
func (h *YouTubeHandler) SearchVideos(ctx *gin.Context) {
	req := &dto.YouTubeSearchRequest{}

	// Parse query parameters
	req.Q = ctx.Query("q")
	if req.Q == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Search query (q) is required",
		})
		return
	}

	if maxResults := ctx.Query("max_results"); maxResults != "" {
		if val, err := strconv.ParseInt(maxResults, 10, 64); err == nil {
			req.MaxResults = val
		}
	}

	req.PageToken = ctx.Query("page_token")
	req.Order = ctx.Query("order")
	req.Type = ctx.Query("type")
	req.ChannelID = ctx.Query("channel_id")
	req.PublishedAfter = ctx.Query("published_after")
	req.PublishedBefore = ctx.Query("published_before")
	req.Duration = ctx.Query("duration")
	req.Definition = ctx.Query("definition")

	response, err := h.youtubeUseCase.SearchVideos(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search videos",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetVideoComments handles GET /api/youtube/videos/:videoId/comments
func (h *YouTubeHandler) GetVideoComments(ctx *gin.Context) {
	videoID := ctx.Param("videoId")
	if videoID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Video ID is required",
		})
		return
	}

	req := &dto.YouTubeCommentListRequest{
		VideoID: videoID,
	}

	// Parse query parameters
	if maxResults := ctx.Query("max_results"); maxResults != "" {
		if val, err := strconv.ParseInt(maxResults, 10, 64); err == nil {
			req.MaxResults = val
		}
	}

	req.PageToken = ctx.Query("page_token")
	req.Order = ctx.Query("order")

	response, err := h.youtubeUseCase.GetVideoComments(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get video comments",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// AddComment handles POST /api/youtube/comments
func (h *YouTubeHandler) AddComment(ctx *gin.Context) {
	var req dto.YouTubeCommentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	comment, err := h.youtubeUseCase.AddComment(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add comment",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    comment,
	})
}

// UpdateComment handles PUT /api/youtube/comments/:commentId
func (h *YouTubeHandler) UpdateComment(ctx *gin.Context) {
	commentID := ctx.Param("commentId")
	if commentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Comment ID is required",
		})
		return
	}

	var reqBody struct {
		Text string `json:"text" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	req := &dto.YouTubeCommentUpdateRequest{
		CommentID: commentID,
		Text:      reqBody.Text,
	}

	comment, err := h.youtubeUseCase.UpdateComment(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update comment",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    comment,
	})
}

// DeleteComment handles DELETE /api/youtube/comments/:commentId
func (h *YouTubeHandler) DeleteComment(ctx *gin.Context) {
	commentID := ctx.Param("commentId")
	if commentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Comment ID is required",
		})
		return
	}

	err := h.youtubeUseCase.DeleteComment(ctx.Request.Context(), commentID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete comment",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Comment deleted successfully",
	})
}

// LikeVideo handles POST /api/youtube/videos/:videoId/like
func (h *YouTubeHandler) LikeVideo(ctx *gin.Context) {
	videoID := ctx.Param("videoId")
	if videoID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Video ID is required",
		})
		return
	}

	err := h.youtubeUseCase.LikeVideo(ctx.Request.Context(), videoID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to like video",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Video liked successfully",
	})
}

// DislikeVideo handles POST /api/youtube/videos/:videoId/dislike
func (h *YouTubeHandler) DislikeVideo(ctx *gin.Context) {
	videoID := ctx.Param("videoId")
	if videoID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Video ID is required",
		})
		return
	}

	err := h.youtubeUseCase.DislikeVideo(ctx.Request.Context(), videoID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to dislike video",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Video disliked successfully",
	})
}

// RemoveVideoRating handles DELETE /api/youtube/videos/:videoId/rating
func (h *YouTubeHandler) RemoveVideoRating(ctx *gin.Context) {
	videoID := ctx.Param("videoId")
	if videoID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Video ID is required",
		})
		return
	}

	err := h.youtubeUseCase.RemoveVideoRating(ctx.Request.Context(), videoID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to remove video rating",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Video rating removed successfully",
	})
}

// LikeComment handles POST /api/youtube/comments/:commentId/like
func (h *YouTubeHandler) LikeComment(ctx *gin.Context) {
	commentID := ctx.Param("commentId")
	if commentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Comment ID is required",
		})
		return
	}

	err := h.youtubeUseCase.LikeComment(ctx.Request.Context(), commentID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to like comment",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Comment liked successfully",
	})
}

// GetMyChannel handles GET /api/youtube/channel
func (h *YouTubeHandler) GetMyChannel(ctx *gin.Context) {
	channel, err := h.youtubeUseCase.GetMyChannel(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get channel",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    channel,
	})
}

// GetChannelDetails handles GET /api/youtube/channels/:channelId
func (h *YouTubeHandler) GetChannelDetails(ctx *gin.Context) {
	channelID := ctx.Param("channelId")
	if channelID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Channel ID is required",
		})
		return
	}

	channel, err := h.youtubeUseCase.GetChannelDetails(ctx.Request.Context(), channelID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get channel details",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    channel,
	})
}

// GetMyPlaylists handles GET /api/youtube/playlists
func (h *YouTubeHandler) GetMyPlaylists(ctx *gin.Context) {
	playlists, err := h.youtubeUseCase.GetMyPlaylists(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get playlists",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    playlists,
	})
}

// CreatePlaylist handles POST /api/youtube/playlists
func (h *YouTubeHandler) CreatePlaylist(ctx *gin.Context) {
	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Privacy     string `json:"privacy"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	playlist, err := h.youtubeUseCase.CreatePlaylist(ctx.Request.Context(), req.Title, req.Description, req.Privacy)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create playlist",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    playlist,
	})
}
