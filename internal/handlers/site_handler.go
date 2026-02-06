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
}

func NewSiteHandler(siteService *services.SiteService, deployService *services.DeployService, redirectService *services.RedirectService, authZoneService *services.AuthZoneService) *SiteHandler {
	return &SiteHandler{
		siteService:     siteService,
		deployService:   deployService,
		redirectService: redirectService,
		authZoneService: authZoneService,
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

	if name == "" {
		c.String(http.StatusBadRequest, "Name is required")
		return
	}

	_, err := h.siteService.Create(name, user.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating site")
		return
	}

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

	site.Name = c.PostForm("name")
	site.IsEnabled = c.PostForm("is_enabled") == "on"

	if err := h.siteService.Update(site); err != nil {
		c.String(http.StatusInternalServerError, "Error updating site")
		return
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

	if err := h.siteService.Delete(id); err != nil {
		c.String(http.StatusInternalServerError, "Error deleting site")
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/")
}
