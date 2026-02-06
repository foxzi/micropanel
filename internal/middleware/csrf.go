package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CSRFTokenKey    = "csrf_token"
	CSRFCookieKey   = "_csrf"
	CSRFHeaderKey   = "X-CSRF-Token"
	CSRFFormKey     = "_csrf"
	CSRFTokenLength = 32
)

func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get or generate CSRF token
		token, err := c.Cookie(CSRFCookieKey)
		if err != nil || len(token) != CSRFTokenLength*2 {
			token = generateCSRFToken()
			c.SetCookie(CSRFCookieKey, token, 86400, "/", "", false, true)
		}

		// Store token in context for templates
		c.Set(CSRFTokenKey, token)

		// Skip validation for safe methods
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// Validate CSRF token for state-changing methods
		requestToken := c.PostForm(CSRFFormKey)
		if requestToken == "" {
			requestToken = c.GetHeader(CSRFHeaderKey)
		}

		if !validateCSRFToken(token, requestToken) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid CSRF token"})
			return
		}

		c.Next()
	}
}

func GetCSRFToken(c *gin.Context) string {
	if token, exists := c.Get(CSRFTokenKey); exists {
		return token.(string)
	}
	return ""
}

func generateCSRFToken() string {
	bytes := make([]byte, CSRFTokenLength)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func validateCSRFToken(expected, actual string) bool {
	if len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
