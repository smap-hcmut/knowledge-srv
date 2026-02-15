package httpserver

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"

	"knowledge-srv/config"
	"knowledge-srv/pkg/discord"
	"knowledge-srv/pkg/encrypter"
	"knowledge-srv/pkg/gemini"
	pkgJWT "knowledge-srv/pkg/jwt"
	"knowledge-srv/pkg/kafka"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/minio"
	pkgQdrant "knowledge-srv/pkg/qdrant"
	pkgRedis "knowledge-srv/pkg/redis"
	"knowledge-srv/pkg/voyage"
)

type HTTPServer struct {
	// Server Configuration
	gin         *gin.Engine
	l           log.Logger
	host        string
	port        int
	mode        string
	environment string

	// Database Configuration
	qdrantClient  pkgQdrant.IQdrant
	postgresDB    *sql.DB
	voyageClient  voyage.IVoyage
	geminiClient  gemini.IGemini
	minioClient   minio.MinIO
	kafkaProducer kafka.IProducer

	// Authentication & Security Configuration
	config       *config.Config
	jwtManager   pkgJWT.IManager
	redisClient  pkgRedis.IRedis
	cookieConfig config.CookieConfig
	encrypter    encrypter.Encrypter

	// Monitoring & Notification Configuration
	discord discord.IDiscord
}

type Config struct {
	// Server Configuration
	Logger      log.Logger
	Host        string
	Port        int
	Mode        string
	Environment string

	// Qdrant - Vector database
	QdrantClient pkgQdrant.IQdrant

	// Voyage - Embedding
	VoyageClient voyage.IVoyage

	// Gemini - LLM
	GeminiClient gemini.IGemini

	// MinIO - Storage
	MinIOClient minio.MinIO

	// Kafka - Event streaming
	KafkaProducer kafka.IProducer

	// PostgreSQL - Metadata, conversation history
	PostgresDB *sql.DB

	// Authentication & Security Configuration
	Config       *config.Config
	JWTManager   pkgJWT.IManager
	RedisClient  pkgRedis.IRedis
	CookieConfig config.CookieConfig
	Encrypter    encrypter.Encrypter

	// Monitoring & Notification Configuration
	Discord discord.IDiscord
}

// New creates a new HTTPServer instance with the provided configuration.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		// Server Configuration
		l:           logger,
		gin:         gin.Default(),
		host:        cfg.Host,
		port:        cfg.Port,
		mode:        cfg.Mode,
		environment: cfg.Environment,

		// Database Configuration
		qdrantClient:  cfg.QdrantClient,
		voyageClient:  cfg.VoyageClient,
		geminiClient:  cfg.GeminiClient,
		minioClient:   cfg.MinIOClient,
		kafkaProducer: cfg.KafkaProducer,
		postgresDB:    cfg.PostgresDB,

		// Authentication & Security Configuration
		config:       cfg.Config,
		jwtManager:   cfg.JWTManager,
		redisClient:  cfg.RedisClient,
		cookieConfig: cfg.CookieConfig,
		encrypter:    cfg.Encrypter,

		// Monitoring & Notification Configuration
		discord: cfg.Discord,
	}

	if err := srv.validate(); err != nil {
		return nil, err
	}

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
	if srv.geminiClient == nil {
		return errors.New("geminiClient is required")
	}
	if srv.minioClient == nil {
		return errors.New("minioClient is required")
	}
	if srv.kafkaProducer == nil {
		return errors.New("kafkaProducer is required")
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
