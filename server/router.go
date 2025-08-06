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
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200", "http://localhost:3000", "https://tulus.tech"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Authorization"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			// Allow localhost for development
			return origin == "http://localhost:4200" || 
				   origin == "http://localhost:3000" || 
				   origin == "https://tulus.tech"
		},
		MaxAge: 12 * time.Hour,
	}))

	api := router.Group("api")
	api.Use(middleware.Auth(userRepository))

	router.POST("/login", userHandler.Login)
	router.POST("/register", userHandler.Register)

	// YouTube OAuth2 authentication routes (only if handler is available)
	if youtubeAuthHandler != nil {
		router.GET("/auth/youtube", youtubeAuthHandler.GetAuthURL)
		router.GET("/auth/youtube/callback", youtubeAuthHandler.HandleCallback)
	}

	// Temporary test route for YouTube API (bypasses authentication)
	if youtubeHandler != nil {
		router.GET("/test/youtube/videos", youtubeHandler.GetMyVideos)
		router.GET("/test/youtube/search", youtubeHandler.SearchVideos)
		router.GET("/test/youtube/channel", youtubeHandler.GetMyChannel)
	}

	router.POST("/healthz", testHandler.Test)

	api.POST("/", func(ctx *gin.Context) {
		res := ctx.Request.Body
		ctx.JSON(http.StatusOK, res)
	})

	// YouTube API routes (only if handler is available)
	if youtubeHandler != nil {
		youtube := api.Group("/youtube")
		{
			// Video operations
			youtube.GET("/videos", youtubeHandler.GetMyVideos)
			youtube.GET("/videos/:videoId", youtubeHandler.GetVideoDetails)
			youtube.POST("/videos/upload", youtubeHandler.UploadVideo)
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
			youtube.POST("/comments/:commentId/like", youtubeHandler.LikeComment)

			// Channel operations
			youtube.GET("/channel", youtubeHandler.GetMyChannel)
			youtube.GET("/channels/:channelId", youtubeHandler.GetChannelDetails)

			// Playlist operations
			youtube.GET("/playlists", youtubeHandler.GetMyPlaylists)
			youtube.POST("/playlists", youtubeHandler.CreatePlaylist)
		}
	} else {
		// Add info endpoint when YouTube is not configured
		api.GET("/youtube/info", func(ctx *gin.Context) {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{
				"error":       "YouTube API not configured",
				"message":     "Please configure YouTube API credentials to enable YouTube features",
				"setup_guide": "/docs/YOUTUBE_API_SETUP.md",
			})
		})
	}

	return router
}
