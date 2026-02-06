package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
	"micropanel/internal/templates/pages"
)

type AuthHandler struct {
	authService  *services.AuthService
	auditService *services.AuditService
}

func NewAuthHandler(authService *services.AuthService, auditService *services.AuditService) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		auditService: auditService,
	}
}

func (h *AuthHandler) LoginPage(c *gin.Context) {
	csrfToken := middleware.GetCSRFToken(c)
	component := pages.Login(csrfToken, "")
	component.Render(c.Request.Context(), c.Writer)
}

func (h *AuthHandler) Login(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")
	ip := c.ClientIP()

	session, err := h.authService.Login(email, password)
	if err != nil {
		// Log failed login attempt
		h.auditService.LogAnonymous(services.ActionLoginFailed, services.EntityUser, map[string]string{"email": email}, ip)

		csrfToken := middleware.GetCSRFToken(c)
		c.Writer.WriteHeader(http.StatusUnauthorized)
		component := pages.Login(csrfToken, "Invalid email or password")
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	// Log successful login
	h.auditService.LogUser(session.UserID, services.ActionLogin, services.EntityUser, &session.UserID, nil, ip)

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
	user := middleware.GetUser(c)
	ip := c.ClientIP()

	sessionID, err := c.Cookie(services.SessionCookieKey)
	if err == nil {
		h.authService.Logout(sessionID)
	}

	// Log logout
	if user != nil {
		h.auditService.LogUser(user.ID, services.ActionLogout, services.EntityUser, &user.ID, nil, ip)
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
