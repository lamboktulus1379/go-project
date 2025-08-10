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

	"my-project/infrastructure/cache"
	tulushost "my-project/infrastructure/clients/tulustech"
	youtubeclient "my-project/infrastructure/clients/youtube"
	"my-project/infrastructure/configuration"
	"my-project/infrastructure/filecsv"
	"my-project/infrastructure/googlesheet"
	"my-project/infrastructure/logger"
	"my-project/infrastructure/persistence"
	"my-project/infrastructure/pubsub"
	"my-project/infrastructure/servicebus"
	"my-project/domain/repository"
	httpHandler "my-project/interfaces/http"
	"my-project/server"
	"my-project/usecase"

	"golang.org/x/sync/errgroup"
)

var httpServer *http.Server

func recoverPanic() {
	if err := recover(); err != nil {
		logger.GetLogger().WithField("error", err).Error("Application panic recovered")
	}
}

func main() {
	InitiateGoroutine()
	defer recoverPanic()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	g, ctx := errgroup.WithContext(ctx)

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
		logger.GetLogger().WithField("error", err).Error("Error while instantiate MongoDB")
		panic(err)
	}
	err = mongoDb.Ping(ctx, nil)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while Ping MongoDB")
		panic(err)
	}
	logger.GetLogger().Info("MongoDB connected successfully")

	logger.GetLogger().
		WithField("MySQLDb", mysqlDb.Ping()).
		WithField("PSQLDb", psqlDb.Ping()).
		Info("Database connected.")

	pubSubClient, err := pubsub.NewPubSub(ctx, configuration.C.Pubsub.ProjectID)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while instantiate PubSub")
		pubSubClient = nil // Set to nil for graceful handling
	}

	azServiceBusClient, err := servicebus.NewServiceBus(ctx, configuration.C.ServiceBus.Namespace)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while instantiate ServiceBus")
		panic(err)
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

	userRepository := persistence.NewUserRepository(psqlDb)
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

	// Initialize YouTube client if we have either access tokens OR API key
	if youtubeConfig != nil &&
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
			// Initialize YouTube use case and handler (assign to outer variable)
			youtubeUsecase := usecase.NewYouTubeUseCase(youtubeClient)
			youtubeHandler = httpHandler.NewYouTubeHandler(youtubeUsecase)
			logger.GetLogger().Info("YouTube API client initialized successfully; registering YouTube routes including PATCH /api/youtube/videos/:videoId")
		}
	} else {
		logger.GetLogger().Info("YouTube API credentials not configured - YouTube features will be disabled (using mock data only; PATCH route will return 501 fallback)")
	}

	userHandler := httpHandler.NewUserHandler(userUsecase)
	testHandler := httpHandler.NewTestHandler(testUsecase)

	// Share feature wiring (now using PostgreSQL DB)
	shareRepo := persistence.NewShareRepository(psqlDb)
	// Use PostgreSQL for OAuth tokens (queries use $1 style placeholders)
	oauthRepo := persistence.NewOAuthTokenRepository(psqlDb)
	if err := persistence.EnsureOAuthTokenSchema(psqlDb); err != nil {
		logger.GetLogger().WithField("error", err).Error("failed ensuring oauth token schema")
	}
	if err := persistence.EnsureShareSchema(psqlDb); err != nil {
		logger.GetLogger().WithField("error", err).Error("failed ensuring share schema (external_ref columns)")
	}
	var shareHandler httpHandler.IShareHandler
	if len(configuration.C.Share.Platforms) == 0 {
		configuration.C.Share.Platforms = []string{"twitter", "facebook", "whatsapp"}
	}
	if youtubeClient != nil {
		shareUsecase := usecase.NewShareUsecase(shareRepo, oauthRepo, configuration.C.Share.Platforms, youtubeClient)
		shareHandler = httpHandler.NewShareHandler(shareUsecase, configuration.C.Share.Platforms)
	} else {
		shareUsecase := usecase.NewShareUsecase(shareRepo, oauthRepo, configuration.C.Share.Platforms)
		shareHandler = httpHandler.NewShareHandler(shareUsecase, configuration.C.Share.Platforms)
	}


	// Facebook OAuth handler (uses same oauth token repo)
	facebookOAuthHandler := httpHandler.NewFacebookOAuthHandler(oauthRepo)

	router := server.InitiateRouter(userHandler, testHandler, youtubeHandler, youtubeAuthHandler, userRepository, shareHandler, facebookOAuthHandler)

	// Background share job processor (simple ticker loop)
	if shareHandler != nil {
		g.Go(func() error {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					// Process up to N pending jobs each tick
					procCtx, cancelProc := context.WithTimeout(ctx, 5*time.Second)
					_ = usecase.ProcessShareJobs(procCtx, shareRepo, oauthRepo, youtubeClient, 10)
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
		httpServer := &http.Server{
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
	var err error

	db, err := persistence.NewNativeDb()
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Cannot connect to the local database")
		return nil, nil, err
	}

	postgres, err := persistence.NewPostgreSQLDB()
	if err != nil {
		return nil, nil, err
	}

	return db, postgres, err
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
