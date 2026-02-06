package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
	"micropanel/internal/templates/pages"
)

type SiteHandler struct {
	siteService     *services.SiteService
	deployService   *services.DeployService
	redirectService *services.RedirectService
	authZoneService *services.AuthZoneService
	auditService    *services.AuditService
}

func NewSiteHandler(siteService *services.SiteService, deployService *services.DeployService, redirectService *services.RedirectService, authZoneService *services.AuthZoneService, auditService *services.AuditService) *SiteHandler {
	return &SiteHandler{
		siteService:     siteService,
		deployService:   deployService,
		redirectService: redirectService,
		authZoneService: authZoneService,
		auditService:    auditService,
	}
}

func (h *SiteHandler) Dashboard(c *gin.Context) {
	user := middleware.GetUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	sites, err := h.siteService.List(user)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error loading sites")
		return
	}

	component := pages.Dashboard(user, sites, csrfToken)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *SiteHandler) Create(c *gin.Context) {
	user := middleware.GetUser(c)
	name := c.PostForm("name")
	ip := c.ClientIP()

	if name == "" {
		c.String(http.StatusBadRequest, "Name is required")
		return
	}

	site, err := h.siteService.Create(name, user.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating site")
		return
	}

	// Log site creation
	h.auditService.LogUser(user.ID, services.ActionSiteCreate, services.EntitySite, &site.ID, map[string]string{"name": name}, ip)

	// HTMX refresh
	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/")
}

func (h *SiteHandler) View(c *gin.Context) {
	user := middleware.GetUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.siteService.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Site not found")
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	// Get deploy history
	deploys, _ := h.deployService.ListDeploys(id, 10)
	canRollback := h.deployService.HasPreviousVersion(id)

	// Get redirects
	redirects, _ := h.redirectService.ListBySite(id)

	// Get auth zones with users
	authZones, _ := h.authZoneService.ListBySiteWithUsers(id)

	component := pages.SiteView(user, site, deploys, redirects, authZones, canRollback, csrfToken)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *SiteHandler) Update(c *gin.Context) {
	user := middleware.GetUser(c)
	ip := c.ClientIP()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.siteService.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Site not found")
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	oldName := site.Name
	oldEnabled := site.IsEnabled

	site.Name = c.PostForm("name")
	site.IsEnabled = c.PostForm("is_enabled") == "on"

	if err := h.siteService.Update(site); err != nil {
		c.String(http.StatusInternalServerError, "Error updating site")
		return
	}

	// Log site update
	if oldName != site.Name || oldEnabled != site.IsEnabled {
		h.auditService.LogUser(user.ID, services.ActionSiteUpdate, services.EntitySite, &site.ID, map[string]interface{}{
			"name":       site.Name,
			"is_enabled": site.IsEnabled,
		}, ip)
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(id, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(id, 10))
}

func (h *SiteHandler) Files(c *gin.Context) {
	user := middleware.GetUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.siteService.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Site not found")
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	component := pages.Files(user, site, csrfToken)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *SiteHandler) Delete(c *gin.Context) {
	user := middleware.GetUser(c)
	ip := c.ClientIP()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.siteService.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Site not found")
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	siteName := site.Name

	if err := h.siteService.Delete(id); err != nil {
		c.String(http.StatusInternalServerError, "Error deleting site")
		return
	}

	// Log site deletion
	h.auditService.LogUser(user.ID, services.ActionSiteDelete, services.EntitySite, &id, map[string]string{"name": siteName}, ip)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/")
}
