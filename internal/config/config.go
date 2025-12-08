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
	JWT      JWTConfig      `mapstructure:"jwt"`
	Server   ServerConfig   `mapstructure:"server"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	AccessTTL  int    `mapstructure:"access_ttl"`  // in minutes
	RefreshTTL int    `mapstructure:"refresh_ttl"` // in minutes
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
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
	viper.SetDefault("jwt.secret", "your-secret-key")
	viper.SetDefault("jwt.access_ttl", 15)
	viper.SetDefault("jwt.refresh_ttl", 10080)
	viper.SetDefault("server.port", "50051")

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
