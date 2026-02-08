package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
	"micropanel/internal/templates/pages"
)

const (
	ActionAPITokenCreate = "api_token_create"
	ActionAPITokenDelete = "api_token_delete"
	EntityAPIToken       = "api_token"
)

type APITokenHandler struct {
	tokenService *services.APITokenService
	auditService *services.AuditService
}

func NewAPITokenHandler(tokenService *services.APITokenService, auditService *services.AuditService) *APITokenHandler {
	return &APITokenHandler{
		tokenService: tokenService,
		auditService: auditService,
	}
}

// List shows the API tokens page
func (h *APITokenHandler) List(c *gin.Context) {
	user := middleware.GetUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	tokens, err := h.tokenService.ListUserTokens(user.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error loading tokens")
		return
	}

	component := pages.APITokens(user, tokens, csrfToken, "")
	component.Render(c.Request.Context(), c.Writer)
}

// Create creates a new API token
func (h *APITokenHandler) Create(c *gin.Context) {
	user := middleware.GetUser(c)
	csrfToken := middleware.GetCSRFToken(c)
	ip := c.ClientIP()

	name := c.PostForm("name")
	if name == "" {
		tokens, _ := h.tokenService.ListUserTokens(user.ID)
		component := pages.APITokens(user, tokens, csrfToken, "Token name is required")
		c.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	token, err := h.tokenService.CreateToken(user.ID, name)
	if err != nil {
		tokens, _ := h.tokenService.ListUserTokens(user.ID)
		component := pages.APITokens(user, tokens, csrfToken, "Error creating token")
		c.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	// Log the action
	h.auditService.LogUser(user.ID, ActionAPITokenCreate, EntityAPIToken, &token.ID, map[string]string{
		"name": name,
	}, ip)

	// Show page with the new token (shown only once)
	tokens, _ := h.tokenService.ListUserTokens(user.ID)
	component := pages.APITokensWithNewToken(user, tokens, csrfToken, token.Token)
	component.Render(c.Request.Context(), c.Writer)
}

// Delete deletes an API token
func (h *APITokenHandler) Delete(c *gin.Context) {
	user := middleware.GetUser(c)
	ip := c.ClientIP()

	tokenID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid token ID")
		return
	}

	if err := h.tokenService.DeleteToken(tokenID, user.ID); err != nil {
		switch err {
		case services.ErrTokenNotFound:
			c.String(http.StatusNotFound, "Token not found")
		case services.ErrTokenNotOwned:
			c.String(http.StatusForbidden, "Access denied")
		default:
			c.String(http.StatusInternalServerError, "Error deleting token")
		}
		return
	}

	// Log the action
	h.auditService.LogUser(user.ID, ActionAPITokenDelete, EntityAPIToken, &tokenID, nil, ip)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/api-tokens")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/api-tokens")
}
