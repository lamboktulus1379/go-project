package youtube

import (
	"context"
	"fmt"
	"strings"
	"time"

	"my-project/domain/dto"
	"my-project/domain/model"
	"my-project/domain/repository"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Client represents YouTube API client
type Client struct {
	service     *youtube.Service
	channelID   string
	accessToken string
}

// Config represents YouTube API configuration
type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ChannelID    string `json:"channel_id"`
}

// NewYouTubeClient creates a new YouTube API client
func NewYouTubeClient(ctx context.Context, config *Config) (repository.IYouTube, error) {
	// OAuth2 configuration
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

	// Create token
	token := &oauth2.Token{
		AccessToken:  config.AccessToken,
		RefreshToken: config.RefreshToken,
		TokenType:    "Bearer",
	}

	// Create HTTP client with OAuth2
	httpClient := oauth2Config.Client(ctx, token)

	// Create YouTube service
	service, err := youtube.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube service: %w", err)
	}

	return &Client{
		service:     service,
		channelID:   config.ChannelID,
		accessToken: config.AccessToken,
	}, nil
}

// GetMyVideos retrieves videos from the authenticated user's channel
func (c *Client) GetMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error) {
	channelID := c.channelID
	if req.ChannelID != "" {
		channelID = req.ChannelID
	}

	call := c.service.Search.List([]string{"id", "snippet"}).
		ChannelId(channelID).
		Type("video").
		Order("date")

	if req.MaxResults > 0 {
		call = call.MaxResults(req.MaxResults)
	} else {
		call = call.MaxResults(25) // Default
	}

	if req.PageToken != "" {
		call = call.PageToken(req.PageToken)
	}

	if req.Order != "" {
		call = call.Order(req.Order)
	}

	if req.Q != "" {
		call = call.Q(req.Q)
	}

	if req.PublishedAfter != "" {
		if publishedAfter, err := time.Parse(time.RFC3339, req.PublishedAfter); err == nil {
			call = call.PublishedAfter(publishedAfter.Format(time.RFC3339))
		}
	}

	if req.PublishedBefore != "" {
		if publishedBefore, err := time.Parse(time.RFC3339, req.PublishedBefore); err == nil {
			call = call.PublishedBefore(publishedBefore.Format(time.RFC3339))
		}
	}

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	// Get video IDs for additional details
	var videoIDs []string
	for _, item := range response.Items {
		videoIDs = append(videoIDs, item.Id.VideoId)
	}

	// Get video statistics and content details
	videos := make([]interface{}, 0)
	if len(videoIDs) > 0 {
		videoDetails, err := c.service.Videos.List([]string{"snippet", "statistics", "contentDetails", "status"}).
			Id(strings.Join(videoIDs, ",")).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get video details: %w", err)
		}

		for _, video := range videoDetails.Items {
			ytVideo := c.convertToYouTubeVideo(video)
			videos = append(videos, ytVideo)
		}
	}

	return &dto.YouTubeVideoResponse{
		YouTubeResponse: dto.YouTubeResponse{
			Kind:          response.Kind,
			ETag:          response.Etag,
			NextPageToken: response.NextPageToken,
			PrevPageToken: response.PrevPageToken,
			PageInfo: dto.PageInfo{
				TotalResults:   response.PageInfo.TotalResults,
				ResultsPerPage: response.PageInfo.ResultsPerPage,
			},
		},
		Items: videos,
	}, nil
}

// GetVideoDetails retrieves details for a specific video
func (c *Client) GetVideoDetails(ctx context.Context, videoID string) (*model.YouTubeVideo, error) {
	call := c.service.Videos.List([]string{"snippet", "statistics", "contentDetails", "status"}).
		Id(videoID)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get video details: %w", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}

	video := c.convertToYouTubeVideo(response.Items[0])
	return &video, nil
}

// convertToYouTubeVideo converts YouTube API video to our model
func (c *Client) convertToYouTubeVideo(video *youtube.Video) model.YouTubeVideo {
	publishedAt, _ := time.Parse(time.RFC3339, video.Snippet.PublishedAt)

	var viewCount, likeCount int64
	if video.Statistics != nil {
		viewCount = int64(video.Statistics.ViewCount)
		likeCount = int64(video.Statistics.LikeCount)
	}

	ytVideo := model.YouTubeVideo{
		ID:          video.Id,
		Title:       video.Snippet.Title,
		Description: video.Snippet.Description,
		PublishedAt: publishedAt,
		ChannelID:   video.Snippet.ChannelId,
		ChannelName: video.Snippet.ChannelTitle,
		ViewCount:   viewCount,
		LikeCount:   likeCount,
		Duration:    video.ContentDetails.Duration,
		Tags:        video.Snippet.Tags,
		Status:      video.Status.PrivacyStatus,
		Category:    video.Snippet.CategoryId,
	}

	// Set thumbnails
	if video.Snippet.Thumbnails != nil {
		if video.Snippet.Thumbnails.Default != nil {
			ytVideo.Thumbnails.Default.URL = video.Snippet.Thumbnails.Default.Url
			ytVideo.Thumbnails.Default.Width = int(video.Snippet.Thumbnails.Default.Width)
			ytVideo.Thumbnails.Default.Height = int(video.Snippet.Thumbnails.Default.Height)
		}
		if video.Snippet.Thumbnails.Medium != nil {
			ytVideo.Thumbnails.Medium.URL = video.Snippet.Thumbnails.Medium.Url
			ytVideo.Thumbnails.Medium.Width = int(video.Snippet.Thumbnails.Medium.Width)
			ytVideo.Thumbnails.Medium.Height = int(video.Snippet.Thumbnails.Medium.Height)
		}
		if video.Snippet.Thumbnails.High != nil {
			ytVideo.Thumbnails.High.URL = video.Snippet.Thumbnails.High.Url
			ytVideo.Thumbnails.High.Width = int(video.Snippet.Thumbnails.High.Width)
			ytVideo.Thumbnails.High.Height = int(video.Snippet.Thumbnails.High.Height)
		}
	}

	return ytVideo
}

// UploadVideo uploads a video to YouTube
func (c *Client) UploadVideo(ctx context.Context, req *dto.YouTubeVideoUploadRequest) (*model.YouTubeVideo, error) {
	// Open the uploaded file
	file, err := req.File.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Prepare video snippet
	video := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       req.Title,
			Description: req.Description,
			Tags:        req.Tags,
			CategoryId:  req.CategoryID,
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus: req.Privacy,
		},
	}

	// Create upload call
	call := c.service.Videos.Insert([]string{"snippet", "status"}, video)

	// Set the media content
	call = call.Media(file)

	// Execute upload
	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}

	// Convert response to our model
	ytVideo := c.convertToYouTubeVideo(response)
	return &ytVideo, nil
}

// SearchVideos searches for videos on YouTube
func (c *Client) SearchVideos(ctx context.Context, req *dto.YouTubeSearchRequest) (*dto.YouTubeVideoResponse, error) {
	call := c.service.Search.List([]string{"id", "snippet"}).
		Q(req.Q).
		Type("video")

	if req.MaxResults > 0 {
		call = call.MaxResults(req.MaxResults)
	} else {
		call = call.MaxResults(25)
	}

	if req.PageToken != "" {
		call = call.PageToken(req.PageToken)
	}

	if req.Order != "" {
		call = call.Order(req.Order)
	}

	if req.ChannelID != "" {
		call = call.ChannelId(req.ChannelID)
	}

	if req.PublishedAfter != "" {
		if publishedAfter, err := time.Parse(time.RFC3339, req.PublishedAfter); err == nil {
			call = call.PublishedAfter(publishedAfter.Format(time.RFC3339))
		}
	}

	if req.PublishedBefore != "" {
		if publishedBefore, err := time.Parse(time.RFC3339, req.PublishedBefore); err == nil {
			call = call.PublishedBefore(publishedBefore.Format(time.RFC3339))
		}
	}

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to search videos: %w", err)
	}

	// Get video IDs for additional details
	var videoIDs []string
	for _, item := range response.Items {
		if item.Id.VideoId != "" {
			videoIDs = append(videoIDs, item.Id.VideoId)
		}
	}

	// Get video statistics and content details
	videos := make([]interface{}, 0)
	if len(videoIDs) > 0 {
		videoDetails, err := c.service.Videos.List([]string{"snippet", "statistics", "contentDetails", "status"}).
			Id(strings.Join(videoIDs, ",")).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get video details: %w", err)
		}

		for _, video := range videoDetails.Items {
			ytVideo := c.convertToYouTubeVideo(video)
			videos = append(videos, ytVideo)
		}
	}

	return &dto.YouTubeVideoResponse{
		YouTubeResponse: dto.YouTubeResponse{
			Kind:          response.Kind,
			ETag:          response.Etag,
			NextPageToken: response.NextPageToken,
			PrevPageToken: response.PrevPageToken,
			PageInfo: dto.PageInfo{
				TotalResults:   response.PageInfo.TotalResults,
				ResultsPerPage: response.PageInfo.ResultsPerPage,
			},
		},
		Items: videos,
	}, nil
}

// Placeholder implementations for other methods
// You'll need to implement these based on your specific requirements

func (c *Client) UpdateVideo(ctx context.Context, videoID string, updates map[string]interface{}) (*model.YouTubeVideo, error) {
	// Implementation for updating video metadata
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) DeleteVideo(ctx context.Context, videoID string) error {
	// Implementation for deleting video
	return fmt.Errorf("not implemented yet")
}

func (c *Client) GetVideoComments(ctx context.Context, req *dto.YouTubeCommentListRequest) (*dto.YouTubeCommentResponse, error) {
	// Implementation for getting video comments
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) AddComment(ctx context.Context, req *dto.YouTubeCommentRequest) (*model.YouTubeComment, error) {
	// Implementation for adding comment
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) UpdateComment(ctx context.Context, req *dto.YouTubeCommentUpdateRequest) (*model.YouTubeComment, error) {
	// Implementation for updating comment
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) DeleteComment(ctx context.Context, commentID string) error {
	// Implementation for deleting comment
	return fmt.Errorf("not implemented yet")
}

func (c *Client) LikeVideo(ctx context.Context, videoID string) error {
	// Implementation for liking video
	return fmt.Errorf("not implemented yet")
}

func (c *Client) DislikeVideo(ctx context.Context, videoID string) error {
	// Implementation for disliking video
	return fmt.Errorf("not implemented yet")
}

func (c *Client) RemoveVideoRating(ctx context.Context, videoID string) error {
	// Implementation for removing video rating
	return fmt.Errorf("not implemented yet")
}

func (c *Client) LikeComment(ctx context.Context, commentID string) error {
	// Implementation for liking comment
	return fmt.Errorf("not implemented yet")
}

func (c *Client) DislikeComment(ctx context.Context, commentID string) error {
	// Implementation for disliking comment
	return fmt.Errorf("not implemented yet")
}

func (c *Client) RemoveCommentRating(ctx context.Context, commentID string) error {
	// Implementation for removing comment rating
	return fmt.Errorf("not implemented yet")
}

func (c *Client) GetMyChannel(ctx context.Context) (*model.YouTubeChannel, error) {
	call := c.service.Channels.List([]string{"snippet", "statistics", "contentDetails"}).
		Mine(true)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get my channel: %w", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("no channel found for authenticated user")
	}

	channel := response.Items[0]
	publishedAt, _ := time.Parse(time.RFC3339, channel.Snippet.PublishedAt)

	ytChannel := &model.YouTubeChannel{
		ID:          channel.Id,
		Title:       channel.Snippet.Title,
		Description: channel.Snippet.Description,
		CustomURL:   channel.Snippet.CustomUrl,
		PublishedAt: publishedAt,
	}

	// Set thumbnails
	if channel.Snippet.Thumbnails != nil {
		if channel.Snippet.Thumbnails.Default != nil {
			ytChannel.Thumbnails.Default.URL = channel.Snippet.Thumbnails.Default.Url
		}
		if channel.Snippet.Thumbnails.Medium != nil {
			ytChannel.Thumbnails.Medium.URL = channel.Snippet.Thumbnails.Medium.Url
		}
		if channel.Snippet.Thumbnails.High != nil {
			ytChannel.Thumbnails.High.URL = channel.Snippet.Thumbnails.High.Url
		}
	}

	// Set statistics
	if channel.Statistics != nil {
		ytChannel.ViewCount = int64(channel.Statistics.ViewCount)
		ytChannel.SubscriberCount = int64(channel.Statistics.SubscriberCount)
		ytChannel.VideoCount = int64(channel.Statistics.VideoCount)
	}

	return ytChannel, nil
}

func (c *Client) GetChannelDetails(ctx context.Context, channelID string) (*model.YouTubeChannel, error) {
	// Implementation for getting channel details
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) GetChannelVideos(ctx context.Context, channelID string, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error) {
	// Implementation for getting channel videos
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) GetMyPlaylists(ctx context.Context) ([]model.YouTubePlaylist, error) {
	// Implementation for getting my playlists
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) GetPlaylistVideos(ctx context.Context, req *dto.YouTubePlaylistRequest) (*dto.YouTubeVideoResponse, error) {
	// Implementation for getting playlist videos
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) CreatePlaylist(ctx context.Context, title, description, privacy string) (*model.YouTubePlaylist, error) {
	// Implementation for creating playlist
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) AddVideoToPlaylist(ctx context.Context, playlistID, videoID string) error {
	// Implementation for adding video to playlist
	return fmt.Errorf("not implemented yet")
}

func (c *Client) RemoveVideoFromPlaylist(ctx context.Context, playlistID, videoID string) error {
	// Implementation for removing video from playlist
	return fmt.Errorf("not implemented yet")
}

func (c *Client) GetVideoAnalytics(ctx context.Context, videoID string, startDate, endDate string) (interface{}, error) {
	// Implementation for getting video analytics
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) GetChannelAnalytics(ctx context.Context, startDate, endDate string) (interface{}, error) {
	// Implementation for getting channel analytics
	return nil, fmt.Errorf("not implemented yet")
}
