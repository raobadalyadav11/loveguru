package config

import (
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Server   ServerConfig   `mapstructure:"server"`
	Agora    AgoraConfig    `mapstructure:"agora"`
	OpenAI   OpenAIConfig   `mapstructure:"openai"`
	FCM      FCMConfig      `mapstructure:"fcm"`
	APNS     APNSConfig     `mapstructure:"apns"`
	Email    EmailConfig    `mapstructure:"email"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	AccessTTL  int    `mapstructure:"access_ttl"`  // in minutes
	RefreshTTL int    `mapstructure:"refresh_ttl"` // in minutes
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type AgoraConfig struct {
	AppID    string `mapstructure:"app_id"`
	AppCert  string `mapstructure:"app_cert"`
	TokenTTL int    `mapstructure:"token_ttl"` // Token expiration time in seconds
}

type OpenAIConfig struct {
	APIKey    string `mapstructure:"api_key"`
	BaseURL   string `mapstructure:"base_url"`
	Model     string `mapstructure:"model"`
	MaxTokens int    `mapstructure:"max_tokens"`
}

type FCMConfig struct {
	ServerKey string `mapstructure:"server_key"`
	ProjectID string `mapstructure:"project_id"`
}

type APNSConfig struct {
	TeamID      string `mapstructure:"team_id"`
	KeyID       string `mapstructure:"key_id"`
	PrivateKey  string `mapstructure:"private_key"`
	BundleID    string `mapstructure:"bundle_id"`
	Environment string `mapstructure:"environment"` // "development" or "production"
}

type EmailConfig struct {
	From     string `mapstructure:"from"`
	Password string `mapstructure:"password"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
}

func Load() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "password")
	viper.SetDefault("database.dbname", "loveguru")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("jwt.secret", "your-secret-key")
	viper.SetDefault("jwt.access_ttl", 15)
	viper.SetDefault("jwt.refresh_ttl", 10080)
	viper.SetDefault("server.port", "50051")
	viper.SetDefault("agora.app_id", "")
	viper.SetDefault("agora.app_cert", "")
	viper.SetDefault("agora.token_ttl", 3600) // 1 hour
	viper.SetDefault("openai.api_key", "")
	viper.SetDefault("openai.base_url", "https://api.openai.com")
	viper.SetDefault("openai.model", "gpt-3.5-turbo")
	viper.SetDefault("openai.max_tokens", 500)
	viper.SetDefault("fcm.server_key", "")
	viper.SetDefault("fcm.project_id", "")
	viper.SetDefault("apns.team_id", "")
	viper.SetDefault("apns.key_id", "")
	viper.SetDefault("apns.private_key", "")
	viper.SetDefault("apns.bundle_id", "")
	viper.SetDefault("apns.environment", "development")
	viper.SetDefault("email.from", "")
	viper.SetDefault("email.password", "")
	viper.SetDefault("email.host", "smtp.gmail.com")
	viper.SetDefault("email.port", "587")

	if err := viper.ReadInConfig(); err != nil {
		// Use defaults if config file not found
	}

	// Check for DATABASE_URL environment variable
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		if err := parseDatabaseURL(dbURL); err != nil {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func parseDatabaseURL(dbURL string) error {
	u, err := url.Parse(dbURL)
	if err != nil {
		return err
	}

	if u.Hostname() != "" {
		viper.Set("database.host", u.Hostname())
	}
	if u.Port() != "" {
		if port, err := strconv.Atoi(u.Port()); err == nil {
			viper.Set("database.port", port)
		}
	}
	if u.User.Username() != "" {
		viper.Set("database.user", u.User.Username())
	}
	if password, ok := u.User.Password(); ok {
		viper.Set("database.password", password)
	}
	if len(u.Path) > 1 {
		viper.Set("database.dbname", u.Path[1:])
	}

	// Check for sslmode in query params
	if sslmode := u.Query().Get("sslmode"); sslmode != "" {
		viper.Set("database.sslmode", sslmode)
	}

	return nil
}
