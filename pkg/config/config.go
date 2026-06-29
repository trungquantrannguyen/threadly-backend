package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	AppEnv string

	ServiceName string
	Port        string
	LogLevel    string

	DatabaseURL string
	RedisAddr   string
	RabbitMQURL string

	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	UserServiceGRPCAddr         string
	ContentServiceGRPCAddr      string
	FeedServiceGRPCAddr         string
	NotificationServiceGRPCAddr string
	StorageServiceGRPCAddr      string

	SupabaseStorageBucket     string
	SupabaseS3Endpoint        string
	SupabaseS3Region          string
	SupabaseS3AccessKeyID     string
	SupabaseS3SecretAccessKey string
	SupabasePublicStorageURL  string
}

func Load(serviceName string, defaultPort string) Config {
	err := godotenv.Load()
	if err != nil {
		log.Info().Str("Info", "No .env file found, Using system enviroment variables")
	}

	return Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		ServiceName: serviceName,
		Port:        getEnv(serviceNameToPortKey(serviceName), defaultPort),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),

		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		JWTSecret:       getEnv("JWT_SECRET", "development_secret"),
		AccessTokenTTL:  getDurationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getDurationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour),

		UserServiceGRPCAddr:         getEnv("USER_SERVICE_GRPC_ADDR", "localhost:50051"),
		ContentServiceGRPCAddr:      getEnv("CONTENT_SERVICE_GRPC_ADDR", "localhost:50052"),
		FeedServiceGRPCAddr:         getEnv("FEED_SERVICE_GRPC_ADDR", "localhost:50053"),
		NotificationServiceGRPCAddr: getEnv("NOTIFICATION_SERVICE_GRPC_ADDR", "localhost:50054"),
		StorageServiceGRPCAddr:      getEnv("STORAGE_SERVICE_GRPC_ADDR", "localhost:50055"),

		SupabaseStorageBucket:     getEnv("SUPABASE_STORAGE_BUCKET", "threadly-media"),
		SupabaseS3Endpoint:        getEnv("SUPABASE_S3_ENDPOINT", ""),
		SupabaseS3Region:          getEnv("SUPABASE_S3_REGION", ""),
		SupabaseS3AccessKeyID:     getEnv("SUPABASE_S3_ACCESS_KEY_ID", ""),
		SupabaseS3SecretAccessKey: getEnv("SUPABASE_S3_SECRET_ACCESS_KEY", ""),
		SupabasePublicStorageURL:  getEnv("SUPABASE_PUBLIC_STORAGE_URL", ""),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

func serviceNameToPortKey(serviceName string) string {
	switch serviceName {
	case "api-gateway":
		return "API_GATEWAY_PORT"
	case "user-service":
		return "USER_SERVICE_GRPC_PORT"
	case "content-service":
		return "CONTENT_SERVICE_GRPC_PORT"
	case "feed-service":
		return "FEED_SERVICE_GRPC_PORT"
	case "storage-service":
		return "STORAGE_SERVICE_GRPC_PORT"
	case "notification-service":
		return "NOTIFICATION_SERVICE_GRPC_PORT"
	default:
		return "PORT"
	}
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Warn().
			Str("key", key).
			Str("value", value).
			Msg("Invalid duration env value, using fallback")

		return fallback
	}

	return duration
}
