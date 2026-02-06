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
}

func NewDomainHandler(domainRepo *repository.DomainRepository, siteService *services.SiteService, nginxService *services.NginxService) *DomainHandler {
	return &DomainHandler{
		domainRepo:   domainRepo,
		siteService:  siteService,
		nginxService: nginxService,
	}
}

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

	// Check if domain already exists
	if _, err := h.domainRepo.GetByHostname(hostname); err == nil {
		c.String(http.StatusConflict, "Domain already exists")
		return
	}

	// Check if this is the first domain (make it primary)
	existingDomains, _ := h.domainRepo.ListBySite(siteID)
	isPrimary := len(existingDomains) == 0

	domain := &models.Domain{
		SiteID:    siteID,
		Hostname:  hostname,
		IsPrimary: isPrimary,
	}

	if err := h.domainRepo.Create(domain); err != nil {
		c.String(http.StatusInternalServerError, "Error creating domain")
		return
	}

	// Regenerate nginx config
	if err := h.nginxService.ApplyConfig(siteID); err != nil {
		// Log error but don't fail - domain is created
		c.Header("X-Nginx-Error", err.Error())
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}

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

	if err := h.domainRepo.Delete(domainID); err != nil {
		c.String(http.StatusInternalServerError, "Error deleting domain")
		return
	}

	// Check if there are remaining domains
	remainingDomains, _ := h.domainRepo.ListBySite(siteID)
	if len(remainingDomains) > 0 {
		// Regenerate nginx config
		if err := h.nginxService.ApplyConfig(siteID); err != nil {
			c.Header("X-Nginx-Error", err.Error())
		}
	} else {
		// Remove nginx config if no domains left
		h.nginxService.RemoveConfig(siteID)
		h.nginxService.Reload()
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}

func (h *DomainHandler) SetPrimary(c *gin.Context) {
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

	if err := h.domainRepo.SetPrimary(siteID, domainID); err != nil {
		c.String(http.StatusInternalServerError, "Error setting primary domain")
		return
	}

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
