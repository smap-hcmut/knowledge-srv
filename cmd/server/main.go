package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"knowledge-srv/config"
	"knowledge-srv/config/kafka"
	"knowledge-srv/config/minio"
	"knowledge-srv/config/postgre"
	"knowledge-srv/config/qdrant"
	"knowledge-srv/config/redis"

	_ "knowledge-srv/docs"
	"knowledge-srv/internal/consumer"
	"knowledge-srv/internal/httpserver"
	"knowledge-srv/pkg/voyage"

	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/encrypter"
	"github.com/smap-hcmut/shared-libs/go/llm"
	"github.com/smap-hcmut/shared-libs/go/log"
	_ "github.com/smap-hcmut/shared-libs/go/response" // For swagger type definitions
)

// rateLimitedLLM wraps llm.LLM with a concurrency semaphore to protect API quotas.
type rateLimitedLLM struct {
	inner llm.LLM
	sem   chan struct{}
}

func (r *rateLimitedLLM) Generate(ctx context.Context, prompt string) (string, error) {
	select {
	case r.sem <- struct{}{}:
		defer func() { <-r.sem }()
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return r.inner.Generate(ctx, prompt)
}

func (r *rateLimitedLLM) Name() string {
	return r.inner.Name()
}

// @title       SMAP Knowledge Service API
// @description SMAP Knowledge Service API documentation.
// @version     1
// @schemes     https http
// @BasePath    /knowledge/api/v1/knowledge
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
	logger := log.NewZapLogger(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	// Create context with signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info(ctx, "Starting Knowledge Service...")

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

	// LLM - Multi-provider with fallback (Gemini, OpenAI, DeepSeek, Qwen)
	var llmProviderConfigs []llm.ProviderConfig
	for _, pc := range cfg.LLM.Providers {
		llmProviderConfigs = append(llmProviderConfigs, llm.ProviderConfig{
			Name:   pc.Name,
			APIKey: pc.APIKey,
			Model:  pc.Model,
		})
	}
	llmBase, err := llm.NewFromConfig(llm.MultiConfig{Providers: llmProviderConfigs})
	if err != nil {
		logger.Error(ctx, "Failed to initialize LLM client: ", err)
		return
	}
	logger.Infof(ctx, "LLM client initialized: %s", llmBase.Name())

	// Wrap LLM with concurrency limiter (max 5 concurrent calls across chat + report)
	var llmClient llm.LLM = &rateLimitedLLM{inner: llmBase, sem: make(chan struct{}, 5)}

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
	jwtManager := auth.NewManager(cfg.JWT.SecretKey)
	logger.Infof(ctx, "JWT Manager initialized")

	// ── Consumer (Kafka) ────────────────────────────────────────────────────
	consumerSrv, err := consumer.New(consumer.Config{
		Logger:        logger,
		KafkaConfig:   cfg.Kafka,
		RedisClient:   redisClient,
		QdrantClient:  qdrantClient,
		PostgresDB:    postgresDB,
		MinIOClient:   minioClient,
		VoyageClient:  voyageClient,
		LLMClient:     llmClient,
		Discord:       discordClient,
		KafkaProducer: kafkaProducer,
	})
	if err != nil {
		logger.Errorf(ctx, "Failed to create consumer server: %v", err)
		return
	}

	consumerErr := make(chan error, 1)
	go func() {
		logger.Info(ctx, "Consumer server starting...")
		if err := consumerSrv.Run(ctx); err != nil {
			consumerErr <- err
		}
	}()

	// ── HTTP Server ─────────────────────────────────────────────────────────
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
		LLMClient:     llmClient,
		MinIOClient:   minioClient,
		KafkaProducer: kafkaProducer,
	})
	if err != nil {
		logger.Error(ctx, "Failed to initialize HTTP server: ", err)
		return
	}

	httpErr := make(chan error, 1)
	go func() {
		if err := httpServer.Run(ctx); err != nil {
			httpErr <- err
		}
	}()

	logger.Info(ctx, "Knowledge Service started")

	// ── Wait for shutdown signal or fatal error ─────────────────────────────
	select {
	case <-ctx.Done():
		logger.Info(ctx, "Shutdown signal received")
	case err := <-consumerErr:
		logger.Errorf(ctx, "Consumer fatal error, shutting down: %v", err)
		stop()
	case err := <-httpErr:
		logger.Errorf(ctx, "HTTP server fatal error, shutting down: %v", err)
		stop()
	}

	logger.Info(ctx, "Knowledge Service stopped gracefully")
}
