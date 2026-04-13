package httpserver

import (
	"database/sql"
	"errors"
	"knowledge-srv/config"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/notebook"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/search"
	"knowledge-srv/pkg/maestro"
	pkgQdrant "knowledge-srv/pkg/qdrant"
	"knowledge-srv/pkg/voyage"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/encrypter"
	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/llm"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/middleware"
	"github.com/smap-hcmut/shared-libs/go/minio"
	"github.com/smap-hcmut/shared-libs/go/redis"
)

type HTTPServer struct {
	// Server Configuration
	gin         *gin.Engine
	l           log.Logger
	port        int
	mode        string
	environment string

	// Database Configuration
	qdrantClient  pkgQdrant.IQdrant
	postgresDB    *sql.DB
	voyageClient  voyage.IVoyage
	llmClient     llm.LLM
	minioClient   minio.MinIO
	kafkaProducer kafka.IProducer

	// Authentication & Security Configuration
	config       *config.Config
	jwtManager   auth.Manager
	redisClient  redis.IRedis
	cookieConfig config.CookieConfig
	encrypter    encrypter.Encrypter

	// Maestro - NotebookLM automation (optional)
	maestroClient maestro.IMaestro

	// Monitoring & Notification Configuration
	discord discord.IDiscord

	// Core Domains (Shared)
	pointUC     point.UseCase
	embeddingUC embedding.UseCase
	searchUC    search.UseCase
	notebookUC  notebook.UseCase
}

type Config struct {
	// Server Configuration
	Logger      log.Logger
	Port        int
	Mode        string
	Environment string

	// Qdrant - Vector database
	QdrantClient pkgQdrant.IQdrant

	// Voyage - Embedding
	VoyageClient voyage.IVoyage

	// LLM - Multi-provider with fallback
	LLMClient llm.LLM

	// MinIO - Storage
	MinIOClient minio.MinIO

	// Kafka - Event streaming
	KafkaProducer kafka.IProducer

	// PostgreSQL - Metadata, conversation history
	PostgresDB *sql.DB

	// Authentication & Security Configuration
	Config       *config.Config
	JWTManager   auth.Manager
	RedisClient  redis.IRedis
	CookieConfig config.CookieConfig
	Encrypter    encrypter.Encrypter

	// Maestro - NotebookLM automation (optional)
	MaestroClient maestro.IMaestro

	// Monitoring & Notification Configuration
	Discord discord.IDiscord
}

// New creates a new HTTPServer instance with the provided configuration.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		// Server Configuration
		l:           logger,
		gin:         gin.New(),
		port:        cfg.Port,
		mode:        cfg.Mode,
		environment: cfg.Environment,

		// Database Configuration
		qdrantClient:  cfg.QdrantClient,
		voyageClient:  cfg.VoyageClient,
		llmClient:     cfg.LLMClient,
		minioClient:   cfg.MinIOClient,
		kafkaProducer: cfg.KafkaProducer,
		postgresDB:    cfg.PostgresDB,

		// Authentication & Security Configuration
		config:       cfg.Config,
		jwtManager:   cfg.JWTManager,
		redisClient:  cfg.RedisClient,
		cookieConfig: cfg.CookieConfig,
		encrypter:    cfg.Encrypter,

		// Maestro - NotebookLM automation (optional)
		maestroClient: cfg.MaestroClient,

		// Monitoring & Notification Configuration
		discord: cfg.Discord,
	}

	if err := srv.validate(); err != nil {
		return nil, err
	}

	// Add middlewares
	srv.gin.Use(middleware.Logger(srv.l, srv.environment))
	srv.gin.Use(gin.Recovery())

	return srv, nil
}

// validate validates that all required dependencies are provided.
func (srv HTTPServer) validate() error {
	// Server Configuration
	if srv.l == nil {
		return errors.New("logger is required")
	}
	if srv.mode == "" {
		return errors.New("mode is required")
	}
	if srv.port == 0 {
		return errors.New("port is required")
	}

	// Core dependencies
	if srv.postgresDB == nil {
		return errors.New("postgresDB is required")
	}
	if srv.qdrantClient == nil {
		return errors.New("qdrantClient is required")
	}
	if srv.voyageClient == nil {
		return errors.New("voyageClient is required")
	}
	if srv.llmClient == nil {
		return errors.New("llmClient is required")
	}
	if srv.minioClient == nil {
		return errors.New("minioClient is required")
	}
	// Authentication & Security
	if srv.config == nil {
		return errors.New("config is required")
	}
	if srv.jwtManager == nil {
		return errors.New("jwtManager is required")
	}
	if srv.redisClient == nil {
		return errors.New("redisClient is required")
	}
	if srv.encrypter == nil {
		return errors.New("encrypter is required")
	}
	if srv.discord == nil {
		return errors.New("discord is required")
	}
	return nil
}
