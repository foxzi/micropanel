package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/models"
	"micropanel/internal/repository"
	"micropanel/internal/services"
)

type DomainHandler struct {
	domainRepo   *repository.DomainRepository
	siteService  *services.SiteService
	nginxService *services.NginxService
	auditService *services.AuditService
}

func NewDomainHandler(domainRepo *repository.DomainRepository, siteService *services.SiteService, nginxService *services.NginxService, auditService *services.AuditService) *DomainHandler {
	return &DomainHandler{
		domainRepo:   domainRepo,
		siteService:  siteService,
		nginxService: nginxService,
		auditService: auditService,
	}
}

// Create adds an alias domain to a site
func (h *DomainHandler) Create(c *gin.Context) {
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

	hostname := c.PostForm("hostname")
	if hostname == "" {
		c.String(http.StatusBadRequest, "Hostname is required")
		return
	}

	// Check if hostname is same as site name or www alias
	if hostname == site.Name || hostname == "www."+site.Name {
		c.String(http.StatusBadRequest, "Cannot add primary domain or www alias as alias")
		return
	}

	// Check if domain already exists
	if _, err := h.domainRepo.GetByHostname(hostname); err == nil {
		c.String(http.StatusConflict, "Domain already exists")
		return
	}

	domain := &models.Domain{
		SiteID:   siteID,
		Hostname: hostname,
	}

	if err := h.domainRepo.Create(domain); err != nil {
		c.String(http.StatusInternalServerError, "Error creating domain alias")
		return
	}

	// Log domain creation
	h.auditService.LogUser(user.ID, services.ActionDomainAdd, services.EntityDomain, &domain.ID, map[string]interface{}{
		"hostname": hostname,
		"site_id":  siteID,
	}, c.ClientIP())

	// Regenerate nginx config
	if err := h.nginxService.ApplyConfig(siteID); err != nil {
		c.Header("X-Nginx-Error", err.Error())
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}

// Delete removes an alias domain from a site
func (h *DomainHandler) Delete(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	domainID, err := strconv.ParseInt(c.Param("domainId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid domain ID")
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

	domain, err := h.domainRepo.GetByID(domainID)
	if err != nil {
		c.String(http.StatusNotFound, "Domain not found")
		return
	}

	if domain.SiteID != siteID {
		c.String(http.StatusForbidden, "Domain does not belong to this site")
		return
	}

	hostname := domain.Hostname

	if err := h.domainRepo.Delete(domainID); err != nil {
		c.String(http.StatusInternalServerError, "Error deleting domain alias")
		return
	}

	// Log domain deletion
	h.auditService.LogUser(user.ID, services.ActionDomainDelete, services.EntityDomain, &domainID, map[string]interface{}{
		"hostname": hostname,
		"site_id":  siteID,
	}, c.ClientIP())

	// Regenerate nginx config
	if err := h.nginxService.ApplyConfig(siteID); err != nil {
		c.Header("X-Nginx-Error", err.Error())
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}
