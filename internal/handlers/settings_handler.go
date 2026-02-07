package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
	"micropanel/internal/templates/pages"
)

type SettingsHandler struct {
	settingsService *services.SettingsService
	auditService    *services.AuditService
}

func NewSettingsHandler(settingsService *services.SettingsService, auditService *services.AuditService) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
		auditService:    auditService,
	}
}

func (h *SettingsHandler) Page(c *gin.Context) {
	user := middleware.GetUser(c)

	if !user.IsAdmin() {
		c.Redirect(http.StatusFound, "/")
		return
	}

	info := h.settingsService.GetServerInfo()
	csrfToken := middleware.GetCSRFToken(c)

	component := pages.Settings(user, info, csrfToken, "")
	component.Render(c.Request.Context(), c.Writer)
}

func (h *SettingsHandler) Update(c *gin.Context) {
	user := middleware.GetUser(c)

	if !user.IsAdmin() {
		c.Redirect(http.StatusFound, "/")
		return
	}

	serverName := c.PostForm("server_name")
	serverNotes := c.PostForm("server_notes")

	if err := h.settingsService.UpdateServerName(serverName); err != nil {
		info := h.settingsService.GetServerInfo()
		csrfToken := middleware.GetCSRFToken(c)
		component := pages.Settings(user, info, csrfToken, "Failed to save settings")
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	if err := h.settingsService.UpdateServerNotes(serverNotes); err != nil {
		info := h.settingsService.GetServerInfo()
		csrfToken := middleware.GetCSRFToken(c)
		component := pages.Settings(user, info, csrfToken, "Failed to save settings")
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	h.auditService.LogUser(user.ID, "settings_update", "settings", nil, nil, c.ClientIP())

	info := h.settingsService.GetServerInfo()
	csrfToken := middleware.GetCSRFToken(c)
	component := pages.Settings(user, info, csrfToken, "Settings saved successfully")
	component.Render(c.Request.Context(), c.Writer)
}
