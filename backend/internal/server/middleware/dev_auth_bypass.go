package middleware

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/devauth"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type devAuthBypassUserReader interface {
	GetByEmail(ctx context.Context, email string) (*service.User, error)
}

func tryApplyDevAuthBypass(c *gin.Context, userService devAuthBypassUserReader, requireAdmin bool) (enabled bool, ok bool) {
	if !devauth.IsBypassEnabled() {
		return false, false
	}

	if userService == nil {
		AbortWithError(c, 500, "DEV_AUTH_BYPASS_MISCONFIGURED", "Dev auth bypass requires user service")
		return true, false
	}

	selectedRole := resolveDevAuthBypassRole(c)
	user, err := loadDevAuthBypassUser(c.Request.Context(), userService, selectedRole)
	if err != nil {
		AbortWithError(c, 500, devAuthBypassUserNotFoundCode(selectedRole), "Dev auth bypass could not load configured user")
		return true, false
	}

	if !user.IsActive() {
		AbortWithError(c, 401, "USER_INACTIVE", "Selected dev bypass user account is not active")
		return true, false
	}

	if requireAdmin && !user.IsAdmin() {
		AbortWithError(c, 403, "FORBIDDEN", "Admin access required")
		return true, false
	}

	c.Set(string(ContextKeyUser), AuthSubject{
		UserID:      user.ID,
		Concurrency: user.Concurrency,
	})
	c.Set(string(ContextKeyUserRole), user.Role)
	c.Set("auth_method", "dev_auth_bypass")
	c.Set("dev_auth_bypass_role", selectedRole)

	return true, true
}

func resolveDevAuthBypassRole(c *gin.Context) string {
	if c == nil {
		return devauth.RoleAdmin
	}

	rawRole, err := c.Cookie(devauth.BypassRoleCookie)
	if err != nil {
		return devauth.RoleAdmin
	}

	return devauth.NormalizeBypassRole(rawRole)
}

func loadDevAuthBypassUser(ctx context.Context, userService devAuthBypassUserReader, selectedRole string) (*service.User, error) {
	if selectedRole == devauth.RoleUser {
		return userService.GetByEmail(ctx, devauth.UserEmail())
	}
	return userService.GetByEmail(ctx, devauth.AdminEmail())
}

func devAuthBypassUserNotFoundCode(selectedRole string) string {
	if selectedRole == devauth.RoleUser {
		return "DEV_AUTH_BYPASS_USER_NOT_FOUND"
	}
	return "DEV_AUTH_BYPASS_ADMIN_NOT_FOUND"
}
