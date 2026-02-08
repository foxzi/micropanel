package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
)

type DeployHandler struct {
	deployService *services.DeployService
	siteService   *services.SiteService
	auditService  *services.AuditService
}

func NewDeployHandler(deployService *services.DeployService, siteService *services.SiteService, auditService *services.AuditService) *DeployHandler {
	return &DeployHandler{
		deployService: deployService,
		siteService:   siteService,
		auditService:  auditService,
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
	filename := strings.ToLower(header.Filename)
	validExt := strings.HasSuffix(filename, ".zip") ||
		strings.HasSuffix(filename, ".tgz") ||
		strings.HasSuffix(filename, ".tar.gz")
	if !validExt {
		c.String(http.StatusBadRequest, "Only ZIP and TGZ files are allowed")
		return
	}

	// Deploy
	deploy, err := h.deployService.Deploy(siteID, user.ID, header.Filename, file, header.Size)
	if err != nil {
		// Return user-friendly error messages without exposing internals
		errMsg := "Deploy failed"
		switch err {
		case services.ErrArchiveTooLarge:
			errMsg = "Archive is too large"
		case services.ErrTooManyFiles:
			errMsg = "Too many files in archive"
		case services.ErrUnsupportedArchive:
			errMsg = "Unsupported archive format"
		case services.ErrPathTraversal:
			errMsg = "Invalid file paths in archive"
		case services.ErrSymlinkDetected:
			errMsg = "Symlinks are not allowed in archive"
		}
		c.String(http.StatusInternalServerError, errMsg)
		return
	}

	// Log deploy
	h.auditService.LogUser(user.ID, services.ActionDeploy, services.EntityDeploy, &deploy.ID, map[string]interface{}{
		"filename": header.Filename,
		"site_id":  siteID,
		"size":     header.Size,
	}, c.ClientIP())

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

	// Log rollback
	h.auditService.LogUser(user.ID, services.ActionRollback, services.EntitySite, &siteID, nil, c.ClientIP())

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}
