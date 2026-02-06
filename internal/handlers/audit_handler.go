package handlers

import (
	"micropanel/internal/middleware"
	"micropanel/internal/repository"
	"micropanel/internal/services"
	"micropanel/internal/templates/pages"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	auditService *services.AuditService
	userRepo     *repository.UserRepository
}

func NewAuditHandler(auditService *services.AuditService, userRepo *repository.UserRepository) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
		userRepo:     userRepo,
	}
}

func (h *AuditHandler) List(c *gin.Context) {
	user := middleware.GetUser(c)

	// Admin only
	if !user.IsAdmin() {
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 50

	logs, total, err := h.auditService.List(page, perPage)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "", gin.H{"error": err.Error()})
		return
	}

	// Get user emails for display
	userEmails := make(map[int64]string)
	for _, log := range logs {
		if log.UserID != nil {
			if _, ok := userEmails[*log.UserID]; !ok {
				if u, err := h.userRepo.GetByID(*log.UserID); err == nil {
					userEmails[*log.UserID] = u.Email
				}
			}
		}
	}

	totalPages := (total + perPage - 1) / perPage

	component := pages.AuditLog(user, logs, userEmails, page, totalPages, total)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *AuditHandler) ListAPI(c *gin.Context) {
	user := middleware.GetUser(c)

	if !user.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))

	logs, total, err := h.auditService.List(page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
	})
}
