package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"micropanel/internal/models"
	"micropanel/internal/services"
)

const UserContextKey = "user"

func Auth(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(services.SessionCookieKey)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user, err := authService.ValidateSession(sessionID)
		if err != nil {
			// Detect if running behind HTTPS
			secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
			http.SetCookie(c.Writer, &http.Cookie{
				Name:     services.SessionCookieKey,
				Value:    "",
				MaxAge:   -1,
				Path:     "/",
				Secure:   secure,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set(UserContextKey, user)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil || !user.IsAdmin() {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func GetUser(c *gin.Context) *models.User {
	if user, exists := c.Get(UserContextKey); exists {
		return user.(*models.User)
	}
	return nil
}

func OptionalAuth(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(services.SessionCookieKey)
		if err != nil {
			c.Next()
			return
		}

		user, err := authService.ValidateSession(sessionID)
		if err == nil {
			c.Set(UserContextKey, user)
		}
		c.Next()
	}
}
