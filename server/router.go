package server

import (
	"net/http"
	"time"

	"my-project/domain/repository"
	httpHandler "my-project/interfaces/http"
	"my-project/interfaces/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func InitiateRouter(
	userHandler httpHandler.IUserHandler,
	testHandler httpHandler.ITestHandler,
	youtubeHandler httpHandler.IYouTubeHandler,
	youtubeAuthHandler httpHandler.IYouTubeAuthHandler,
	userRepository repository.IUser,
	shareHandler httpHandler.IShareHandler,
	facebookOAuthHandler httpHandler.IFacebookOAuthHandler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://tulus.tech", "https://admin.tulus.tech", "http://localhost:4201", "http://localhost:4200", "https://localhost:4201", "https://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return origin == "https://tulus.tech" || origin == "https://admin.tulus.tech" || origin == "http://localhost:4201" || origin == "http://localhost:4200" || origin == "https://localhost:4201" || origin == "https://localhost:4200"
		},
		MaxAge: 12 * time.Hour,
	}))

	api := router.Group("api")
	api.Use(middleware.Auth(userRepository))

	router.POST("/login", userHandler.Login)
	router.POST("/register", userHandler.Register)

	// OAuth authentication routes
	if youtubeAuthHandler != nil {
		router.GET("/auth/youtube", youtubeAuthHandler.GetAuthURL)
		router.GET("/auth/youtube/callback", youtubeAuthHandler.HandleCallback)
		api.GET("/youtube/oauth/status", youtubeAuthHandler.Status)
	}
	if facebookOAuthHandler != nil {
		router.GET("/auth/facebook", facebookOAuthHandler.GetAuthURL)
		router.GET("/auth/facebook/callback", facebookOAuthHandler.Callback)
		api.GET("/facebook/status", facebookOAuthHandler.Status)
		api.POST("/facebook/refresh-pages", facebookOAuthHandler.RefreshPages)
		api.POST("/facebook/link-page", facebookOAuthHandler.LinkPage)
		api.POST("/facebook/link-page-url", facebookOAuthHandler.LinkPageURL)
	}

	// Temporary test route for YouTube API (bypasses authentication)
	if youtubeHandler != nil {
		router.GET("/test/youtube/videos", youtubeHandler.GetMyVideos)
		router.GET("/test/youtube/search", youtubeHandler.SearchVideos)
		router.GET("/test/youtube/channel", youtubeHandler.GetMyChannel)
	} else {
		// Fallback mock endpoint if YouTube handler is not available
		router.GET("/test/youtube/videos", func(ctx *gin.Context) {
			ctx.Header("Access-Control-Allow-Origin", "*")
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			ctx.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")

			ctx.JSON(http.StatusOK, gin.H{
				"error":   false,
				"message": "YouTube API not configured - returning mock data",
				"data": []gin.H{
					{
						"id":          "mock-video-1",
						"title":       "Sample Video 1",
						"description": "This is a mock video for testing",
						"thumbnail":   "https://placehold.co/320x180/png?text=Mock+Video+1",
						"viewCount":   "1234",
						"likeCount":   "56",
						"publishedAt": "2025-08-06T00:00:00Z",
					},
					{
						"id":          "mock-video-2",
						"title":       "Sample Video 2",
						"description": "Another mock video for testing",
						"thumbnail":   "https://placehold.co/320x180/png?text=Mock+Video+2",
						"viewCount":   "5678",
						"likeCount":   "89",
						"publishedAt": "2025-08-05T00:00:00Z",
					},
				},
			})
		})
	}

	router.POST("/healthz", testHandler.Test)

	api.POST("/", func(ctx *gin.Context) {
		res := ctx.Request.Body
		ctx.JSON(http.StatusOK, res)
	})

	// Share platform capability endpoint (not tied to YouTube availability)
	if shareHandler != nil {
		api.GET("/share/platforms", shareHandler.GetPlatforms)
		api.POST("/share/process-jobs", shareHandler.ProcessJobs)
	}

	// YouTube API routes (only if handler is available)
	if youtubeHandler != nil {
		youtube := api.Group("/youtube")
		{
			// Video operations
			youtube.GET("/videos", youtubeHandler.GetMyVideos)
			youtube.POST("/sync", youtubeHandler.SyncMyVideos)
			youtube.GET("/videos/:videoId", youtubeHandler.GetVideoDetails)
			youtube.POST("/videos/upload", youtubeHandler.UploadVideo)
			youtube.PATCH("/videos/:videoId", youtubeHandler.UpdateVideo)
			youtube.GET("/search", youtubeHandler.SearchVideos)

			// Video rating operations
			youtube.POST("/videos/:videoId/like", youtubeHandler.LikeVideo)
			youtube.POST("/videos/:videoId/dislike", youtubeHandler.DislikeVideo)
			youtube.DELETE("/videos/:videoId/rating", youtubeHandler.RemoveVideoRating)

			// Comment operations
			youtube.GET("/videos/:videoId/comments", youtubeHandler.GetVideoComments)
			youtube.POST("/comments", youtubeHandler.AddComment)
			youtube.PUT("/comments/:commentId", youtubeHandler.UpdateComment)
			youtube.DELETE("/comments/:commentId", youtubeHandler.DeleteComment)
			youtube.POST("/comments/:commentId/like", youtubeHandler.ToggleCommentLike)
			youtube.POST("/comments/:commentId/heart", youtubeHandler.ToggleCommentHeart)

			// Channel operations
			youtube.GET("/channel", youtubeHandler.GetMyChannel)
			youtube.GET("/channels/:channelId", youtubeHandler.GetChannelDetails)

			// Playlist operations
			youtube.GET("/playlists", youtubeHandler.GetMyPlaylists)
			youtube.POST("/playlists", youtubeHandler.CreatePlaylist)
			// Share endpoints (video social sharing)
			youtube.POST("/videos/:videoId/share", func(c *gin.Context) {
				if shareHandler != nil {
					shareHandler.ShareVideo(c)
					return
				}
				c.JSON(http.StatusNotImplemented, gin.H{"error": "share handler not configured"})
			})
			youtube.GET("/videos/:videoId/share-status", func(c *gin.Context) {
				if shareHandler != nil {
					shareHandler.GetShareStatus(c)
					return
				}
				c.JSON(http.StatusNotImplemented, gin.H{"error": "share handler not configured"})
			})
		}
	} else {
		// Add fallback endpoints when YouTube is not configured
		youtube := api.Group("/youtube")
		{
			youtube.GET("/videos", func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{
					"error":   false,
					"message": "YouTube API not configured - returning mock data",
					"data": []gin.H{
						{
							"id":          "mock-video-1",
							"title":       "Sample Video 1",
							"description": "This is a mock video for testing",
							"thumbnail":   "https://placehold.co/320x180/png?text=Mock+Video",
							"viewCount":   "1234",
							"likeCount":   "56",
							"publishedAt": "2025-08-06T00:00:00Z",
						},
						{
							"id":          "mock-video-2",
							"title":       "Sample Video 2",
							"description": "Another mock video for testing",
							"thumbnail":   "https://placehold.co/320x180/png?text=Mock+Video+2",
							"viewCount":   "5678",
							"likeCount":   "89",
							"publishedAt": "2025-08-05T00:00:00Z",
						},
					},
				})
			})

			youtube.GET("/videos/:videoId", func(ctx *gin.Context) {
				videoId := ctx.Param("videoId")
				ctx.JSON(http.StatusOK, gin.H{
					"error":   false,
					"message": "YouTube API not configured - returning mock data",
					"data": gin.H{
						"id":          videoId,
						"title":       "Mock Video Details",
						"description": "This is a mock video detail for testing",
						"thumbnail":   "https://placehold.co/320x180/png?text=Mock+Thumb",
						"viewCount":   "1234",
						"likeCount":   "56",
						"publishedAt": "2025-08-06T00:00:00Z",
						"url":         "https://www.youtube.com/watch?v=" + videoId,
					},
				})
			})

			// Fallback for update route to avoid 404 and provide clear guidance
			youtube.PATCH("/videos/:videoId", func(ctx *gin.Context) {
				ctx.JSON(http.StatusNotImplemented, gin.H{
					"error":         true,
					"message":       "Video update not available - YouTube API not fully configured (OAuth credentials required)",
					"hint":          "Provide access & refresh tokens or complete OAuth flow at /auth/youtube to enable editing",
					"video_id":      ctx.Param("videoId"),
					"documentation": "/docs/YOUTUBE_API_SETUP.md",
				})
			})

			youtube.GET("/channel", func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{
					"error":   false,
					"message": "YouTube API not configured - returning mock data",
					"data": gin.H{
						"id":              "mock-channel-id",
						"title":           "Mock Channel",
						"description":     "This is a mock channel for testing",
						"subscriberCount": "1000",
						"viewCount":       "50000",
						"videoCount":      "25",
					},
				})
			})

			youtube.GET("/search", func(ctx *gin.Context) {
				query := ctx.Query("q")
				ctx.JSON(http.StatusOK, gin.H{
					"error":   false,
					"message": "YouTube API not configured - returning mock search results",
					"query":   query,
					"data": []gin.H{
						{
							"id":          "search-result-1",
							"title":       "Search Result 1 for: " + query,
							"description": "Mock search result",
							"thumbnail":   "https://via.placeholder.com/320x180",
							"viewCount":   "999",
						},
					},
				})
			})

			// For other endpoints, return a "not configured" message
			youtube.GET("/info", func(ctx *gin.Context) {
				ctx.JSON(http.StatusServiceUnavailable, gin.H{
					"error":       "YouTube API not configured",
					"message":     "Please configure YouTube API credentials to enable YouTube features",
					"setup_guide": "/docs/YOUTUBE_API_SETUP.md",
				})
			})
		}
	}

	return router
}
