package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
	"micropanel/internal/templates/pages"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) LoginPage(c *gin.Context) {
	csrfToken := middleware.GetCSRFToken(c)
	component := pages.Login(csrfToken, "")
	component.Render(c.Request.Context(), c.Writer)
}

func (h *AuthHandler) Login(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	session, err := h.authService.Login(email, password)
	if err != nil {
		csrfToken := middleware.GetCSRFToken(c)
		c.Writer.WriteHeader(http.StatusUnauthorized)
		component := pages.Login(csrfToken, "Invalid email or password")
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	c.SetCookie(
		services.SessionCookieKey,
		session.ID,
		int(services.SessionDuration.Seconds()),
		"/",
		"",
		false,
		true,
	)

	// HTMX redirect
	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, err := c.Cookie(services.SessionCookieKey)
	if err == nil {
		h.authService.Logout(sessionID)
	}

	c.SetCookie(services.SessionCookieKey, "", -1, "/", "", false, true)

	// HTMX redirect
	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/login")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/login")
}
