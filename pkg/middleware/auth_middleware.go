package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/trungquantrannguyen/threadly/pkg/auth"
	"github.com/trungquantrannguyen/threadly/pkg/config"
)

const (
	UserIDKey   = "user_id"
	EmailKey    = "email"
	UsernameKey = "username"
	RoleKey     = "role"
)

func AuthMiddleware(cfg config.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")

		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Authorization header is required",
			})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid authorization header format",
			})
			return
		}

		tokenStrings := strings.TrimSpace(parts[1])

		claims, err := auth.ValidateAccessToken(tokenStrings, cfg.JWTSecret)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid or expired token",
			})
			return
		}

		ctx.Set(UserIDKey, claims.UserID)
		ctx.Set(EmailKey, claims.Email)
		ctx.Set(UsernameKey, claims.Username)
		ctx.Set(RoleKey, claims.Role)

		ctx.Next()
	}
}

func GetUserID(c *gin.Context) string {
	userID, _ := c.Get(UserIDKey)
	if value, ok := userID.(string); ok {
		return value
	}
	return ""
}

func GetUserRole(c *gin.Context) string {
	role, _ := c.Get(RoleKey)
	if value, ok := role.(string); ok {
		return value
	}
	return ""
}

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role := GetUserRole(ctx)

		if role == "" {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"message": "Role is required",
			})
			return
		}

		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				ctx.Next()
				return
			}
		}

		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"message": "You do not have permission to access this resource",
		})
	}
}
