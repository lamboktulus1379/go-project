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
	oauthConfig *oauth2.Config
	token       *oauth2.Token
	ctx         context.Context
}

// Config represents YouTube API configuration
type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ChannelID    string `json:"channel_id"`
	APIKey       string `json:"api_key"`
}

// NewYouTubeClient creates a new YouTube API client
func NewYouTubeClient(ctx context.Context, config *Config) (repository.IYouTube, error) {
	// If we don't have OAuth credentials but we do have an API key, use API key only mode (read-only)
	if (config.AccessToken == "" || config.RefreshToken == "") && config.APIKey != "" {
		service, err := youtube.NewService(ctx, option.WithAPIKey(config.APIKey))
		if err != nil {
			return nil, fmt.Errorf("failed to create YouTube service with API key: %w", err)
		}
		return &Client{
			service:     service,
			channelID:   config.ChannelID,
			accessToken: "", // no bearer token
			oauthConfig: nil,
			token:       nil,
			ctx:         ctx,
		}, nil
	}

	// Full OAuth2 mode
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

	token := &oauth2.Token{
		AccessToken:  config.AccessToken,
		RefreshToken: config.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-1 * time.Minute), // Force refresh on first use
	}

	httpClient := oauth2Config.Client(ctx, token)
	service, err := youtube.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube service: %w", err)
	}

	return &Client{
		service:     service,
		channelID:   config.ChannelID,
		accessToken: config.AccessToken,
		oauthConfig: oauth2Config,
		token:       token,
		ctx:         ctx,
	}, nil
}

// GetMyVideos retrieves videos from the authenticated user's channel
func (c *Client) GetMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error) {
	// If no OAuth (API key mode) skip refresh
	if c.oauthConfig != nil && c.token != nil {
		if err := c.refreshTokenIfNeeded(); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	// Fallback mock data when channel ID not set
	channelID := c.channelID
	if req.ChannelID != "" {
		channelID = req.ChannelID
	}
	if channelID == "" {
		mock := &dto.YouTubeVideoResponse{
			YouTubeResponse: dto.YouTubeResponse{
				Kind:     "youtube#searchListResponse",
				PageInfo: dto.PageInfo{TotalResults: 2, ResultsPerPage: 2},
			},
			Items: []interface{}{
				map[string]interface{}{"id": "mock-video-1", "title": "Configure YOUTUBE_CHANNEL_ID", "description": "Set YOUTUBE_CHANNEL_ID env or add to config.json", "view_count": 0, "like_count": 0},
				map[string]interface{}{"id": "mock-video-2", "title": "Using API Key mode", "description": "Provide access & refresh token for authenticated channel data", "view_count": 0, "like_count": 0},
			},
		}
		return mock, nil
	}

	call := c.service.Search.List([]string{"id", "snippet"}).
		ChannelId(channelID).
		Type("video").
		Order("date")

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

	var videoIDs []string
	for _, item := range response.Items {
		videoIDs = append(videoIDs, item.Id.VideoId)
	}

	videos := make([]interface{}, 0)
	if len(videoIDs) > 0 {
		videoDetails, err := c.service.Videos.List([]string{"snippet", "statistics", "contentDetails", "status"}).Id(strings.Join(videoIDs, ",")).Do()
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
			PageInfo:      dto.PageInfo{TotalResults: response.PageInfo.TotalResults, ResultsPerPage: response.PageInfo.ResultsPerPage},
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
	if videoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates provided")
	}

	// Need OAuth for update; API key mode insufficient
	if c.oauthConfig == nil || c.token == nil {
		return nil, fmt.Errorf("video update requires OAuth credentials (access + refresh token)")
	}
	if err := c.refreshTokenIfNeeded(); err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	// Fetch existing video to preserve unchanged fields
	existingResp, err := c.service.Videos.List([]string{"snippet", "status"}).Id(videoID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing video: %w", err)
	}
	if len(existingResp.Items) == 0 {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}
	existing := existingResp.Items[0]

	// Apply updates
	if title, ok := updates["title"].(string); ok {
		existing.Snippet.Title = title
	}
	if desc, ok := updates["description"].(string); ok {
		existing.Snippet.Description = desc
	}
	if tags, ok := updates["tags"].([]string); ok {
		existing.Snippet.Tags = tags
	}
	if cat, ok := updates["category"].(string); ok {
		existing.Snippet.CategoryId = cat
	}
	if privacy, ok := updates["privacy"].(string); ok {
		if existing.Status == nil { existing.Status = &youtube.VideoStatus{} }
		existing.Status.PrivacyStatus = privacy
	}

	// Perform update call (videos.update requires both snippet & status parts when modifying those fields)
	updatedResp, err := c.service.Videos.Update([]string{"snippet", "status"}, existing).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update video: %w", err)
	}

	ytVideo := c.convertToYouTubeVideo(updatedResp)
	return &ytVideo, nil
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

// refreshTokenIfNeeded checks if the token is expired and refreshes it automatically
func (c *Client) refreshTokenIfNeeded() error {
	// In API key mode (no oauthConfig/token) nothing to do
	if c.oauthConfig == nil || c.token == nil {
		return nil
	}
	if c.token.Expiry.IsZero() || time.Until(c.token.Expiry) < 5*time.Minute {
		newToken, err := c.oauthConfig.TokenSource(c.ctx, c.token).Token()
		if err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}
		c.token = newToken
		c.accessToken = newToken.AccessToken
		httpClient := c.oauthConfig.Client(c.ctx, newToken)
		service, err := youtube.NewService(c.ctx, option.WithHTTPClient(httpClient))
		if err != nil {
			return fmt.Errorf("failed to recreate YouTube service with refreshed token: %w", err)
		}
		c.service = service
		fmt.Printf("Token refreshed successfully. New expiry: %v\n", newToken.Expiry)
	}
	return nil
}

func (c *Client) GetChannelAnalytics(ctx context.Context, startDate, endDate string) (interface{}, error) {
	// Implementation for getting channel analytics
	return nil, fmt.Errorf("not implemented yet")
}
