package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
)

type DeployHandler struct {
	deployService *services.DeployService
	siteService   *services.SiteService
}

func NewDeployHandler(deployService *services.DeployService, siteService *services.SiteService) *DeployHandler {
	return &DeployHandler{
		deployService: deployService,
		siteService:   siteService,
	}
}

func (h *DeployHandler) Upload(c *gin.Context) {
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

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	// Validate file extension
	if len(header.Filename) < 4 || header.Filename[len(header.Filename)-4:] != ".zip" {
		c.String(http.StatusBadRequest, "Only ZIP files are allowed")
		return
	}

	// Deploy
	deploy, err := h.deployService.Deploy(siteID, user.ID, header.Filename, file, header.Size)
	if err != nil {
		if deploy != nil {
			// Deploy record was created but failed
			c.String(http.StatusInternalServerError, "Deploy failed: %s", err.Error())
		} else {
			c.String(http.StatusInternalServerError, "Deploy failed: %s", err.Error())
		}
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}

func (h *DeployHandler) Rollback(c *gin.Context) {
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

	if err := h.deployService.Rollback(siteID); err != nil {
		c.String(http.StatusInternalServerError, "Rollback failed: %s", err.Error())
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}
