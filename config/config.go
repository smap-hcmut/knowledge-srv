package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	// Environment Configuration
	Environment EnvironmentConfig

	// Server Configuration
	HTTPServer HTTPServerConfig
	Logger     LoggerConfig

	// Database Configuration
	Postgres PostgresConfig

	// Message Queue Configuration (Kafka replaces RabbitMQ)
	Kafka KafkaConfig

	// Cache Configuration
	Redis RedisConfig

	// MinIO Configuration
	MinIO MinIOConfig

	// Vector Database Configuration
	Qdrant QdrantConfig

	// AI Configuration
	AI AIConfig

	// Project Service Configuration
	Project ProjectConfig

	// Authentication & Security Configuration
	OAuth2         OAuth2Config
	JWT            JWTConfig
	Cookie         CookieConfig
	Encrypter      EncrypterConfig
	InternalConfig InternalConfig

	// Google Workspace Integration
	GoogleWorkspace GoogleWorkspaceConfig

	// Access Control
	AccessControl AccessControlConfig

	// Session Configuration
	Session SessionConfig

	// Token Blacklist
	Blacklist BlacklistConfig

	// Monitoring & Notification Configuration
	Discord DiscordConfig
}

// EnvironmentConfig is the configuration for the deployment environment.
type EnvironmentConfig struct {
	Name string
}

// OAuth2Config is the configuration for OAuth2 authentication
type OAuth2Config struct {
	Provider     string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	OktaDomain   string // Only for Okta provider
}

// GoogleWorkspaceConfig is the configuration for Google Workspace integration
type GoogleWorkspaceConfig struct {
	ServiceAccountKey string
	AdminEmail        string
	Domain            string
}

// AccessControlConfig is the configuration for access control
type AccessControlConfig struct {
	AllowedDomains      []string
	BlockedEmails       []string
	AllowedRedirectURLs []string
	UserRoles           map[string]string
	DefaultRole         string
}

// SessionConfig is the configuration for session management
type SessionConfig struct {
	TTL           int // in seconds
	RememberMeTTL int // in seconds
	Backend       string
}

// BlacklistConfig is the configuration for token blacklist
type BlacklistConfig struct {
	Enabled   bool
	Backend   string
	KeyPrefix string
}

// KafkaConfig is the configuration for Kafka
type KafkaConfig struct {
	Brokers []string
	Topic   string
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

// AIConfig is the configuration for AI services
type AIConfig struct {
	VoyageAPIKey string
	GeminiAPIKey string
	GeminiModel  string
}

// ProjectConfig is the configuration for Project Service
type ProjectConfig struct {
	URL     string
	Timeout int // in seconds
}

// CookieConfig is the configuration for the cookie
type CookieConfig struct {
	Domain         string
	Secure         bool
	SameSite       string
	MaxAge         int
	MaxAgeRemember int
	Name           string
}

// JWTConfig is the configuration for JWT
type JWTConfig struct {
	Algorithm string
	Issuer    string
	Audience  []string
	SecretKey string
	TTL       int // in seconds
}

// HTTPServerConfig is the configuration for the HTTP server
type HTTPServerConfig struct {
	Host string
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

type DiscordConfig struct {
	WebhookID    string
	WebhookToken string
}

// EncrypterConfig is the configuration for the encrypter
type EncrypterConfig struct {
	Key string
}

// InternalConfig is the configuration for internal service authentication
type InternalConfig struct {
	ServiceKeys map[string]string
}

// Load loads configuration using Viper
func Load() (*Config, error) {
	// Set config file name and paths
	viper.SetConfigName("auth-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/smap/")

	// Enable environment variable override
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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

	// Environment
	cfg.Environment.Name = viper.GetString("environment.name")

	// HTTP Server
	cfg.HTTPServer.Host = viper.GetString("http_server.host")
	cfg.HTTPServer.Port = viper.GetInt("http_server.port")
	cfg.HTTPServer.Mode = viper.GetString("http_server.mode")

	// Logger
	cfg.Logger.Level = viper.GetString("logger.level")
	cfg.Logger.Mode = viper.GetString("logger.mode")
	cfg.Logger.Encoding = viper.GetString("logger.encoding")
	cfg.Logger.ColorEnabled = viper.GetBool("logger.color_enabled")

	// Postgres
	cfg.Postgres.Host = viper.GetString("postgres.host")
	cfg.Postgres.Port = viper.GetInt("postgres.port")
	cfg.Postgres.User = viper.GetString("postgres.user")
	cfg.Postgres.Password = viper.GetString("postgres.password")
	cfg.Postgres.DBName = viper.GetString("postgres.dbname")
	cfg.Postgres.SSLMode = viper.GetString("postgres.sslmode")
	cfg.Postgres.Schema = viper.GetString("postgres.schema")

	// Redis
	cfg.Redis.Host = viper.GetString("redis.host")
	cfg.Redis.Port = viper.GetInt("redis.port")
	cfg.Redis.Password = viper.GetString("redis.password")
	cfg.Redis.DB = viper.GetInt("redis.db")

	// MinIO
	cfg.MinIO.Endpoint = viper.GetString("minio.endpoint")
	cfg.MinIO.AccessKey = viper.GetString("minio.access_key")
	cfg.MinIO.SecretKey = viper.GetString("minio.secret_key")
	cfg.MinIO.UseSSL = viper.GetBool("minio.use_ssl")
	cfg.MinIO.Region = viper.GetString("minio.region")
	cfg.MinIO.Bucket = viper.GetString("minio.bucket")
	cfg.MinIO.AsyncUploadWorkers = viper.GetInt("minio.async_upload_workers")
	cfg.MinIO.AsyncUploadQueueSize = viper.GetInt("minio.async_upload_queue_size")

	// Qdrant
	cfg.Qdrant.Host = viper.GetString("qdrant.host")
	cfg.Qdrant.Port = viper.GetInt("qdrant.port")
	cfg.Qdrant.APIKey = viper.GetString("qdrant.api_key")
	cfg.Qdrant.UseTLS = viper.GetBool("qdrant.use_tls")
	cfg.Qdrant.Timeout = viper.GetInt("qdrant.timeout")

	// AI
	cfg.AI.VoyageAPIKey = viper.GetString("ai.voyage_api_key")
	cfg.AI.GeminiAPIKey = viper.GetString("ai.gemini_api_key")
	cfg.AI.GeminiModel = viper.GetString("ai.gemini_model")

	// Project Service
	cfg.Project.URL = viper.GetString("project.url")
	cfg.Project.Timeout = viper.GetInt("project.timeout")

	// Kafka
	cfg.Kafka.Brokers = viper.GetStringSlice("kafka.brokers")
	cfg.Kafka.Topic = viper.GetString("kafka.topic")

	// OAuth2
	cfg.OAuth2.Provider = viper.GetString("oauth2.provider")
	cfg.OAuth2.ClientID = viper.GetString("oauth2.client_id")
	cfg.OAuth2.ClientSecret = viper.GetString("oauth2.client_secret")
	cfg.OAuth2.RedirectURI = viper.GetString("oauth2.redirect_uri")
	cfg.OAuth2.Scopes = viper.GetStringSlice("oauth2.scopes")
	cfg.OAuth2.OktaDomain = viper.GetString("oauth2.okta_domain")

	// Google Workspace
	cfg.GoogleWorkspace.ServiceAccountKey = viper.GetString("google_workspace.service_account_key")
	cfg.GoogleWorkspace.AdminEmail = viper.GetString("google_workspace.admin_email")
	cfg.GoogleWorkspace.Domain = viper.GetString("google_workspace.domain")

	// JWT
	cfg.JWT.Algorithm = viper.GetString("jwt.algorithm")
	cfg.JWT.Issuer = viper.GetString("jwt.issuer")
	cfg.JWT.Audience = viper.GetStringSlice("jwt.audience")
	cfg.JWT.SecretKey = viper.GetString("jwt.secret_key")
	cfg.JWT.TTL = viper.GetInt("jwt.ttl")

	// Cookie
	cfg.Cookie.Domain = viper.GetString("cookie.domain")
	cfg.Cookie.Secure = viper.GetBool("cookie.secure")
	cfg.Cookie.SameSite = viper.GetString("cookie.samesite")
	cfg.Cookie.MaxAge = viper.GetInt("cookie.max_age")
	cfg.Cookie.MaxAgeRemember = viper.GetInt("cookie.max_age_remember")
	cfg.Cookie.Name = viper.GetString("cookie.name")

	// Access Control
	cfg.AccessControl.AllowedDomains = viper.GetStringSlice("access_control.allowed_domains")
	cfg.AccessControl.BlockedEmails = viper.GetStringSlice("access_control.blocked_emails")
	cfg.AccessControl.AllowedRedirectURLs = viper.GetStringSlice("access_control.allowed_redirect_urls")
	cfg.AccessControl.DefaultRole = viper.GetString("access_control.default_role")

	// User roles mapping (email -> role)
	cfg.AccessControl.UserRoles = viper.GetStringMapString("access_control.user_roles")

	// Session
	cfg.Session.TTL = viper.GetInt("session.ttl")
	cfg.Session.RememberMeTTL = viper.GetInt("session.remember_me_ttl")
	cfg.Session.Backend = viper.GetString("session.backend")

	// Blacklist
	cfg.Blacklist.Enabled = viper.GetBool("blacklist.enabled")
	cfg.Blacklist.Backend = viper.GetString("blacklist.backend")
	cfg.Blacklist.KeyPrefix = viper.GetString("blacklist.key_prefix")

	// Encrypter
	cfg.Encrypter.Key = viper.GetString("encrypter.key")

	// Internal Service Keys
	serviceKeys := make(map[string]string)
	if viper.IsSet("internal.service_keys") {
		serviceKeysRaw := viper.GetStringMapString("internal.service_keys")
		for service, key := range serviceKeysRaw {
			serviceKeys[service] = key
		}
	}
	cfg.InternalConfig.ServiceKeys = serviceKeys

	// Discord
	cfg.Discord.WebhookID = viper.GetString("discord.webhook_id")
	cfg.Discord.WebhookToken = viper.GetString("discord.webhook_token")

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
	viper.SetDefault("http_server.host", "")
	viper.SetDefault("http_server.port", 8080)
	viper.SetDefault("http_server.mode", "debug")

	// Logger
	viper.SetDefault("logger.level", "debug")
	viper.SetDefault("logger.mode", "debug")
	viper.SetDefault("logger.encoding", "console")
	viper.SetDefault("logger.color_enabled", true)

	// Postgres
	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", 5432)
	viper.SetDefault("postgres.user", "postgres")
	viper.SetDefault("postgres.password", "postgres")
	viper.SetDefault("postgres.dbname", "postgres")
	viper.SetDefault("postgres.sslmode", "prefer")

	// Redis
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// MinIO
	viper.SetDefault("minio.endpoint", "localhost:9000")
	viper.SetDefault("minio.access_key", "minioadmin")
	viper.SetDefault("minio.secret_key", "minioadmin")
	viper.SetDefault("minio.use_ssl", false)
	viper.SetDefault("minio.region", "us-east-1")
	viper.SetDefault("minio.bucket", "smap-bucket")
	viper.SetDefault("minio.async_upload_workers", 4)
	viper.SetDefault("minio.async_upload_queue_size", 100)

	// Qdrant
	viper.SetDefault("qdrant.host", "localhost")
	viper.SetDefault("qdrant.port", 6333)
	viper.SetDefault("qdrant.use_tls", false)
	viper.SetDefault("qdrant.timeout", 30)

	// AI
	viper.SetDefault("ai.gemini_model", "gemini-1.5-pro")

	// Project Service
	viper.SetDefault("project.url", "http://project-service:8080")
	viper.SetDefault("project.timeout", 10)

	// Kafka
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topic", "audit.events")

	// OAuth2
	viper.SetDefault("oauth2.provider", "google")
	viper.SetDefault("oauth2.scopes", []string{"openid", "email", "profile"})

	// JWT
	viper.SetDefault("jwt.algorithm", "HS256")
	viper.SetDefault("jwt.issuer", "smap-auth-service")
	viper.SetDefault("jwt.audience", []string{"identity-srv"})
	viper.SetDefault("jwt.ttl", 28800) // 8 hours

	// Cookie
	viper.SetDefault("cookie.domain", ".smap.com")
	viper.SetDefault("cookie.secure", true)
	viper.SetDefault("cookie.samesite", "Lax")
	viper.SetDefault("cookie.max_age", 28800)           // 8 hours
	viper.SetDefault("cookie.max_age_remember", 604800) // 7 days
	viper.SetDefault("cookie.name", "smap_auth_token")

	// Access Control
	viper.SetDefault("access_control.default_role", "VIEWER")
	viper.SetDefault("access_control.allowed_redirect_urls", []string{"/dashboard", "/", "http://localhost:3000", "http://localhost:5173"})

	// Session
	viper.SetDefault("session.ttl", 28800)              // 8 hours
	viper.SetDefault("session.remember_me_ttl", 604800) // 7 days
	viper.SetDefault("session.backend", "redis")

	// Blacklist
	viper.SetDefault("blacklist.enabled", true)
	viper.SetDefault("blacklist.backend", "redis")
	viper.SetDefault("blacklist.key_prefix", "blacklist:")
}

func validate(cfg *Config) error {
	// Validate required OAuth2 fields
	if cfg.OAuth2.ClientID == "" {
		return fmt.Errorf("oauth2.client_id is required")
	}
	if cfg.OAuth2.ClientSecret == "" {
		return fmt.Errorf("oauth2.client_secret is required")
	}
	if cfg.OAuth2.RedirectURI == "" {
		return fmt.Errorf("oauth2.redirect_uri is required")
	}
	// Validate redirect URI format (Task 4.4)
	if !strings.HasPrefix(cfg.OAuth2.RedirectURI, "http://") && !strings.HasPrefix(cfg.OAuth2.RedirectURI, "https://") {
		return fmt.Errorf("oauth2.redirect_uri must be a valid HTTP/HTTPS URL")
	}

	// Validate JWT fields
	if cfg.JWT.SecretKey == "" {
		return fmt.Errorf("jwt.secret_key is required")
	}
	if len(cfg.JWT.SecretKey) < 32 {
		return fmt.Errorf("jwt.secret_key must be at least 32 characters for security")
	}
	if cfg.JWT.Issuer == "" {
		return fmt.Errorf("jwt.issuer is required")
	}
	if len(cfg.JWT.Audience) == 0 {
		return fmt.Errorf("jwt.audience must have at least one value")
	}
	if cfg.JWT.TTL <= 0 {
		return fmt.Errorf("jwt.ttl must be greater than 0")
	}

	// Validate Access Control
	if len(cfg.AccessControl.AllowedDomains) == 0 {
		return fmt.Errorf("access_control.allowed_domains must have at least one domain")
	}
	// Validate domain format (Task 4.4)
	for _, domain := range cfg.AccessControl.AllowedDomains {
		if domain == "" {
			return fmt.Errorf("access_control.allowed_domains contains empty domain")
		}
	}
	// Validate default role (Task 4.4)
	validRoles := map[string]bool{"ADMIN": true, "ANALYST": true, "VIEWER": true}
	if !validRoles[cfg.AccessControl.DefaultRole] {
		return fmt.Errorf("access_control.default_role must be one of: ADMIN, ANALYST, VIEWER")
	}

	// Validate Encrypter
	if cfg.Encrypter.Key == "" {
		return fmt.Errorf("encrypter.key is required")
	}
	// Validate encrypter key length (Task 4.4)
	if len(cfg.Encrypter.Key) < 32 {
		return fmt.Errorf("encrypter.key must be at least 32 characters for security")
	}

	// Validate Database Configuration (Task 4.4)
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

	// Validate Redis Configuration (Task 4.4)
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

	// Validate Session Configuration (Task 4.4)
	if cfg.Session.TTL <= 0 {
		return fmt.Errorf("session.ttl must be greater than 0")
	}
	if cfg.Session.RememberMeTTL <= 0 {
		return fmt.Errorf("session.remember_me_ttl must be greater than 0")
	}

	// Validate Cookie Configuration (Task 4.4)
	if cfg.Cookie.Name == "" {
		return fmt.Errorf("cookie.name is required")
	}

	return nil
}
