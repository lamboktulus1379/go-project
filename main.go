package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"my-project/domain/model"
	"my-project/domain/repository"
	"my-project/infrastructure/cache"
	tulushost "my-project/infrastructure/clients/tulustech"
	youtubeclient "my-project/infrastructure/clients/youtube"
	"my-project/infrastructure/configuration"
	"my-project/infrastructure/filecsv"
	"my-project/infrastructure/googlesheet"
	"my-project/infrastructure/logger"
	"my-project/infrastructure/persistence"
	"my-project/infrastructure/pubsub"
	"my-project/infrastructure/realtime"
	"my-project/infrastructure/servicebus"
	httpHandler "my-project/interfaces/http"
	"my-project/interfaces/middleware"
	"my-project/server"
	"my-project/usecase"

	"github.com/gin-gonic/gin"

	"golang.org/x/sync/errgroup"
)

var httpServer *http.Server

func recoverPanic() {
	if err := recover(); err != nil {
		logger.GetLogger().WithField("error", err).Error("Application panic recovered")
	}
}

func main() {
	// InitiateGoroutine()
	defer recoverPanic()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	g, ctx := errgroup.WithContext(ctx)

	// Load env from files (non-destructive; OS env still has precedence)
	configuration.LoadEnvFromFile("config.env", ".env")
	// Log which env files are present to help diagnose prod config loading
	if _, err := os.Stat("config.env"); err == nil {
		logger.GetLogger().Info("Detected config.env in working directory")
	} else {
		logger.GetLogger().Info("config.env not found in working directory")
	}
	if _, err := os.Stat(".env"); err == nil {
		logger.GetLogger().Info("Detected .env in working directory")
	} else {
		logger.GetLogger().Info(".env not found in working directory")
	}
	// configuration.LoadConfig()

	app := configuration.C.App

	mysqlDb, psqlDb, err := InitiateDatabase()
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Database initialization failed")
	}

	mongoDb, err := persistence.NewMongoDb(
		configuration.C.Database.Mongo.Host,
		configuration.C.Database.Mongo.Port,
		configuration.C.Database.Mongo.User,
		configuration.C.Database.Mongo.Password,
		configuration.C.Database.Mongo.Name,
	)
	if err != nil {
		logger.GetLogger().WithField("error", err).Warn("MongoDB not available - continuing without Mongo features")
		mongoDb = nil
	} else {
		if err := mongoDb.Ping(ctx, nil); err != nil {
			logger.GetLogger().WithField("error", err).Warn("MongoDB ping failed - continuing without Mongo features")
			mongoDb = nil
		} else {
			logger.GetLogger().Info("MongoDB connected successfully")
		}
	}

	// Log DB connectivity safely
	var psqlPing interface{}
	if psqlDb != nil {
		psqlPing = psqlDb.Ping()
	} else {
		psqlPing = "nil"
	}
	logger.GetLogger().
		WithField("PrimaryDB", mysqlDb.Ping()).
		WithField("PSQLDb", psqlPing).
		Info("Database connected.")

	pubSubClient, err := pubsub.NewPubSub(ctx, configuration.C.Pubsub.ProjectID)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while instantiate PubSub")
		pubSubClient = nil // Set to nil for graceful handling
	}

	azServiceBusClient, err := servicebus.NewServiceBus(ctx, configuration.C.ServiceBus.Namespace)
	if err != nil {
		logger.GetLogger().WithField("error", err).Warn("Azure Service Bus not available - continuing without Service Bus features")
		azServiceBusClient = nil
	}
	redisClient, _ := cache.NewCache(
		ctx,
		fmt.Sprintf("%s:%s", configuration.C.RedisClient.Host, configuration.C.RedisClient.Port),
		configuration.C.RedisClient.Username,
		configuration.C.RedisClient.Password,
	)

	testRepository := persistence.NewTestRepository(mongoDb, psqlDb)
	project, err := testRepository.Test(ctx)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while fetching data")
	}
	logger.GetLogger().WithField("project", project).Info("Test project data retrieved")
	testCache := cache.NewTestCache(redisClient)

	logger.GetLogger().Info("Redis client initialized successfully.")

	tulusTechHost := tulushost.NewTulusHost(configuration.C.TulusTech.Host)

	testPubSub := pubsub.NewTestPubSub(pubSubClient)
	testServiceBus := servicebus.NewTestServiceBus(azServiceBusClient)

	// Repository wiring: use MSSQL in production, otherwise PostgreSQL.
	var userRepository repository.IUser
	if psqlDb == nil { // production/MSSQL path from InitiateDatabase
		userRepository = persistence.NewUserRepositoryMSSQL(mysqlDb)
	} else {
		userRepository = persistence.NewUserRepository(psqlDb)
	}
	userUsecase := usecase.NewUserUsecase(userRepository)
	testUsecase := usecase.NewTestUsecase(tulusTechHost, testPubSub, testServiceBus, testCache)

	// Initialize YouTube components
	youtubeConfig, err := configuration.GetYouTubeConfig()
	if err != nil {
		logger.GetLogger().WithField("error", err).Warn("YouTube configuration not found - YouTube features will be disabled")
		// Continue without YouTube functionality
	}

	if youtubeConfig != nil {
		logger.GetLogger().WithFields(map[string]interface{}{
			"hasAccessToken":  youtubeConfig.AccessToken != "" && youtubeConfig.AccessToken != "your_access_token_here",
			"hasRefreshToken": youtubeConfig.RefreshToken != "" && youtubeConfig.RefreshToken != "your_refresh_token_here",
			"hasAPIKey":       youtubeConfig.APIKey != "" && youtubeConfig.APIKey != "YOUR_YOUTUBE_API_KEY",
			"channelIDSet":    youtubeConfig.ChannelID != "",
			"clientIDSet":     youtubeConfig.ClientID != "" && youtubeConfig.ClientID != "your_client_id_here",
		}).Info("Loaded YouTube configuration state")
	} else {
		logger.GetLogger().Info("YouTube configuration struct is nil (no config file loaded)")
	}

	var youtubeHandler httpHandler.IYouTubeHandler
	var youtubeAuthHandler httpHandler.IYouTubeAuthHandler
	var youtubeClient repository.IYouTube // keep reference for share enrichment

	// Always try to initialize YouTube auth handler (doesn't require tokens)
	youtubeAuthHandler, err = httpHandler.NewYouTubeAuthHandler()
	if err != nil {
		logger.GetLogger().WithField("error", err).Warn("Failed to initialize YouTube auth handler")
		youtubeAuthHandler = nil
	}

	// Respect an explicit switch to force mock-only mode regardless of credentials
	forceMockMode := os.Getenv("YOUTUBE_MODE") == "mock" || os.Getenv("YOUTUBE_MODE") == "disabled" || os.Getenv("YOUTUBE_ENABLED") == "false"
	if forceMockMode {
		logger.GetLogger().WithFields(map[string]interface{}{
			"YOUTUBE_MODE":    os.Getenv("YOUTUBE_MODE"),
			"YOUTUBE_ENABLED": os.Getenv("YOUTUBE_ENABLED"),
		}).Info("Forcing mock-only mode for YouTube: skipping real YouTube client initialization")
	} else if youtubeConfig != nil &&
		((youtubeConfig.AccessToken != "" && youtubeConfig.AccessToken != "your_access_token_here") ||
			(youtubeConfig.APIKey != "" && youtubeConfig.APIKey != "YOUR_YOUTUBE_API_KEY")) {

		logger.GetLogger().Info("Attempting to initialize YouTube client (tokens or API key present)")

		// Convert configuration to YouTube client config
		youtubeClientConfig := &youtubeclient.Config{
			ClientID:     youtubeConfig.ClientID,
			ClientSecret: youtubeConfig.ClientSecret,
			RedirectURL:  youtubeConfig.RedirectURL,
			AccessToken:  youtubeConfig.AccessToken,
			RefreshToken: youtubeConfig.RefreshToken,
			ChannelID:    youtubeConfig.ChannelID,
			APIKey:       youtubeConfig.APIKey,
		}

		// Initialize YouTube client
		youtubeClient, err = youtubeclient.NewYouTubeClient(ctx, youtubeClientConfig)
		if err != nil {
			logger.GetLogger().WithField("error", err).Warn("Failed to initialize YouTube client - YouTube features will be disabled")
		} else {
			// Ensure cache schema and attach cache repository
			if psqlDb != nil {
				if err := persistence.EnsureYouTubeCacheSchema(psqlDb); err != nil {
					logger.GetLogger().WithField("error", err).Error("failed ensuring youtube cache schema")
				}
			}
			ytCache := persistence.NewYouTubeCacheRepository(psqlDb)
			// Create YouTubeRepository that combines API client and cache
			youtubeRepo := &persistence.YouTubeRepository{
				CacheRepo:        ytCache,
				YouTubeAPIClient: youtubeClient,
			}
			youtubeUC := usecase.NewYouTubeUseCaseWithCache(youtubeRepo, ytCache)
			youtubeHandler = httpHandler.NewYouTubeHandler(youtubeUC)
			logger.GetLogger().Info("YouTubeRepository initialized successfully with API client and DB cache; registering YouTube routes including PATCH /api/youtube/videos/:videoId")
		}
	} else {
		logger.GetLogger().Info("YouTube API credentials not configured - YouTube features will be disabled (using mock data only; PATCH route will return 501 fallback)")
	}

	// Summarize effective YouTube mode for quick visibility
	effectiveMode := "mock"
	if forceMockMode {
		effectiveMode = "disabled"
	} else if youtubeClient != nil {
		effectiveMode = "live"
	}
	logger.GetLogger().WithFields(map[string]interface{}{
		"effectiveMode": effectiveMode,
		"handlerActive": youtubeHandler != nil,
	}).Info("YouTube initialization summary")

	userHandler := httpHandler.NewUserHandler(userUsecase)
	testHandler := httpHandler.NewTestHandler(testUsecase)

	// Share feature wiring (PostgreSQL only for now)
	var shareHandler httpHandler.IShareHandler
	shareHub := realtime.NewShareHub()
	if psqlDb != nil {
		shareRepo := persistence.NewShareRepository(psqlDb)
		oauthRepo := persistence.NewOAuthTokenRepository(psqlDb)
		if err := persistence.EnsureOAuthTokenSchema(psqlDb); err != nil {
			logger.GetLogger().WithField("error", err).Error("failed ensuring oauth token schema")
		}
		if err := persistence.EnsureShareSchema(psqlDb); err != nil {
			logger.GetLogger().WithField("error", err).Error("failed ensuring share schema (external_ref columns)")
		}
		if len(configuration.C.Share.Platforms) == 0 {
			configuration.C.Share.Platforms = []string{"twitter", "facebook", "whatsapp"}
		}
		if youtubeClient != nil {
			shareUC := usecase.NewShareUsecase(shareRepo, oauthRepo, configuration.C.Share.Platforms, youtubeClient)
			shareUC = shareUC.WithBroadcaster(func(rec *model.VideoShareRecord) { shareHub.BroadcastShareStatus(rec) })
			shareHandler = httpHandler.NewShareHandler(shareUC, configuration.C.Share.Platforms)
		} else {
			shareUC := usecase.NewShareUsecase(shareRepo, oauthRepo, configuration.C.Share.Platforms)
			shareUC = shareUC.WithBroadcaster(func(rec *model.VideoShareRecord) { shareHub.BroadcastShareStatus(rec) })
			shareHandler = httpHandler.NewShareHandler(shareUC, configuration.C.Share.Platforms)
		}
	} else {
		logger.GetLogger().Info("PostgreSQL not available in this environment; Share feature disabled")
	}

	// Facebook OAuth handler (uses PostgreSQL-backed token repo); only when Postgres is available
	var facebookOAuthHandler httpHandler.IFacebookOAuthHandler
	if psqlDb != nil {
		facebookOAuthHandler = httpHandler.NewFacebookOAuthHandler(persistence.NewOAuthTokenRepository(psqlDb))
	}

	router := server.InitiateRouter(userHandler, testHandler, youtubeHandler, youtubeAuthHandler, userRepository, shareHandler, facebookOAuthHandler)

	// SSE endpoint for real-time share status
	if shareHandler != nil {
		// Re-create group with auth middleware so user_id is populated (previous code lacked middleware causing 401)
		api := router.Group("api")
		api.Use(middleware.Auth(userRepository))
		api.GET("/share/stream", func(c *gin.Context) { shareHub.Serve(c) })
	}

	// Background share job processor (simple ticker loop)
	if shareHandler != nil && psqlDb != nil {
		// Recreate repos for job processor scope
		shareRepo := persistence.NewShareRepository(psqlDb)
		oauthRepo := persistence.NewOAuthTokenRepository(psqlDb)
		g.Go(func() error {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					procCtx, cancelProc := context.WithTimeout(ctx, 5*time.Second)
					_ = usecase.ProcessShareJobs(procCtx, shareRepo, oauthRepo, youtubeClient, 10, func(rec *model.VideoShareRecord) {
						shareHub.BroadcastShareStatus(rec)
					})
					cancelProc()
				}
			}
		})
	}

	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while StartSubscription")
	}

	// Comment out Test() function to prevent Google Sheets OAuth blocking
	// Test()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	port := app.Port
	logger.GetLogger().WithFields(map[string]interface{}{"port": port, "tls": app.TLSEnabled}).Info("Starting application")
	g.Go(func() error {
		httpServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      router,
			ReadTimeout:  0,
			WriteTimeout: 0,
		}
		// Keep reference for graceful shutdown
		// (re-use package-level httpServer variable if needed elsewhere)
		if app.TLSEnabled {
			cert := app.TLSCertFile
			key := app.TLSKeyFile
			if cert == "" || key == "" {
				logger.GetLogger().Error("TLS enabled but cert or key path empty; falling back to HTTP")
				if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
					return err
				}
			} else {
				logger.GetLogger().WithFields(map[string]interface{}{"cert": cert, "key": key}).Info("Serving HTTPS")
				if err := httpServer.ListenAndServeTLS(cert, key); !errors.Is(err, http.ErrServerClosed) {
					return err
				}
			}
		} else {
			if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		}
		return nil
	})

	select {
	case <-interrupt:
		logger.GetLogger().Info("Application shutdown requested")
		os.Exit(1)
	case <-ctx.Done():
		break
	}

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if httpServer != nil {
		_ = httpServer.Shutdown(shutdownCtx)
	}

	err = g.Wait()
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Server returned an error")
		os.Exit(2)
	}
}

func InitiateDatabase() (*sql.DB, *sql.DB, error) {
	// Contract: return (primaryDB, psqlDB). In production, primaryDB = MSSQL and psqlDB may be nil.
	// Locally, primaryDB = MySQL native (existing) and psqlDB = PostgreSQL.
	env := os.Getenv("ENV")
	// Allow overriding vendor explicitly (e.g., DB_VENDOR=mssql) for local tests against Typing's docker-compose
	if v := os.Getenv("DB_VENDOR"); v == "mssql" {
		mssql, err := persistence.NewMSSQLDB()
		if err != nil {
			logger.GetLogger().WithField("error", err).Error("Cannot connect to MSSQL (DB_VENDOR=mssql)")
			return nil, nil, err
		}
		return mssql, nil, nil
	}
	if env == "production" || env == "prod" {
		mssql, err := persistence.NewMSSQLDB()
		if err != nil {
			logger.GetLogger().WithField("error", err).Error("Cannot connect to MSSQL (production)")
			return nil, nil, err
		}
		// In production we don’t require local PostgreSQL; repositories that need Postgres-only features should be adjusted by caller.
		return mssql, nil, nil
	}

	// Default/local: keep current behavior (MySQL native + PostgreSQL)
	db, err := persistence.NewNativeDb()
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Cannot connect to the local database")
		return nil, nil, err
	}
	postgres, err := persistence.NewPostgreSQLDB()
	if err != nil {
		return nil, nil, err
	}
	return db, postgres, nil
}

func InitiateGoroutine() {
	logger.GetLogger().Info("Initializing goroutines")

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.GetLogger().WithField("goroutine_id", id).Debug("Goroutine started")
		}(i)
	}
}

func Test() {
	file, err := filecsv.NewFile("cover.txt")
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while loading file")
	}

	validateCsv := filecsv.NewValidateCsv(file)
	logger.GetLogger().WithField("validateCsv", validateCsv).Info("Validate CSV initialized")

	googleSheet, err := googlesheet.NewGoogleSheet()
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while loading Google Sheet")
	}

	logger.GetLogger().WithField("googleSheet", googleSheet).Info("Google sheet initialized")
}
