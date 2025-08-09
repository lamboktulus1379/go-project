package usecase

import (
	"context"
	"fmt"
	"my-project/domain/dto"
	"my-project/domain/model"
	"my-project/domain/repository"
)

// IYouTubeUseCase defines the interface for YouTube use case operations
type IYouTubeUseCase interface {
	// Video operations
	GetMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error)
	GetVideoDetails(ctx context.Context, videoID string) (*model.YouTubeVideo, error)
	UploadVideo(ctx context.Context, req *dto.YouTubeVideoUploadRequest) (*model.YouTubeVideo, error)
	UpdateVideo(ctx context.Context, videoID string, req *dto.YouTubeVideoUpdateRequest) (*model.YouTubeVideo, error)
	SearchVideos(ctx context.Context, req *dto.YouTubeSearchRequest) (*dto.YouTubeVideoResponse, error)

	// Comment operations
	GetVideoComments(ctx context.Context, req *dto.YouTubeCommentListRequest) (*dto.YouTubeCommentResponse, error)
	AddComment(ctx context.Context, req *dto.YouTubeCommentRequest) (*model.YouTubeComment, error)
	UpdateComment(ctx context.Context, req *dto.YouTubeCommentUpdateRequest) (*model.YouTubeComment, error)
	DeleteComment(ctx context.Context, commentID string) error

	// Like operations
	LikeVideo(ctx context.Context, videoID string) error
	DislikeVideo(ctx context.Context, videoID string) error
	RemoveVideoRating(ctx context.Context, videoID string) error
	ToggleCommentLike(ctx context.Context, userID, commentID string) (bool, error)
	ToggleCommentHeart(ctx context.Context, userID, commentID string) (bool, error)

	// Channel operations
	GetMyChannel(ctx context.Context) (*model.YouTubeChannel, error)
	GetChannelDetails(ctx context.Context, channelID string) (*model.YouTubeChannel, error)

	// Playlist operations
	GetMyPlaylists(ctx context.Context) ([]model.YouTubePlaylist, error)
	CreatePlaylist(ctx context.Context, title, description, privacy string) (*model.YouTubePlaylist, error)
}

// YouTubeUseCase implements the YouTube use case operations
type YouTubeUseCase struct {
	youtubeRepo repository.IYouTube
}

// NewYouTubeUseCase creates a new YouTube use case instance
func NewYouTubeUseCase(youtubeRepo repository.IYouTube) IYouTubeUseCase {
	return &YouTubeUseCase{
		youtubeRepo: youtubeRepo,
	}
}

// GetMyVideos retrieves videos from the authenticated user's channel
func (u *YouTubeUseCase) GetMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error) {
	if req == nil {
		req = &dto.YouTubeVideoListRequest{
			MaxResults: 25,
			Order:      "date",
		}
	}

	// Set default values if not provided
	if req.MaxResults == 0 {
		req.MaxResults = 25
	}
	if req.Order == "" {
		req.Order = "date"
	}

	// Validate max results limit
	if req.MaxResults > 50 {
		req.MaxResults = 50
	}

	response, err := u.youtubeRepo.GetMyVideos(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get my videos: %w", err)
	}

	return response, nil
}

// GetVideoDetails retrieves details for a specific video
func (u *YouTubeUseCase) GetVideoDetails(ctx context.Context, videoID string) (*model.YouTubeVideo, error) {
	if videoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}

	video, err := u.youtubeRepo.GetVideoDetails(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video details: %w", err)
	}

	return video, nil
}

// UploadVideo uploads a video to YouTube
func (u *YouTubeUseCase) UploadVideo(ctx context.Context, req *dto.YouTubeVideoUploadRequest) (*model.YouTubeVideo, error) {
	if req == nil {
		return nil, fmt.Errorf("upload request is required")
	}

	if req.Title == "" {
		return nil, fmt.Errorf("video title is required")
	}

	if req.File == nil {
		return nil, fmt.Errorf("video file is required")
	}

	// Set default privacy if not provided
	if req.Privacy == "" {
		req.Privacy = "private"
	}

	// Validate privacy setting
	validPrivacySettings := map[string]bool{
		"private":  true,
		"public":   true,
		"unlisted": true,
	}
	if !validPrivacySettings[req.Privacy] {
		return nil, fmt.Errorf("invalid privacy setting: %s", req.Privacy)
	}

	video, err := u.youtubeRepo.UploadVideo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}

	return video, nil
}

// UpdateVideo updates metadata for an existing video
func (u *YouTubeUseCase) UpdateVideo(ctx context.Context, videoID string, req *dto.YouTubeVideoUpdateRequest) (*model.YouTubeVideo, error) {
	if videoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update request is required")
	}

	// Build updates map only with provided fields
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}
	if req.Privacy != nil {
		p := *req.Privacy
		validPrivacySettings := map[string]bool{"private": true, "public": true, "unlisted": true}
		if !validPrivacySettings[p] {
			return nil, fmt.Errorf("invalid privacy setting: %s", p)
		}
		updates["privacy"] = p
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields provided to update")
	}

	video, err := u.youtubeRepo.UpdateVideo(ctx, videoID, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update video: %w", err)
	}
	return video, nil
}

// SearchVideos searches for videos on YouTube
func (u *YouTubeUseCase) SearchVideos(ctx context.Context, req *dto.YouTubeSearchRequest) (*dto.YouTubeVideoResponse, error) {
	if req == nil || req.Q == "" {
		return nil, fmt.Errorf("search query is required")
	}

	// Set default values
	if req.MaxResults == 0 {
		req.MaxResults = 25
	}
	if req.Order == "" {
		req.Order = "relevance"
	}
	if req.Type == "" {
		req.Type = "video"
	}

	// Validate max results limit
	if req.MaxResults > 50 {
		req.MaxResults = 50
	}

	response, err := u.youtubeRepo.SearchVideos(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search videos: %w", err)
	}

	return response, nil
}

// GetVideoComments retrieves comments for a specific video
func (u *YouTubeUseCase) GetVideoComments(ctx context.Context, req *dto.YouTubeCommentListRequest) (*dto.YouTubeCommentResponse, error) {
	if req == nil || req.VideoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}

	// Set default values
	if req.MaxResults == 0 {
		req.MaxResults = 20
	}
	if req.Order == "" {
		req.Order = "time"
	}

	// Validate max results limit
	if req.MaxResults > 100 {
		req.MaxResults = 100
	}

	response, err := u.youtubeRepo.GetVideoComments(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get video comments: %w", err)
	}

	return response, nil
}

// AddComment adds a comment to a video
func (u *YouTubeUseCase) AddComment(ctx context.Context, req *dto.YouTubeCommentRequest) (*model.YouTubeComment, error) {
	if req == nil {
		return nil, fmt.Errorf("comment request is required")
	}

	if req.VideoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}

	if req.Text == "" {
		return nil, fmt.Errorf("comment text is required")
	}

	// Validate comment length
	if len(req.Text) > 10000 {
		return nil, fmt.Errorf("comment text too long (max 10000 characters)")
	}

	comment, err := u.youtubeRepo.AddComment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	return comment, nil
}

// UpdateComment updates an existing comment
func (u *YouTubeUseCase) UpdateComment(ctx context.Context, req *dto.YouTubeCommentUpdateRequest) (*model.YouTubeComment, error) {
	if req == nil {
		return nil, fmt.Errorf("comment update request is required")
	}

	if req.CommentID == "" {
		return nil, fmt.Errorf("comment ID is required")
	}

	if req.Text == "" {
		return nil, fmt.Errorf("comment text is required")
	}

	// Validate comment length
	if len(req.Text) > 10000 {
		return nil, fmt.Errorf("comment text too long (max 10000 characters)")
	}

	comment, err := u.youtubeRepo.UpdateComment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	return comment, nil
}

// DeleteComment deletes a comment
func (u *YouTubeUseCase) DeleteComment(ctx context.Context, commentID string) error {
	if commentID == "" {
		return fmt.Errorf("comment ID is required")
	}

	err := u.youtubeRepo.DeleteComment(ctx, commentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	return nil
}

// LikeVideo likes a video
func (u *YouTubeUseCase) LikeVideo(ctx context.Context, videoID string) error {
	if videoID == "" {
		return fmt.Errorf("video ID is required")
	}

	err := u.youtubeRepo.LikeVideo(ctx, videoID)
	if err != nil {
		return fmt.Errorf("failed to like video: %w", err)
	}

	return nil
}

// DislikeVideo dislikes a video
func (u *YouTubeUseCase) DislikeVideo(ctx context.Context, videoID string) error {
	if videoID == "" {
		return fmt.Errorf("video ID is required")
	}

	err := u.youtubeRepo.DislikeVideo(ctx, videoID)
	if err != nil {
		return fmt.Errorf("failed to dislike video: %w", err)
	}

	return nil
}

// RemoveVideoRating removes rating from a video
func (u *YouTubeUseCase) RemoveVideoRating(ctx context.Context, videoID string) error {
	if videoID == "" {
		return fmt.Errorf("video ID is required")
	}

	err := u.youtubeRepo.RemoveVideoRating(ctx, videoID)
	if err != nil {
		return fmt.Errorf("failed to remove video rating: %w", err)
	}

	return nil
}


// ToggleCommentLike toggles like state (in-memory) for a user's comment
func (u *YouTubeUseCase) ToggleCommentLike(ctx context.Context, userID, commentID string) (bool, error) {
	if userID == "" || commentID == "" { return false, fmt.Errorf("userID and commentID are required") }
	liked, err := u.youtubeRepo.ToggleUserCommentLike(ctx, userID, commentID)
	if err != nil { return false, fmt.Errorf("failed to toggle comment like: %w", err) }
	return liked, nil
}

// ToggleCommentHeart toggles heart state (in-memory) for a user's comment
func (u *YouTubeUseCase) ToggleCommentHeart(ctx context.Context, userID, commentID string) (bool, error) {
	if userID == "" || commentID == "" { return false, fmt.Errorf("userID and commentID are required") }
	loved, err := u.youtubeRepo.ToggleUserCommentHeart(ctx, userID, commentID)
	if err != nil { return false, fmt.Errorf("failed to toggle comment heart: %w", err) }
	return loved, nil
}

// GetMyChannel retrieves the authenticated user's channel information
func (u *YouTubeUseCase) GetMyChannel(ctx context.Context) (*model.YouTubeChannel, error) {
	channel, err := u.youtubeRepo.GetMyChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get my channel: %w", err)
	}

	return channel, nil
}

// GetChannelDetails retrieves details for a specific channel
func (u *YouTubeUseCase) GetChannelDetails(ctx context.Context, channelID string) (*model.YouTubeChannel, error) {
	if channelID == "" {
		return nil, fmt.Errorf("channel ID is required")
	}

	channel, err := u.youtubeRepo.GetChannelDetails(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel details: %w", err)
	}

	return channel, nil
}

// GetMyPlaylists retrieves the authenticated user's playlists
func (u *YouTubeUseCase) GetMyPlaylists(ctx context.Context) ([]model.YouTubePlaylist, error) {
	playlists, err := u.youtubeRepo.GetMyPlaylists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get my playlists: %w", err)
	}

	return playlists, nil
}

// CreatePlaylist creates a new playlist
func (u *YouTubeUseCase) CreatePlaylist(ctx context.Context, title, description, privacy string) (*model.YouTubePlaylist, error) {
	if title == "" {
		return nil, fmt.Errorf("playlist title is required")
	}

	// Set default privacy if not provided
	if privacy == "" {
		privacy = "private"
	}

	// Validate privacy setting
	validPrivacySettings := map[string]bool{
		"private":  true,
		"public":   true,
		"unlisted": true,
	}
	if !validPrivacySettings[privacy] {
		return nil, fmt.Errorf("invalid privacy setting: %s", privacy)
	}

	playlist, err := u.youtubeRepo.CreatePlaylist(ctx, title, description, privacy)
	if err != nil {
		return nil, fmt.Errorf("failed to create playlist: %w", err)
	}

	return playlist, nil
}
