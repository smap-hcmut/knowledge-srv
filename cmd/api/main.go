package main

import (
	"context"
	"fmt"
	"knowledge-srv/config"
	"knowledge-srv/config/kafka"
	"knowledge-srv/config/minio"
	"knowledge-srv/config/postgre"
	"knowledge-srv/config/qdrant"
	"knowledge-srv/config/redis"
	_ "knowledge-srv/docs" // Swagger docs - blank import to trigger init()
	"knowledge-srv/internal/httpserver"
	"knowledge-srv/pkg/discord"
	"knowledge-srv/pkg/encrypter"
	"knowledge-srv/pkg/gemini"
	"knowledge-srv/pkg/jwt"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/voyage"
	"os"
	"os/signal"
	"syscall"
)

// @title       SMAP Knowledge Service API
// @description SMAP Knowledge Service API documentation.
// @version     1
// @host        localhost:8080
// @schemes     http
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
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		return
	}

	// Initialize logger
	logger := log.Init(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	// Create context with signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info(ctx, "Starting Knowledge API Service...")

	// Encrypter
	encrypterInstance := encrypter.New(cfg.Encrypter.Key)

	// Qdrant
	qdrantClient, err := qdrant.Connect(ctx, cfg.Qdrant)
	if err != nil {
		logger.Error(ctx, "Failed to connect to Qdrant: ", err)
		return
	}
	defer qdrant.Disconnect()
	logger.Infof(ctx, "Qdrant client initialized")

	// Voyage - Embedding
	voyageClient, err := voyage.NewVoyage(voyage.VoyageConfig{APIKey: cfg.Voyage.APIKey})
	if err != nil {
		logger.Error(ctx, "Failed to initialize Voyage client: ", err)
		return
	}
	logger.Info(ctx, "Voyage client initialized")

	geminiClient, err := gemini.NewGemini(gemini.GeminiConfig{APIKey: cfg.Gemini.APIKey, Model: cfg.Gemini.Model})
	if err != nil {
		logger.Error(ctx, "Failed to initialize Gemini client: ", err)
		return
	}
	logger.Info(ctx, "Gemini client initialized")

	// PostgreSQL - Metadata, conversation history
	postgresDB, err := postgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Error(ctx, "Failed to connect to PostgreSQL: ", err)
		return
	}
	defer postgre.Disconnect(ctx, postgresDB)
	logger.Infof(ctx, "PostgreSQL client initialized")

	// Redis - Caching, rate limiting
	redisClient, err := redis.Connect(ctx, cfg.Redis)
	if err != nil {
		logger.Error(ctx, "Failed to connect to Redis: ", err)
		return
	}
	defer redis.Disconnect()
	logger.Infof(ctx, "Redis client initialized")

	// MinIO - Report storage (PDF/DOCX)
	minioClient, err := minio.Connect(ctx, &cfg.MinIO)
	if err != nil {
		logger.Error(ctx, "Failed to connect to MinIO: ", err)
		return
	}
	defer minio.Disconnect()
	logger.Infof(ctx, "MinIO client initialized")

	// Kafka - Event publishing (optional)
	kafkaProducer, err := kafka.ConnectProducer(cfg.Kafka)
	if err != nil {
		logger.Warnf(ctx, "Kafka not configured or unavailable (optional): %v", err)
		kafkaProducer = nil
	} else {
		defer kafka.DisconnectProducer()
		logger.Infof(ctx, "Kafka producer client initialized")
	}

	// Discord - Monitoring & Notification
	discordClient, err := discord.New(logger, cfg.Discord.WebhookURL)
	if err != nil {
		logger.Warnf(ctx, "Discord webhook not configured (optional): %v", err)
		discordClient = nil
	} else {
		logger.Info(ctx, "Discord client initialized")
	}

	// JWT Manager (verify tokens from cookie/header)
	jwtManager, err := jwt.New(jwt.Config{SecretKey: cfg.JWT.SecretKey})
	if err != nil {
		logger.Error(ctx, "Failed to initialize JWT manager: ", err)
		return
	}
	logger.Infof(ctx, "JWT Manager initialized")

	// HTTP server
	httpServer, err := httpserver.New(logger, httpserver.Config{
		Logger:      logger,
		Port:        cfg.HTTPServer.Port,
		Mode:        cfg.HTTPServer.Mode,
		Environment: cfg.Environment.Name,

		PostgresDB: postgresDB,

		Config:       cfg,
		JWTManager:   jwtManager,
		RedisClient:  redisClient,
		CookieConfig: cfg.Cookie,
		Encrypter:    encrypterInstance,

		Discord: discordClient,

		QdrantClient:  qdrantClient,
		VoyageClient:  voyageClient,
		GeminiClient:  geminiClient,
		MinIOClient:   minioClient,
		KafkaProducer: kafkaProducer,
	})
	if err != nil {
		logger.Error(ctx, "Failed to initialize HTTP server: ", err)
		return
	}

	if err := httpServer.Run(); err != nil {
		logger.Error(ctx, "Failed to run server: ", err)
		return
	}

	logger.Info(ctx, "API server stopped gracefully")
}
