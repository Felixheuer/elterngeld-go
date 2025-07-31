package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Stripe   StripeConfig
	Upload   UploadConfig
	S3       S3Config
	Email    EmailConfig
	Admin    AdminConfig
	Log      LogConfig
	Migrate  MigrateConfig
	Dev      DevConfig
	CORS     CORSConfig
	RateLimit RateLimitConfig
}

type ServerConfig struct {
	Port string
	Host string
	Env  string
}

type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	SQLitePath string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	SuccessURL    string
	CancelURL     string
}

type UploadConfig struct {
	Path              string
	MaxSize           int64
	AllowedExtensions []string
}

type S3Config struct {
	UseS3           bool
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

type EmailConfig struct {
	Provider     string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	From         string
	FromName     string
	MailgunDomain string
	MailgunAPIKey string
}

type AdminConfig struct {
	Email    string
	Password string
}

type LogConfig struct {
	Level  string
	Format string
}

type MigrateConfig struct {
	Path string
}

type DevConfig struct {
	AutoMigrate bool
	SeedData    bool
}

type CORSConfig struct {
	Origins     []string
	Credentials bool
}

type RateLimitConfig struct {
	Requests int
	Window   int
}

var Cfg *Config

func Load() error {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "localhost"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Driver:     getEnv("DB_DRIVER", "sqlite"),
			Host:       getEnv("DB_HOST", "localhost"),
			Port:       getEnv("DB_PORT", "5432"),
			User:       getEnv("DB_USER", "postgres"),
			Password:   getEnv("DB_PASSWORD", "password"),
			Name:       getEnv("DB_NAME", "elterngeld_portal"),
			SSLMode:    getEnv("DB_SSLMODE", "disable"),
			SQLitePath: getEnv("SQLITE_PATH", "./data/database.db"),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "dev-secret"),
			AccessExpiry:  parseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m")),
			RefreshExpiry: parseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h")),
		},
		Stripe: StripeConfig{
			SecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
			SuccessURL:    getEnv("STRIPE_SUCCESS_URL", "http://localhost:8080/payment/success"),
			CancelURL:     getEnv("STRIPE_CANCEL_URL", "http://localhost:8080/payment/cancel"),
		},
		Upload: UploadConfig{
			Path:              getEnv("UPLOAD_PATH", "./storage/uploads"),
			MaxSize:           parseInt64(getEnv("MAX_UPLOAD_SIZE", "10485760")),
			AllowedExtensions: strings.Split(getEnv("ALLOWED_EXTENSIONS", ".pdf,.png,.jpg,.jpeg"), ","),
		},
		S3: S3Config{
			UseS3:           parseBool(getEnv("USE_S3", "false")),
			Region:          getEnv("AWS_REGION", "eu-central-1"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			Bucket:          getEnv("S3_BUCKET", ""),
		},
		Email: EmailConfig{
			Provider:      getEnv("EMAIL_PROVIDER", "smtp"),
			SMTPHost:      getEnv("SMTP_HOST", "localhost"),
			SMTPPort:      parseInt(getEnv("SMTP_PORT", "1025")),
			SMTPUser:      getEnv("SMTP_USER", ""),
			SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
			From:          getEnv("EMAIL_FROM", "noreply@elterngeld-portal.de"),
			FromName:      getEnv("EMAIL_FROM_NAME", "Elterngeld Portal"),
			MailgunDomain: getEnv("MAILGUN_DOMAIN", ""),
			MailgunAPIKey: getEnv("MAILGUN_API_KEY", ""),
		},
		Admin: AdminConfig{
			Email:    getEnv("ADMIN_EMAIL", "admin@elterngeld-portal.de"),
			Password: getEnv("ADMIN_PASSWORD", "admin123"),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Migrate: MigrateConfig{
			Path: getEnv("MIGRATE_PATH", "./migrations"),
		},
		Dev: DevConfig{
			AutoMigrate: parseBool(getEnv("AUTO_MIGRATE", "true")),
			SeedData:    parseBool(getEnv("SEED_DATA", "true")),
		},
		CORS: CORSConfig{
			Origins:     strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:8080"), ","),
			Credentials: parseBool(getEnv("CORS_CREDENTIALS", "true")),
		},
		RateLimit: RateLimitConfig{
			Requests: parseInt(getEnv("RATE_LIMIT_REQUESTS", "100")),
			Window:   parseInt(getEnv("RATE_LIMIT_WINDOW", "60")),
		},
	}

	Cfg = cfg
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func parseInt64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func parseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return b
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Hour
	}
	return d
}

// ParseDuration is a public wrapper for parseDuration
func ParseDuration(s string) time.Duration {
	return parseDuration(s)
}

func (c *Config) GetDSN() string {
	switch c.Database.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			c.Database.Host,
			c.Database.Port,
			c.Database.User,
			c.Database.Password,
			c.Database.Name,
			c.Database.SSLMode,
		)
	case "sqlite":
		return c.Database.SQLitePath
	default:
		return c.Database.SQLitePath
	}
}

func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}