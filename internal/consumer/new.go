package consumer

import (
	"fmt"
)

// New creates a new consumer server with dependency validation
func New(cfg Config) (*ConsumerServer, error) {
	srv := &ConsumerServer{
		l:             cfg.Logger,
		kafkaConfig:   cfg.KafkaConfig,
		redisClient:   cfg.RedisClient,
		qdrantClient:  cfg.QdrantClient,
		postgresDB:    cfg.PostgresDB,
		minioClient:   cfg.MinIOClient,
		voyageClient:  cfg.VoyageClient,
		geminiClient:  cfg.GeminiClient,
		discord:       cfg.Discord,
		kafkaProducer: cfg.KafkaProducer,
	}

	if err := srv.validate(); err != nil {
		return nil, err
	}

	return srv, nil
}

// validate validates that all required dependencies are provided
func (srv *ConsumerServer) validate() error {
	// Core Configuration
	if srv.l == nil {
		return fmt.Errorf("logger is required")
	}
	if len(srv.kafkaConfig.Brokers) == 0 {
		return fmt.Errorf("kafka brokers are required")
	}

	// Infrastructure clients
	if srv.redisClient == nil {
		return fmt.Errorf("redis client is required")
	}
	if srv.qdrantClient == nil {
		return fmt.Errorf("qdrant client is required")
	}
	if srv.postgresDB == nil {
		return fmt.Errorf("postgres db is required")
	}
	if srv.minioClient == nil {
		return fmt.Errorf("minio client is required")
	}
	if srv.kafkaProducer == nil {
		return fmt.Errorf("kafka producer is required")
	}

	// AI/ML clients
	if srv.voyageClient == nil {
		return fmt.Errorf("voyage client is required")
	}
	if srv.geminiClient == nil {
		return fmt.Errorf("gemini client is required")
	}

	return nil
}
