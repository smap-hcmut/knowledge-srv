package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all service configuration.
type Config struct {
	// Environment Configuration
	Environment EnvironmentConfig

	// Server Configuration
	HTTPServer HTTPServerConfig
	Logger     LoggerConfig

	// Qdrant - Vector database
	Qdrant QdrantConfig

	// Voyage - Embedding
	Voyage VoyageConfig

	// Gemini - LLM
	Gemini GeminiConfig

	// PostgreSQL - Metadata, conversation history
	Postgres PostgresConfig

	// Redis - Caching, rate limiting
	Redis RedisConfig

	// Project Service - Campaign projects
	Project ProjectConfig

	// MinIO - Storage
	MinIO MinIOConfig

	// Kafka - Event streaming
	Kafka KafkaConfig

	// JWT - Authentication
	JWT            JWTConfig
	Cookie         CookieConfig
	Encrypter      EncrypterConfig
	InternalConfig InternalConfig

	// Maestro - NotebookLM browser automation
	Maestro MaestroConfig

	// Notebook - NotebookLM sync configuration
	Notebook NotebookConfig

	// Router - Query routing configuration
	Router RouterConfig

	// Monitoring & Notification Configuration
	Discord DiscordConfig
}

// EnvironmentConfig is the configuration for the deployment environment.
type EnvironmentConfig struct {
	Name string
}

// KafkaConfig is the configuration for Kafka
type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

// RedisConfig is the configuration for Redis
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// MinIOConfig is the configuration for MinIO
type MinIOConfig struct {
	Endpoint             string
	AccessKey            string
	SecretKey            string
	UseSSL               bool
	Region               string
	Bucket               string
	AsyncUploadWorkers   int
	AsyncUploadQueueSize int
}

// QdrantConfig is the configuration for Qdrant
type QdrantConfig struct {
	Host    string
	Port    int
	APIKey  string
	UseTLS  bool
	Timeout int // in seconds
}

// VoyageConfig is the configuration for Voyage AI (embedding). Same shape as pkg/voyage.VoyageConfig.
type VoyageConfig struct {
	APIKey string
}

// GeminiConfig is the configuration for Google Gemini (LLM). Same shape as pkg/gemini.GeminiConfig.
type GeminiConfig struct {
	APIKey string
	Model  string
}

// ProjectConfig is the configuration for Project Service
type ProjectConfig struct {
	URL     string
	Timeout int // in seconds
}

// CookieConfig is the configuration for HttpOnly cookie authentication
// Note: Secure and SameSite are now dynamically determined by auth.Middleware
// based on the request Origin header. Bearer token acceptance is controlled by ENVIRONMENT_NAME.
type CookieConfig struct {
	Name   string // Cookie name (e.g., "smap_auth_token")
	MaxAge int    // Cookie max age in seconds (e.g., 28800 for 8 hours)
	Domain string // Production domain for cookies (e.g., ".tantai.dev")
}

// JWTConfig is for verifying tokens only
type JWTConfig struct {
	SecretKey string
}

// HTTPServerConfig is the configuration for the HTTP server
type HTTPServerConfig struct {
	Port int
	Mode string
}

// LoggerConfig is the configuration for the logger
type LoggerConfig struct {
	Level        string
	Mode         string
	Encoding     string
	ColorEnabled bool
}

// PostgresConfig is the configuration for Postgres
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	Schema   string // Added Schema support
}

// DiscordConfig: webhook URL từ Discord
type DiscordConfig struct {
	WebhookURL string
}

// EncrypterConfig is the configuration for the encrypter
type EncrypterConfig struct {
	Key string
}

// InternalConfig
type InternalConfig struct {
	InternalKey string
}

// MaestroConfig is the configuration for Maestro NotebookLM automation API
type MaestroConfig struct {
	BaseURL                  string
	APIKey                   string
	SessionEnv               string // "LOCAL" | "BROWSERBASE"
	SessionHealthIntervalSec int
	JobPollIntervalMs        int
	JobPollMaxAttempts       int
	WebhookSecret            string
	WebhookCallbackURL       string
}

// NotebookConfig is the configuration for NotebookLM sync
type NotebookConfig struct {
	Enabled              bool
	MaxPostsPerPart      int
	RetentionQuarters    int
	SyncRetryIntervalMin int
	SyncMaxRetries       int
	ChatTimeoutSec       int
}

// RouterConfig is the configuration for query routing between Qdrant and NotebookLM
type RouterConfig struct {
	DefaultBackend          string // "qdrant" | "notebook" | "hybrid"
	NotebookFallbackEnabled bool
	IntentClassifier        string // "rules" | "gemini_flash"
}

// Load loads configuration using Viper
func Load() (*Config, error) {
	// Set config file name and paths
	viper.SetConfigName("knowledge-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/smap/")

	// Enable environment variable override
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicit env bindings for keys where AutomaticEnv + replacer doesn't work.
	// Viper's replacer converts "." -> "_" for env lookup, but env vars also use "_"
	// as word separator, causing ambiguity (e.g. KAFKA_BROKERS != kafka.brokers).
	// Postgres
	_ = viper.BindEnv("postgres.host", "POSTGRES_HOST")
	_ = viper.BindEnv("postgres.port", "POSTGRES_PORT")
	_ = viper.BindEnv("postgres.user", "POSTGRES_USER")
	_ = viper.BindEnv("postgres.password", "POSTGRES_PASSWORD")
	_ = viper.BindEnv("postgres.dbname", "POSTGRES_DB")
	_ = viper.BindEnv("postgres.sslmode", "POSTGRES_SSLMODE")
	_ = viper.BindEnv("postgres.schema", "POSTGRES_SCHEMA")
	// Redis
	_ = viper.BindEnv("redis.host", "REDIS_HOST")
	_ = viper.BindEnv("redis.port", "REDIS_PORT")
	_ = viper.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = viper.BindEnv("redis.db", "REDIS_DB")
	// Kafka
	_ = viper.BindEnv("kafka.brokers", "KAFKA_BROKERS")
	_ = viper.BindEnv("kafka.topic", "KAFKA_TOPIC")
	_ = viper.BindEnv("kafka.group_id", "KAFKA_GROUP_ID")
	// Qdrant
	_ = viper.BindEnv("qdrant.host", "QDRANT_HOST")
	_ = viper.BindEnv("qdrant.port", "QDRANT_PORT")
	_ = viper.BindEnv("qdrant.api_key", "QDRANT_API_KEY")
	_ = viper.BindEnv("qdrant.use_tls", "QDRANT_USE_TLS")
	_ = viper.BindEnv("qdrant.timeout", "QDRANT_TIMEOUT")
	// MinIO
	_ = viper.BindEnv("minio.endpoint", "MINIO_ENDPOINT")
	_ = viper.BindEnv("minio.access_key", "MINIO_ACCESS_KEY")
	_ = viper.BindEnv("minio.secret_key", "MINIO_SECRET_KEY")
	_ = viper.BindEnv("minio.use_ssl", "MINIO_USE_SSL")
	_ = viper.BindEnv("minio.region", "MINIO_REGION")
	_ = viper.BindEnv("minio.bucket", "MINIO_BUCKET")
	// Gemini / Voyage
	_ = viper.BindEnv("gemini.api_key", "GEMINI_API_KEY")
	_ = viper.BindEnv("gemini.model", "GEMINI_MODEL")
	_ = viper.BindEnv("voyage.api_key", "VOYAGE_API_KEY")
	// Maestro
	_ = viper.BindEnv("maestro.base_url", "MAESTRO_BASE_URL")
	_ = viper.BindEnv("maestro.api_key", "MAESTRO_API_KEY")
	_ = viper.BindEnv("maestro.webhook_secret", "MAESTRO_WEBHOOK_SECRET")
	_ = viper.BindEnv("maestro.webhook_callback_url", "MAESTRO_WEBHOOK_CALLBACK_URL")
	// Notebook
	_ = viper.BindEnv("notebook.enabled", "NOTEBOOK_ENABLED")
	// JWT / Cookie / Encrypter / Internal
	_ = viper.BindEnv("jwt.secret_key", "JWT_SECRET_KEY")
	_ = viper.BindEnv("jwt.secret_key", "JWT_SECRET") // alt name used in k8s secret
	_ = viper.BindEnv("cookie.name", "COOKIE_NAME")
	_ = viper.BindEnv("cookie.max_age", "COOKIE_MAX_AGE")
	_ = viper.BindEnv("cookie.domain", "COOKIE_DOMAIN")
	_ = viper.BindEnv("encrypter.key", "ENCRYPTER_KEY")
	_ = viper.BindEnv("internal.internal_key", "INTERNAL_KEY")
	// Project / Environment / HTTP
	_ = viper.BindEnv("project.url", "PROJECT_URL")
	_ = viper.BindEnv("project.timeout", "PROJECT_TIMEOUT")
	_ = viper.BindEnv("environment.name", "ENVIRONMENT_NAME")
	_ = viper.BindEnv("http_server.port", "HTTP_SERVER_PORT")
	_ = viper.BindEnv("http_server.mode", "HTTP_SERVER_MODE")
	// Logger
	_ = viper.BindEnv("logger.level", "LOGGER_LEVEL")
	_ = viper.BindEnv("logger.mode", "LOGGER_MODE")
	_ = viper.BindEnv("logger.encoding", "LOGGER_ENCODING")
	_ = viper.BindEnv("logger.color_enabled", "LOGGER_COLOR_ENABLED")

	// Set defaults
	setDefaults()

	// Read config file (optional - will use env vars if file not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; using environment variables
	}

	cfg := &Config{}

	// Environment & Server
	cfg.Environment.Name = viper.GetString("environment.name")
	cfg.HTTPServer.Port = viper.GetInt("http_server.port")
	cfg.HTTPServer.Mode = viper.GetString("http_server.mode")
	cfg.Logger.Level = viper.GetString("logger.level")
	cfg.Logger.Mode = viper.GetString("logger.mode")
	cfg.Logger.Encoding = viper.GetString("logger.encoding")
	cfg.Logger.ColorEnabled = viper.GetBool("logger.color_enabled")

	// Qdrant
	cfg.Qdrant.Host = viper.GetString("qdrant.host")
	cfg.Qdrant.Port = viper.GetInt("qdrant.port")
	cfg.Qdrant.APIKey = viper.GetString("qdrant.api_key")
	cfg.Qdrant.UseTLS = viper.GetBool("qdrant.use_tls")
	cfg.Qdrant.Timeout = viper.GetInt("qdrant.timeout")

	// Voyage - Embedding
	cfg.Voyage.APIKey = viper.GetString("voyage.api_key")
	if cfg.Voyage.APIKey == "" {
		cfg.Voyage.APIKey = viper.GetString("ai.voyage_api_key") // backward compat
	}

	// Gemini - LLM
	cfg.Gemini.APIKey = viper.GetString("gemini.api_key")
	if cfg.Gemini.APIKey == "" {
		cfg.Gemini.APIKey = viper.GetString("ai.gemini_api_key") // backward compat
	}
	cfg.Gemini.Model = viper.GetString("gemini.model")
	if cfg.Gemini.Model == "" {
		cfg.Gemini.Model = viper.GetString("ai.gemini_model") // backward compat
	}

	// PostgreSQL - Metadata, conversation history
	cfg.Postgres.Host = viper.GetString("postgres.host")
	cfg.Postgres.Port = viper.GetInt("postgres.port")
	cfg.Postgres.User = viper.GetString("postgres.user")
	cfg.Postgres.Password = viper.GetString("postgres.password")
	cfg.Postgres.DBName = viper.GetString("postgres.dbname")
	cfg.Postgres.SSLMode = viper.GetString("postgres.sslmode")
	cfg.Postgres.Schema = viper.GetString("postgres.schema")

	// Redis - Caching, rate limiting
	cfg.Redis.Host = viper.GetString("redis.host")
	cfg.Redis.Port = viper.GetInt("redis.port")
	cfg.Redis.Password = viper.GetString("redis.password")
	cfg.Redis.DB = viper.GetInt("redis.db")

	// Project Service - Get campaign projects
	cfg.Project.URL = viper.GetString("project.url")
	cfg.Project.Timeout = viper.GetInt("project.timeout")

	// MinIO - Report storage (PDF/DOCX)
	cfg.MinIO.Endpoint = viper.GetString("minio.endpoint")
	cfg.MinIO.AccessKey = viper.GetString("minio.access_key")
	cfg.MinIO.SecretKey = viper.GetString("minio.secret_key")
	cfg.MinIO.UseSSL = viper.GetBool("minio.use_ssl")
	cfg.MinIO.Region = viper.GetString("minio.region")
	cfg.MinIO.Bucket = viper.GetString("minio.bucket")
	cfg.MinIO.AsyncUploadWorkers = viper.GetInt("minio.async_upload_workers")
	cfg.MinIO.AsyncUploadQueueSize = viper.GetInt("minio.async_upload_queue_size")

	// Kafka - Event publishing (optional)
	cfg.Kafka.Brokers = viper.GetStringSlice("kafka.brokers")
	cfg.Kafka.Topic = viper.GetString("kafka.topic")
	cfg.Kafka.GroupID = viper.GetString("kafka.group_id")

	// JWT
	cfg.JWT.SecretKey = viper.GetString("jwt.secret_key")

	// Cookie
	cfg.Cookie.Name = viper.GetString("cookie.name")
	cfg.Cookie.MaxAge = viper.GetInt("cookie.max_age")
	cfg.Cookie.Domain = viper.GetString("cookie.domain")

	// Encrypter
	cfg.Encrypter.Key = viper.GetString("encrypter.key")

	// Internal auth
	cfg.InternalConfig.InternalKey = viper.GetString("internal.internal_key")

	// Discord
	cfg.Discord.WebhookURL = viper.GetString("discord.webhook_url")

	// Maestro - NotebookLM automation
	cfg.Maestro.BaseURL = viper.GetString("maestro.base_url")
	cfg.Maestro.APIKey = viper.GetString("maestro.api_key")
	cfg.Maestro.SessionEnv = viper.GetString("maestro.session_env")
	cfg.Maestro.SessionHealthIntervalSec = viper.GetInt("maestro.session_health_interval_sec")
	cfg.Maestro.JobPollIntervalMs = viper.GetInt("maestro.job_poll_interval_ms")
	cfg.Maestro.JobPollMaxAttempts = viper.GetInt("maestro.job_poll_max_attempts")
	cfg.Maestro.WebhookSecret = viper.GetString("maestro.webhook_secret")
	cfg.Maestro.WebhookCallbackURL = viper.GetString("maestro.webhook_callback_url")

	// Notebook - NotebookLM sync
	cfg.Notebook.Enabled = viper.GetBool("notebook.enabled")
	cfg.Notebook.MaxPostsPerPart = viper.GetInt("notebook.max_posts_per_part")
	cfg.Notebook.RetentionQuarters = viper.GetInt("notebook.retention_quarters")
	cfg.Notebook.SyncRetryIntervalMin = viper.GetInt("notebook.sync_retry_interval_min")
	cfg.Notebook.SyncMaxRetries = viper.GetInt("notebook.sync_max_retries")
	cfg.Notebook.ChatTimeoutSec = viper.GetInt("notebook.chat_timeout_sec")

	// Router - Query routing
	cfg.Router.DefaultBackend = viper.GetString("router.default_backend")
	cfg.Router.NotebookFallbackEnabled = viper.GetBool("router.notebook_fallback_enabled")
	cfg.Router.IntentClassifier = viper.GetString("router.intent_classifier")

	// Validate required fields
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func setDefaults() {
	// Environment
	viper.SetDefault("environment.name", "production")

	// HTTP Server
	viper.SetDefault("http_server.port", 8080)
	viper.SetDefault("http_server.mode", "debug")

	// Logger
	viper.SetDefault("logger.level", "debug")
	viper.SetDefault("logger.mode", "debug")
	viper.SetDefault("logger.encoding", "console")
	viper.SetDefault("logger.color_enabled", true)

	// 1. Qdrant
	viper.SetDefault("qdrant.host", "localhost")
	viper.SetDefault("qdrant.port", 6334)
	viper.SetDefault("qdrant.use_tls", false)
	viper.SetDefault("qdrant.timeout", 30)

	// 2. AI (Voyage + Gemini)
	viper.SetDefault("gemini.model", "gemini-1.5-pro")

	// 3. PostgreSQL (schema per specs: knowledge)
	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", 5432)
	viper.SetDefault("postgres.user", "postgres")
	viper.SetDefault("postgres.password", "postgres")
	viper.SetDefault("postgres.dbname", "postgres")
	viper.SetDefault("postgres.sslmode", "prefer")
	viper.SetDefault("postgres.schema", "knowledge")

	// 4. Redis
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// 5. Project Service
	viper.SetDefault("project.url", "http://project-service:8080")
	viper.SetDefault("project.timeout", 10)

	// 6. MinIO (bucket per specs: smap-reports)
	viper.SetDefault("minio.endpoint", "localhost:9000")
	viper.SetDefault("minio.access_key", "minioadmin")
	viper.SetDefault("minio.secret_key", "minioadmin")
	viper.SetDefault("minio.use_ssl", false)
	viper.SetDefault("minio.region", "us-east-1")
	viper.SetDefault("minio.bucket", "smap-reports")
	viper.SetDefault("minio.async_upload_workers", 4)
	viper.SetDefault("minio.async_upload_queue_size", 100)

	// 7. Kafka (topic per specs: knowledge.events)
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topic", "knowledge.events")

	// OAuth2
	viper.SetDefault("oauth2.provider", "google")
	viper.SetDefault("oauth2.scopes", []string{"openid", "email", "profile"})

	// Cookie
	viper.SetDefault("cookie.name", "smap_auth_token")
	viper.SetDefault("cookie.max_age", 28800) // 8 hours
	viper.SetDefault("cookie.domain", ".tantai.dev")
	viper.SetDefault("access_control.allowed_redirect_urls", []string{"/dashboard", "/", "http://localhost:3000", "http://localhost:5173"})

	// Session
	viper.SetDefault("session.ttl", 28800)              // 8 hours
	viper.SetDefault("session.remember_me_ttl", 604800) // 7 days
	viper.SetDefault("session.backend", "redis")

	// Blacklist
	viper.SetDefault("blacklist.enabled", true)
	viper.SetDefault("blacklist.backend", "redis")
	viper.SetDefault("blacklist.key_prefix", "blacklist:")

	// Maestro
	viper.SetDefault("maestro.base_url", "https://maestro.tantai.dev/maestro")
	viper.SetDefault("maestro.session_env", "LOCAL")
	viper.SetDefault("maestro.session_health_interval_sec", 60)
	viper.SetDefault("maestro.job_poll_interval_ms", 2000)
	viper.SetDefault("maestro.job_poll_max_attempts", 30)
	viper.SetDefault("maestro.webhook_callback_url", "http://knowledge-api/internal/notebook/callback")

	// Notebook
	viper.SetDefault("notebook.enabled", false)
	viper.SetDefault("notebook.max_posts_per_part", 50)
	viper.SetDefault("notebook.retention_quarters", 2)
	viper.SetDefault("notebook.sync_retry_interval_min", 30)
	viper.SetDefault("notebook.sync_max_retries", 3)
	viper.SetDefault("notebook.chat_timeout_sec", 45)

	// Router
	viper.SetDefault("router.default_backend", "qdrant")
	viper.SetDefault("router.notebook_fallback_enabled", true)
	viper.SetDefault("router.intent_classifier", "rules")
}

func validate(cfg *Config) error {
	// Validate JWT fields
	if cfg.JWT.SecretKey == "" {
		return fmt.Errorf("jwt.secret_key is required")
	}
	if len(cfg.JWT.SecretKey) < 32 {
		return fmt.Errorf("jwt.secret_key must be at least 32 characters for security")
	}

	// Validate Encrypter
	if cfg.Encrypter.Key == "" {
		return fmt.Errorf("encrypter.key is required")
	}
	if len(cfg.Encrypter.Key) < 32 {
		return fmt.Errorf("encrypter.key must be at least 32 characters for security")
	}

	if cfg.Postgres.Host == "" {
		return fmt.Errorf("postgres.host is required")
	}
	if cfg.Postgres.Port == 0 {
		return fmt.Errorf("postgres.port is required")
	}
	if cfg.Postgres.DBName == "" {
		return fmt.Errorf("postgres.db_name is required")
	}
	if cfg.Postgres.User == "" {
		return fmt.Errorf("postgres.user is required")
	}

	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
	}
	if cfg.Redis.Port == 0 {
		return fmt.Errorf("redis.port is required")
	}

	// Validate Qdrant Configuration
	if cfg.Qdrant.Host == "" {
		return fmt.Errorf("qdrant.host is required")
	}
	if cfg.Qdrant.Port == 0 {
		return fmt.Errorf("qdrant.port is required")
	}

	// Validate Project Service Configuration
	if cfg.Project.URL == "" {
		return fmt.Errorf("project.url is required")
	}

	// Validate MinIO Configuration
	if cfg.MinIO.Endpoint == "" {
		return fmt.Errorf("minio.endpoint is required")
	}
	if cfg.MinIO.AccessKey == "" {
		return fmt.Errorf("minio.access_key is required")
	}
	if cfg.MinIO.SecretKey == "" {
		return fmt.Errorf("minio.secret_key is required")
	}
	if cfg.MinIO.Bucket == "" {
		return fmt.Errorf("minio.bucket is required")
	}

	// Validate Cookie Configuration
	if cfg.Cookie.Name == "" {
		return fmt.Errorf("cookie.name is required")
	}

	return nil
}
