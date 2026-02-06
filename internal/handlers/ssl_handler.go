package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
)

type SSLHandler struct {
	sslService   *services.SSLService
	siteService  *services.SiteService
	auditService *services.AuditService
}

func NewSSLHandler(sslService *services.SSLService, siteService *services.SiteService, auditService *services.AuditService) *SSLHandler {
	return &SSLHandler{
		sslService:   sslService,
		siteService:  siteService,
		auditService: auditService,
	}
}

func (h *SSLHandler) Issue(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.String(http.StatusNotFound, "Site not found")
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	if err := h.sslService.IssueCertificate(siteID); err != nil {
		c.String(http.StatusInternalServerError, "SSL issue failed: %s", err.Error())
		return
	}

	// Log SSL issue
	h.auditService.LogUser(user.ID, services.ActionSSLIssue, services.EntitySite, &siteID, nil, c.ClientIP())

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}

func (h *SSLHandler) Renew(c *gin.Context) {
	user := middleware.GetUser(c)

	// Only admins can trigger manual renewal
	if !user.IsAdmin() {
		c.String(http.StatusForbidden, "Admin access required")
		return
	}

	if err := h.sslService.RenewCertificates(); err != nil {
		c.String(http.StatusInternalServerError, "SSL renewal failed: %s", err.Error())
		return
	}

	// Log SSL renewal
	h.auditService.LogUser(user.ID, services.ActionSSLRenew, services.EntitySite, nil, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Certificates renewed"})
}
