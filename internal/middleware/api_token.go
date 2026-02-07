package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"micropanel/internal/config"
)

const APITokenContextKey = "api_token"

// APIToken returns middleware that validates Bearer token from Authorization header.
func APIToken(tokens []config.APIToken) gin.HandlerFunc {
	// Build a map for O(1) lookups
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

		token := parts[1]
		apiToken, exists := tokenMap[token]
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		// Store token info in context for logging/audit
		c.Set(APITokenContextKey, apiToken)
		c.Next()
	}
}

// GetAPIToken returns the API token from context, if present.
func GetAPIToken(c *gin.Context) *config.APIToken {
	if token, exists := c.Get(APITokenContextKey); exists {
		return token.(*config.APIToken)
	}
	return nil
}
