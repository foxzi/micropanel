package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"micropanel/internal/config"
	"micropanel/internal/services"
)

const APITokenContextKey = "api_token"
const APITokenUserIDKey = "api_token_user_id"

// APITokenInfo holds token info for context
type APITokenInfo struct {
	Name   string
	UserID int64
}

// APIToken returns middleware that validates Bearer token from Authorization header.
// It first checks database tokens via service, then falls back to config tokens.
func APIToken(tokens []config.APIToken, tokenService *services.APITokenService) gin.HandlerFunc {
	// Build a map for O(1) lookups of config tokens
	tokenMap := make(map[string]*config.APIToken)
	for i := range tokens {
		tokenMap[tokens[i].Token] = &tokens[i]
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing Authorization header",
			})
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid Authorization header format, expected: Bearer <token>",
			})
			return
		}

		tokenString := parts[1]

		// First try database tokens via service
		if tokenService != nil {
			if dbToken, err := tokenService.ValidateToken(tokenString); err == nil {
				// Store token info in context
				c.Set(APITokenContextKey, &APITokenInfo{
					Name:   dbToken.Name,
					UserID: dbToken.UserID,
				})
				c.Set(APITokenUserIDKey, dbToken.UserID)
				c.Next()
				return
			}
		}

		// Fall back to config tokens for backward compatibility
		apiToken, exists := tokenMap[tokenString]
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		// Store token info in context for logging/audit
		c.Set(APITokenContextKey, &APITokenInfo{
			Name:   apiToken.Name,
			UserID: apiToken.UserID,
		})
		c.Set(APITokenUserIDKey, apiToken.UserID)
		c.Next()
	}
}

// GetAPIToken returns the API token info from context, if present.
func GetAPIToken(c *gin.Context) *APITokenInfo {
	if token, exists := c.Get(APITokenContextKey); exists {
		return token.(*APITokenInfo)
	}
	return nil
}

// GetAPITokenUserID returns the user ID associated with the API token.
func GetAPITokenUserID(c *gin.Context) int64 {
	if userID, exists := c.Get(APITokenUserIDKey); exists {
		return userID.(int64)
	}
	return 0
}
