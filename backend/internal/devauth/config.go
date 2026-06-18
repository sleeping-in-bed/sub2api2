package devauth

import (
	"os"
	"strings"
)

const (
	BypassEnabledEnv   = "DEV_AUTH_BYPASS_ENABLED"
	BypassRoleCookie   = "sub2api_dev_auth_as"
	RoleAdmin          = "admin"
	RoleUser           = "user"
	DefaultAdminEmail  = "admin@sub2api.local"
	DefaultUserEmail   = "dev+61cc3f3e5c3a40f0a16296bc@sub2api.local"
	DefaultUserPassword = "8f6a00f6e5ad0db2a44a3c0718ea613bd13ec4ab346163e940946f7f8a5f8ba9"
)

func IsBypassEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(BypassEnabledEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func NormalizeBypassRole(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case RoleUser:
		return RoleUser
	default:
		return RoleAdmin
	}
}

func AdminEmail() string {
	return envOrDefault("ADMIN_EMAIL", DefaultAdminEmail)
}

func UserEmail() string {
	return envOrDefault("DEV_USER_EMAIL", DefaultUserEmail)
}

func UserPassword() string {
	return envOrDefault("DEV_USER_PASSWORD", DefaultUserPassword)
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
