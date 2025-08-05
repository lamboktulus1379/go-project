package repository

import (
	"context"
	"my-project/domain/dto"
	"my-project/domain/model"
)

// IYouTube defines the interface for YouTube repository operations
type IYouTube interface {
	// Video operations
	GetMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error)
	GetVideoDetails(ctx context.Context, videoID string) (*model.YouTubeVideo, error)
	UploadVideo(ctx context.Context, req *dto.YouTubeVideoUploadRequest) (*model.YouTubeVideo, error)
	UpdateVideo(ctx context.Context, videoID string, updates map[string]interface{}) (*model.YouTubeVideo, error)
	DeleteVideo(ctx context.Context, videoID string) error

	// Search operations
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
	LikeComment(ctx context.Context, commentID string) error
	DislikeComment(ctx context.Context, commentID string) error
	RemoveCommentRating(ctx context.Context, commentID string) error

	// Channel operations
	GetMyChannel(ctx context.Context) (*model.YouTubeChannel, error)
	GetChannelDetails(ctx context.Context, channelID string) (*model.YouTubeChannel, error)
	GetChannelVideos(ctx context.Context, channelID string, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error)

	// Playlist operations
	GetMyPlaylists(ctx context.Context) ([]model.YouTubePlaylist, error)
	GetPlaylistVideos(ctx context.Context, req *dto.YouTubePlaylistRequest) (*dto.YouTubeVideoResponse, error)
	CreatePlaylist(ctx context.Context, title, description, privacy string) (*model.YouTubePlaylist, error)
	AddVideoToPlaylist(ctx context.Context, playlistID, videoID string) error
	RemoveVideoFromPlaylist(ctx context.Context, playlistID, videoID string) error

	// Analytics (if you want to add analytics data)
	GetVideoAnalytics(ctx context.Context, videoID string, startDate, endDate string) (interface{}, error)
	GetChannelAnalytics(ctx context.Context, startDate, endDate string) (interface{}, error)
}
