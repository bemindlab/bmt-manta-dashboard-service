package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config เก็บการตั้งค่าทั้งหมดของแอปพลิเคชัน
type Config struct {
	// การตั้งค่าทั่วไป
	Port       string
	Env        string

	// การตั้งค่า PostgreSQL
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	// การตั้งค่า Firebase
	FirebaseProjectID      string
	FirebaseCredentialsFile string

	// การตั้งค่า Firebase Storage
	FirebaseStorageBucket string

	// การตั้งค่า S3 (สำหรับเก็บรูปภาพ)
	S3Enabled      bool
	S3Endpoint     string
	S3Region       string
	S3Bucket       string
	S3AccessKey    string
	S3SecretKey    string
	S3UsePathStyle bool
	
	// การตั้งค่า Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// การตั้งค่าการยืนยันตัวตน
	JWTSecret    string
	JWTExpiresIn time.Duration
	APIKey       string

	// การตั้งค่า Rate Limiting
	RateLimitMax      int
	RateLimitDuration time.Duration
}

// Load โหลดการตั้งค่าจากไฟล์ .env และตัวแปรสภาพแวดล้อม
func Load() (*Config, error) {
	// โหลดไฟล์ .env ถ้ามี
	godotenv.Load()

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	rateLimitMax, _ := strconv.Atoi(getEnv("RATE_LIMIT_MAX", "100"))
	
	jwtExpiration, _ := time.ParseDuration(getEnv("JWT_EXPIRES_IN", "24h"))
	rateLimitDuration, _ := time.ParseDuration(getEnv("RATE_LIMIT_DURATION", "60s"))

	s3Enabled, _ := strconv.ParseBool(getEnv("S3_ENABLED", "false"))
	s3UsePathStyle, _ := strconv.ParseBool(getEnv("S3_USE_PATH_STYLE", "false"))

	return &Config{
		// การตั้งค่าทั่วไป
		Port:       getEnv("PORT", "8080"),
		Env:        getEnv("ENV", "development"),

		// การตั้งค่า PostgreSQL
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:       getEnv("POSTGRES_DB", "manta_dashboard"),
		PostgresSSLMode:  getEnv("POSTGRES_SSL_MODE", "disable"),

		// การตั้งค่า Firebase
		FirebaseProjectID:      getEnv("FIREBASE_PROJECT_ID", ""),
		FirebaseCredentialsFile: getEnv("FIREBASE_CREDENTIALS_FILE", "./config/firebase-credentials.json"),
		
		// การตั้งค่า Firebase Storage
		FirebaseStorageBucket: getEnv("FIREBASE_STORAGE_BUCKET", ""),

		// การตั้งค่า S3 (สำหรับเก็บรูปภาพ)
		S3Enabled:      s3Enabled,
		S3Endpoint:     getEnv("S3_ENDPOINT", ""),
		S3Region:       getEnv("S3_REGION", ""),
		S3Bucket:       getEnv("S3_BUCKET", ""),
		S3AccessKey:    getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:    getEnv("S3_SECRET_KEY", ""),
		S3UsePathStyle: s3UsePathStyle,

		// การตั้งค่า Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,

		// การตั้งค่าการยืนยันตัวตน
		JWTSecret:    getEnv("JWT_SECRET", "default-jwt-secret"),
		JWTExpiresIn: jwtExpiration,
		APIKey:       getEnv("API_KEY", ""),

		// การตั้งค่า Rate Limiting
		RateLimitMax:      rateLimitMax,
		RateLimitDuration: rateLimitDuration,
	}, nil
}

// getEnv รับค่าจากตัวแปรสภาพแวดล้อม หรือใช้ค่าเริ่มต้นถ้าไม่มี
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
} 