package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/samber/lo"
)

type Config struct {
	GinPort                            *string
	InitialSuperAdminEmail             *string
	InitialSuperAdminPrivKey           *string
	InitialSuperAdminName              *string
	InitialSuperAdminNumber            *string
	InitialSuperAdminBirthDate         *time.Time
	InitialSuperAdminGender            *string
	InitialSuperAdminMeta              map[string]any
	WalletEncryptionKey                *string
	FileEncryptionKey                  *string
	RPCURL                             *string
	RelayerPrivateKey                  *string
	HardhatMnemonic                    *string
	AuthorityContract                  *string
	RegistryContract                   *string
	IssuingOrganizationName            *string
	JWTSecret                          *string
	JWTAccessExpiryMinutes             *int
	JWTRefreshExpiryHours              *int
	PostgresUser                       *string
	PostgresPassword                   *string
	PostgresDB                         *string
	PostgresDSN                        *string
	DBMaxOpenConns                     *int
	DBMaxIdleConns                     *int
	DBConnMaxLifetime                  *int
	DBConnMaxIdleTime                  *int
	MongoInitDBUsername                *string
	MongoInitPassword                  *string
	MongoURI                           *string
	MongoDatabase                      *string
	AIVerificationCacheTTLHours        *int
	RiverMaxWorkers                    *int
	GoogleClientID                     *string
	GoogleClientSecret                 *string
	GoogleRedirectURI                  *string
	GinCorsAllowOrigins                []string
	GinCorsAllowMethods                []string
	GinCorsAllowHeaders                []string
	GinCorsExposeHeaders               []string
	GinCorsAllowCredentials            *bool
	GinCorsMaxAge                      time.Duration
	LogLevel                           *string
	LogOutput                          *string
	I18nLocalesDir                     *string
	CookieDomain                       *string
	CookieSecure                       *bool
	CookieSameSite                     *string
	CookieAccessPath                   *string
	CookieRefreshPath                  *string
	PythonAIBaseURL                    *string
	PythonAIAPIKey                     *string
	PythonAITimeoutSeconds             *int
	CredentialExtractWorkerCount       *int
	CredentialExtractWorkerPollSeconds *int
	CredentialExtractWorkerMaxAttempts *int
	StoragePath                        *string
	CredentialFileStoragePath          *string
}

func getIntEnv(key string, defaultVal *int) *int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return &intVal
}

func getStringSliceEnv(key string, defaultVal []string) []string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return strings.Split(val, ",")
}

func getBoolEnv(key string, defaultVal *bool) *bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	result := val == "true"
	return &result
}

func NewConfig(envPath string) (*Config, error) {
	err := godotenv.Load(envPath)
	if err != nil {
		// Continue without .env file (env vars may be set directly)
	}

	defaultJWTAccessExpiry := 15
	defaultJWTRefreshExpiry := 168
	defaultDBMaxOpenConns := 25
	defaultDBMaxIdleConns := 25
	defaultDBConnMaxLifetime := 5
	defaultDBConnMaxIdleTime := 1
	defaultGinCorsMaxAge := 43200
	defaultGinCorsAllowCredentials := true
	defaultCORSOrigins := []string{"*"}
	defaultCORSMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	defaultCORSHeaders := []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	defaultCORSExposeHeaders := []string{"Content-Length"}
	defaultGoogleRedirectURI := "http://localhost:3000/google/callback"
	defaultCookieSecure := false
	defaultCookieSameSite := "strict"
	defaultCookieAccessPath := "/api"
	defaultCookieRefreshPath := "/api/auth"
	defaultMongoDatabase := "credchain"
	defaultAIVerificationCacheTTLHours := 24
	defaultRiverMaxWorkers := 10
	defaultPythonAIBaseURL := "http://localhost:8081"
	defaultPythonAITimeout := 120
	defaultCredentialExtractWorkerCount := 1
	defaultCredentialExtractWorkerPoll := 2
	defaultCredentialExtractWorkerMaxAttempts := 3
	defaultStoragePath := "uploads"
	defaultCredentialFileStoragePath := "credentials"

	cfg := &Config{
		GinPort:                            getEnv("GIN_PORT", lo.ToPtr("8080")),
		InitialSuperAdminEmail:             getEnv("INITIAL_SUPER_ADMIN_EMAIL", nil),
		InitialSuperAdminPrivKey:           getEnv("INITIAL_SUPER_ADMIN_PRIVATE_KEY", nil),
		InitialSuperAdminName:              getEnv("INITIAL_SUPER_ADMIN_NAME", nil),
		InitialSuperAdminNumber:            getEnv("INITIAL_SUPER_ADMIN_NUMBER", nil),
		InitialSuperAdminBirthDate:         getTimeEnv("INITIAL_SUPER_ADMIN_BIRTH_DATE", nil),
		InitialSuperAdminGender:            getEnv("INITIAL_SUPER_ADMIN_GENDER", nil),
		InitialSuperAdminMeta:              getJSONEnv("INITIAL_SUPER_ADMIN_META", nil),
		WalletEncryptionKey:                getEnv("WALLET_ENCRYPTION_KEY", nil),
		FileEncryptionKey:                  getEnv("FILE_ENCRYPTION_KEY", nil),
		RPCURL:                             getEnv("RPC_URL", nil),
		RelayerPrivateKey:                  getEnv("RELAYER_PRIVATE_KEY", nil),
		HardhatMnemonic:                    getEnv("HARDHAT_MNEMONIC", nil),
		AuthorityContract:                  getEnv("AUTHORITY_CONTRACT", nil),
		RegistryContract:                   getEnv("REGISTRY_CONTRACT", nil),
		IssuingOrganizationName:            getEnv("ISSUING_ORGANIZATION_NAME", nil),
		JWTSecret:                          getEnv("JWT_SECRET", nil),
		JWTAccessExpiryMinutes:             getIntEnv("JWT_ACCESS_EXPIRY_MINUTES", &defaultJWTAccessExpiry),
		JWTRefreshExpiryHours:              getIntEnv("JWT_REFRESH_EXPIRY_HOURS", &defaultJWTRefreshExpiry),
		PostgresUser:                       getEnv("POSTGRES_USER", nil),
		PostgresPassword:                   getEnv("POSTGRES_PASSWORD", nil),
		PostgresDB:                         getEnv("POSTGRES_DB", nil),
		PostgresDSN:                        getEnv("POSTGRES_DSN", nil),
		DBMaxOpenConns:                     getIntEnv("DB_MAX_OPEN_CONNS", &defaultDBMaxOpenConns),
		DBMaxIdleConns:                     getIntEnv("DB_MAX_IDLE_CONNS", &defaultDBMaxIdleConns),
		DBConnMaxLifetime:                  getIntEnv("DB_CONN_MAX_LIFETIME", &defaultDBConnMaxLifetime),
		DBConnMaxIdleTime:                  getIntEnv("DB_CONN_MAX_IDLE_TIME", &defaultDBConnMaxIdleTime),
		MongoInitDBUsername:                getEnv("MONGO_INIT_DB_USERNAME", nil),
		MongoInitPassword:                  getEnv("MONGO_INITDB_ROOT_PASSWORD", nil),
		MongoURI:                           getEnv("MONGO_URI", nil),
		MongoDatabase:                      getEnv("MONGO_DATABASE", &defaultMongoDatabase),
		AIVerificationCacheTTLHours:        getIntEnv("AI_VERIFICATION_CACHE_TTL_HOURS", &defaultAIVerificationCacheTTLHours),
		RiverMaxWorkers:                    getIntEnv("RIVER_MAX_WORKERS", &defaultRiverMaxWorkers),
		GoogleClientID:                     getEnv("GOOGLE_CLIENT_ID", nil),
		GoogleClientSecret:                 getEnv("GOOGLE_CLIENT_SECRET", nil),
		GoogleRedirectURI:                  getEnv("GOOGLE_REDIRECT_URI", &defaultGoogleRedirectURI),
		GinCorsAllowOrigins:                getStringSliceEnv("GIN_CORS_ALLOW_ORIGINS", defaultCORSOrigins),
		GinCorsAllowMethods:                getStringSliceEnv("GIN_CORS_ALLOW_METHODS", defaultCORSMethods),
		GinCorsAllowHeaders:                getStringSliceEnv("GIN_CORS_ALLOW_HEADERS", defaultCORSHeaders),
		GinCorsExposeHeaders:               getStringSliceEnv("GIN_CORS_EXPOSE_HEADERS", defaultCORSExposeHeaders),
		GinCorsAllowCredentials:            getBoolEnv("GIN_CORS_ALLOW_CREDENTIALS", &defaultGinCorsAllowCredentials),
		GinCorsMaxAge:                      time.Duration(*getIntEnv("GIN_CORS_MAX_AGE", &defaultGinCorsMaxAge)) * time.Second,
		LogLevel:                           getEnv("LOG_LEVEL", lo.ToPtr("info")),
		LogOutput:                          getEnv("LOG_OUTPUT", lo.ToPtr("stdout")),
		I18nLocalesDir:                     getEnv("I18N_LOCALES_DIR", lo.ToPtr("./locales")),
		CookieDomain:                       getEnv("COOKIE_DOMAIN", lo.ToPtr("")),
		CookieSecure:                       getBoolEnv("COOKIE_SECURE", &defaultCookieSecure),
		CookieSameSite:                     getEnv("COOKIE_SAMESITE", &defaultCookieSameSite),
		CookieAccessPath:                   getEnv("COOKIE_ACCESS_PATH", &defaultCookieAccessPath),
		CookieRefreshPath:                  getEnv("COOKIE_REFRESH_PATH", &defaultCookieRefreshPath),
		PythonAIBaseURL:                    getEnv("PYTHON_AI_BASE_URL", &defaultPythonAIBaseURL),
		PythonAIAPIKey:                     getEnv("PYTHON_AI_API_KEY", nil),
		PythonAITimeoutSeconds:             getIntEnv("PYTHON_AI_TIMEOUT_SECONDS", &defaultPythonAITimeout),
		CredentialExtractWorkerCount:       getIntEnv("CREDENTIAL_EXTRACT_WORKER_COUNT", &defaultCredentialExtractWorkerCount),
		CredentialExtractWorkerPollSeconds: getIntEnv("CREDENTIAL_EXTRACT_WORKER_POLL_SECONDS", &defaultCredentialExtractWorkerPoll),
		CredentialExtractWorkerMaxAttempts: getIntEnv("CREDENTIAL_EXTRACT_WORKER_MAX_ATTEMPTS", &defaultCredentialExtractWorkerMaxAttempts),
		StoragePath:                        getEnv("STORAGE_PATH", &defaultStoragePath),
		CredentialFileStoragePath:          getEnv("CREDENTIAL_FILE_STORAGE_PATH", &defaultCredentialFileStoragePath),
	}

	if cfg.JWTSecret == nil {
		return nil, fmt.Errorf("jwt_secret is required")
	}

	if cfg.WalletEncryptionKey == nil {
		return nil, fmt.Errorf("wallet_encryption_key is required")
	}

	if keyLen := len([]byte(*cfg.WalletEncryptionKey)); keyLen != 32 {
		return nil, fmt.Errorf("wallet_encryption_key must be exactly 32 bytes (AES-256), got %d", keyLen)
	}

	if cfg.FileEncryptionKey == nil {
		return nil, fmt.Errorf("file_encryption_key is required")
	}
	if keyLen := len([]byte(*cfg.FileEncryptionKey)); keyLen != 32 {
		return nil, fmt.Errorf("file_encryption_key must be exactly 32 bytes (AES-256), got %d", keyLen)
	}

	if cfg.IssuingOrganizationName == nil || *cfg.IssuingOrganizationName == "" {
		return nil, fmt.Errorf("issuing_organization_name is required")
	}

	if cfg.CookieSameSite != nil {
		ss := strings.ToLower(*cfg.CookieSameSite)
		if ss != "strict" && ss != "lax" && ss != "none" {
			return nil, fmt.Errorf("cookie_samesite must be one of: strict, lax, none (got %q)", *cfg.CookieSameSite)
		}
		if ss == "none" && (cfg.CookieSecure == nil || !*cfg.CookieSecure) {
			return nil, fmt.Errorf("cookie_samesite=none requires cookie_secure=true")
		}
	}

	return cfg, nil
}

func getEnv(key string, fallback *string) *string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return &val
}

// getTimeEnv parses an environment variable as a time.Time in ISO 8601 format (YYYY-MM-DD).
// Returns nil if the environment variable is empty or parsing fails.
func getTimeEnv(key string, fallback *time.Time) *time.Time {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	t, err := time.Parse(time.DateOnly, val)
	if err != nil {
		return fallback
	}
	return &t
}

// getJSONEnv parses an environment variable as a JSON object into map[string]any.
// Returns nil if the environment variable is empty or parsing fails.
func getJSONEnv(key string, fallback map[string]any) map[string]any {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return fallback
	}
	return result
}
