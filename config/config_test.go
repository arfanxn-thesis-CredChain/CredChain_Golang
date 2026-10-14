package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_KEY_EMPTY", "")
	fb := "fallback"
	got := getEnv("TEST_KEY_EMPTY", &fb)
	assert.Equal(t, &fb, got)
}

func TestGetEnv_Set_ReturnsValue(t *testing.T) {
	t.Setenv("TEST_KEY_SET", "hello")
	got := getEnv("TEST_KEY_SET", nil)
	assert.NotNil(t, got)
	assert.Equal(t, "hello", *got)
}

func TestGetIntEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_INT_EMPTY", "")
	fb := 42
	got := getIntEnv("TEST_INT_EMPTY", &fb)
	assert.Equal(t, 42, *got)
}

func TestGetIntEnv_Valid(t *testing.T) {
	t.Setenv("TEST_INT_VALID", "99")
	got := getIntEnv("TEST_INT_VALID", nil)
	assert.Equal(t, 99, *got)
}

func TestGetIntEnv_Invalid_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_INT_INVALID", "not-a-number")
	fb := 7
	got := getIntEnv("TEST_INT_INVALID", &fb)
	assert.Equal(t, 7, *got)
}

func TestGetBoolEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_BOOL_EMPTY", "")
	fb := true
	got := getBoolEnv("TEST_BOOL_EMPTY", &fb)
	assert.True(t, *got)
}

func TestGetBoolEnv_True(t *testing.T) {
	t.Setenv("TEST_BOOL_TRUE", "true")
	got := getBoolEnv("TEST_BOOL_TRUE", nil)
	assert.True(t, *got)
}

func TestGetBoolEnv_Other_ReturnsFalse(t *testing.T) {
	t.Setenv("TEST_BOOL_OTHER", "yes")
	got := getBoolEnv("TEST_BOOL_OTHER", nil)
	assert.False(t, *got)
}

func TestGetStringSliceEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_SLICE_EMPTY", "")
	fb := []string{"a", "b"}
	got := getStringSliceEnv("TEST_SLICE_EMPTY", fb)
	assert.Equal(t, fb, got)
}

func TestGetStringSliceEnv_CommaSplit(t *testing.T) {
	t.Setenv("TEST_SLICE_SET", "x,y,z")
	got := getStringSliceEnv("TEST_SLICE_SET", nil)
	assert.Equal(t, []string{"x", "y", "z"}, got)
}

func TestGetTimeEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_TIME_EMPTY", "")
	got := getTimeEnv("TEST_TIME_EMPTY", nil)
	assert.Nil(t, got)
}

func TestGetTimeEnv_Valid(t *testing.T) {
	t.Setenv("TEST_TIME_VALID", "1990-01-15")
	got := getTimeEnv("TEST_TIME_VALID", nil)
	assert.NotNil(t, got)
	assert.Equal(t, 1990, got.Year())
	assert.Equal(t, 15, got.Day())
}

func TestGetTimeEnv_Malformed_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_TIME_BAD", "not-a-date")
	got := getTimeEnv("TEST_TIME_BAD", nil)
	assert.Nil(t, got)
}

func TestGetJSONEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_JSON_EMPTY", "")
	got := getJSONEnv("TEST_JSON_EMPTY", nil)
	assert.Nil(t, got)
}

func TestGetJSONEnv_Valid(t *testing.T) {
	t.Setenv("TEST_JSON_VALID", `{"key":"value","num":42}`)
	got := getJSONEnv("TEST_JSON_VALID", nil)
	assert.Equal(t, "value", got["key"])
	assert.Equal(t, float64(42), got["num"])
}

func TestGetJSONEnv_Invalid_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_JSON_INVALID", "not-json")
	got := getJSONEnv("TEST_JSON_INVALID", nil)
	assert.Nil(t, got)
}

func TestNewConfig_MissingJWTSecret_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("WALLET_ENCRYPTION_KEY", "some-key")
	t.Setenv("FILE_ENCRYPTION_KEY", "file-encryption-key-32chars-xxxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
}

func TestNewConfig_MissingWalletKey_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "")
	t.Setenv("FILE_ENCRYPTION_KEY", "file-encryption-key-32chars-xxxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
}

func TestNewConfig_BothSet_ReturnsConfig(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	cfg, err := NewConfig(".env.nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, "my-secret", *cfg.JWTSecret)
	assert.Equal(t, "my-wallet-key-exactly-32-chars-x", *cfg.WalletEncryptionKey)
	assert.Equal(t, "my-file-key-exactly-32-chars-xxx", *cfg.FileEncryptionKey)
}

func TestNewConfig_WalletKeyTooShort_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "too-short")
	t.Setenv("FILE_ENCRYPTION_KEY", "file-encryption-key-32chars-xxxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestNewConfig_WalletKeyTooLong_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "this-is-a-64-char-hex-encoded-key-that-should-fail-validation-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "file-encryption-key-32chars-xxxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestNewConfig_MissingFileEncryptionKey_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file_encryption_key is required")
}

func TestNewConfig_FileEncryptionKeyTooShort_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "too-short")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file_encryption_key must be exactly 32 bytes")
}

func TestNewConfig_MissingIssuingOrganizationName_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "issuing_organization_name is required")
}

func TestNewConfig_IssuingOrganizationName_Set(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "University of Indonesia")
	cfg, err := NewConfig(".env.nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, "University of Indonesia", *cfg.IssuingOrganizationName)
}

func TestNewConfig_MailConfig_DisabledWhenHostEmpty(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	t.Setenv("MAIL_HOST", "")
	t.Setenv("SMTP_HOST", "")

	cfg, err := NewConfig(".env.nonexistent")
	assert.NoError(t, err)
	assert.False(t, cfg.IsMailEnabled())
}

func TestNewConfig_MailConfig_MissingFrom_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	t.Setenv("MAIL_HOST", "sandbox.smtp.mailtrap.io")
	t.Setenv("APP_BASE_URL", "http://localhost:5173")
	t.Setenv("MAIL_FROM", "")
	t.Setenv("SMTP_FROM", "")

	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mail_from")
}

func TestNewConfig_MailConfig_InvalidFrom_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	t.Setenv("MAIL_HOST", "sandbox.smtp.mailtrap.io")
	t.Setenv("MAIL_FROM", "not-an-email")
	t.Setenv("APP_BASE_URL", "http://localhost:5173")

	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mail_from")
}

func TestNewConfig_MailConfig_MissingAppBaseURL_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	t.Setenv("MAIL_HOST", "sandbox.smtp.mailtrap.io")
	t.Setenv("MAIL_FROM", "noreply@credchain.com")
	t.Setenv("APP_BASE_URL", "")

	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app_base_url")
}

func TestNewConfig_MailConfig_InvalidAppBaseURL_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	t.Setenv("MAIL_HOST", "sandbox.smtp.mailtrap.io")
	t.Setenv("MAIL_FROM", "noreply@credchain.com")
	t.Setenv("APP_BASE_URL", "http://localhost:5173/path?query=1#frag")

	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app_base_url")
}

func TestNewConfig_MailConfig_InvalidPort_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	t.Setenv("MAIL_HOST", "sandbox.smtp.mailtrap.io")
	t.Setenv("MAIL_PORT", "0")
	t.Setenv("MAIL_FROM", "noreply@credchain.com")
	t.Setenv("APP_BASE_URL", "http://localhost:5173")

	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mail_port")
}

func TestNewConfig_MailConfig_MismatchedAuth_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	t.Setenv("MAIL_HOST", "sandbox.smtp.mailtrap.io")
	t.Setenv("MAIL_PORT", "2525")
	t.Setenv("MAIL_FROM", "noreply@credchain.com")
	t.Setenv("APP_BASE_URL", "http://localhost:5173")
	t.Setenv("MAIL_USERNAME", "someuser")
	t.Setenv("MAIL_PASSWORD", "")

	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mail_password")
}

func TestNewConfig_MailConfig_ValidConfig(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "Test Org")
	t.Setenv("MAIL_HOST", "sandbox.smtp.mailtrap.io")
	t.Setenv("MAIL_PORT", "2525")
	t.Setenv("MAIL_USERNAME", "test-username")
	t.Setenv("MAIL_PASSWORD", "test-password")
	t.Setenv("MAIL_FROM", "noreply@credchain.com")
	t.Setenv("APP_BASE_URL", "http://localhost:5173")

	cfg, err := NewConfig(".env.nonexistent")
	assert.NoError(t, err)
	assert.True(t, cfg.IsMailEnabled())
	assert.Equal(t, "sandbox.smtp.mailtrap.io", *cfg.MailHost)
	assert.Equal(t, 2525, *cfg.MailPort)
	assert.Equal(t, "test-username", *cfg.MailUsername)
	assert.Equal(t, "test-password", *cfg.MailPassword)
	assert.Equal(t, "noreply@credchain.com", *cfg.MailFrom)
	assert.Equal(t, "http://localhost:5173", *cfg.AppBaseURL)
}
