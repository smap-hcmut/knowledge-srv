package main

import (
	"context"
	"fmt"
	"knowledge-srv/config"
	configPostgre "knowledge-srv/config/postgre"
	_ "knowledge-srv/docs" // Import swagger docs
	authUsecase "knowledge-srv/internal/authentication/usecase"
	"knowledge-srv/internal/httpserver"
	"knowledge-srv/pkg/discord"
	"knowledge-srv/pkg/encrypter"
	pkgJWT "knowledge-srv/pkg/jwt"
	"knowledge-srv/pkg/log"
	pkgRedis "knowledge-srv/pkg/redis"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// @title       SMAP Identity Service API
// @description SMAP Identity Service API documentation.
// @version     1
// @host        knowledge-srv.tantai.dev
// @schemes     https
// @BasePath    /knowledge
//
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name smap_auth_token
// @description Authentication token stored in HttpOnly cookie. Set automatically by /login endpoint.
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Legacy Bearer token authentication (deprecated - use cookie authentication instead). Format: "Bearer {token}"
func main() {
	// 1. Load configuration
	// Reads config from YAML file and environment variables
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		return
	}

	// 2. Initialize logger
	logger := log.Init(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	// 3. Register graceful shutdown
	registerGracefulShutdown(logger)

	// 4. Initialize encrypter
	encrypterInstance := encrypter.New(cfg.Encrypter.Key)

	// 5. Initialize PostgreSQL
	ctx := context.Background()
	postgresDB, err := configPostgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Error(ctx, "Failed to connect to PostgreSQL: ", err)
		return
	}
	defer configPostgre.Disconnect(ctx, postgresDB)
	logger.Infof(ctx, "PostgreSQL connected successfully to %s:%d/%s", cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.DBName)

	// 6. Initialize Discord (optional)
	discordClient, err := discord.New(logger, &discord.DiscordWebhook{
		ID:    cfg.Discord.WebhookID,
		Token: cfg.Discord.WebhookToken,
	})
	if err != nil {
		logger.Warnf(ctx, "Discord webhook not configured (optional): %v", err)
		discordClient = nil // Continue without Discord
	} else {
		logger.Infof(ctx, "Discord webhook initialized successfully")
	}

	// 7. Initialize Redis
	redisClient, err := pkgRedis.New(pkgRedis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		logger.Error(ctx, "Failed to connect to Redis: ", err)
		return
	}
	logger.Infof(ctx, "Redis connected successfully to %s:%d (DB %d)", cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB)

	// 9. Initialize JWT Manager
	jwtManager, err := initializeJWTManager(ctx, logger, cfg)
	if err != nil {
		logger.Error(ctx, "Failed to initialize JWT manager: ", err)
		return
	}
	logger.Infof(ctx, "JWT Manager initialized with algorithm: %s", cfg.JWT.Algorithm)

	// 10. Initialize Redirect Validator
	// Validates OAuth redirect URLs against whitelist to prevent open redirect attacks
	redirectValidator := authUsecase.NewRedirectValidator(cfg.AccessControl.AllowedRedirectURLs)
	logger.Infof(ctx, "Redirect validator initialized with %d allowed URLs", len(cfg.AccessControl.AllowedRedirectURLs))

	// 11. Initialize HTTP server
	// 11. Initialize HTTP server
	// Main application server that handles all HTTP requests and routes
	httpServer, err := httpserver.New(logger, httpserver.Config{
		// Server Configuration
		Logger:      logger,
		Host:        cfg.HTTPServer.Host,
		Port:        cfg.HTTPServer.Port,
		Mode:        cfg.HTTPServer.Mode,
		Environment: cfg.Environment.Name,

		// Database Configuration
		PostgresDB: postgresDB,

		// Authentication & Security Configuration
		Config:            cfg,
		JWTManager:        jwtManager,
		RedisClient:       redisClient,
		RedirectValidator: redirectValidator,
		CookieConfig:      cfg.Cookie,
		Encrypter:         encrypterInstance,

		// Monitoring & Notification Configuration
		Discord: discordClient,
	})
	if err != nil {
		logger.Error(ctx, "Failed to initialize HTTP server: ", err)
		return
	}

	if err := httpServer.Run(); err != nil {
		logger.Error(ctx, "Failed to run server: ", err)
		return
	}
}

// registerGracefulShutdown registers a signal handler for graceful shutdown.
func registerGracefulShutdown(logger log.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info(context.Background(), "Shutting down gracefully...")

		logger.Info(context.Background(), "Cleanup completed")
		os.Exit(0)
	}()
}

// initializeJWTManager initializes JWT manager with HS256 symmetric key
func initializeJWTManager(ctx context.Context, logger log.Logger, cfg *config.Config) (*pkgJWT.Manager, error) {
	// Create JWT manager with secret key from config
	return pkgJWT.New(pkgJWT.Config{
		SecretKey: cfg.JWT.SecretKey,
		Issuer:    cfg.JWT.Issuer,
		Audience:  cfg.JWT.Audience,
		TTL:       time.Duration(cfg.JWT.TTL) * time.Second,
	})
}
