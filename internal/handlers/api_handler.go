package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
)

type APIHandler struct {
	siteService   *services.SiteService
	deployService *services.DeployService
	nginxService  *services.NginxService
	auditService  *services.AuditService
}

func NewAPIHandler(siteService *services.SiteService, deployService *services.DeployService, nginxService *services.NginxService, auditService *services.AuditService) *APIHandler {
	return &APIHandler{
		siteService:   siteService,
		deployService: deployService,
		nginxService:  nginxService,
		auditService:  auditService,
	}
}

type createSiteRequest struct {
	Name string `json:"name" binding:"required"`
}

type siteResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	IsEnabled bool   `json:"is_enabled"`
	SSLEnabled bool  `json:"ssl_enabled"`
}

type deployResponse struct {
	DeployID int64  `json:"deploy_id"`
	Status   string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// CreateSite creates a new site via API.
// POST /api/v1/sites
func (h *APIHandler) CreateSite(c *gin.Context) {
	var req createSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "name is required"})
		return
	}

	// Check if site with this name already exists
	if existing, _ := h.siteService.GetByName(req.Name); existing != nil {
		c.JSON(http.StatusConflict, errorResponse{Error: "site with this name already exists"})
		return
	}

	// Use owner_id = 1 (admin) for API-created sites
	site, err := h.siteService.Create(req.Name, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to create site"})
		return
	}

	// Generate nginx config
	h.nginxService.WriteConfig(site.ID)

	// Log via audit
	token := middleware.GetAPIToken(c)
	tokenName := ""
	if token != nil {
		tokenName = token.Name
	}
	h.auditService.LogAnonymous(services.ActionSiteCreate, services.EntitySite, map[string]string{
		"name":      req.Name,
		"api_token": tokenName,
	}, c.ClientIP())

	c.JSON(http.StatusCreated, siteResponse{
		ID:        site.ID,
		Name:      site.Name,
		IsEnabled: site.IsEnabled,
		SSLEnabled: site.SSLEnabled,
	})
}

// ListSites returns all sites.
// GET /api/v1/sites
func (h *APIHandler) ListSites(c *gin.Context) {
	sites, err := h.siteService.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list sites"})
		return
	}

	var response []siteResponse
	for _, site := range sites {
		response = append(response, siteResponse{
			ID:        site.ID,
			Name:      site.Name,
			IsEnabled: site.IsEnabled,
			SSLEnabled: site.SSLEnabled,
		})
	}

	if response == nil {
		response = []siteResponse{}
	}

	c.JSON(http.StatusOK, response)
}

// GetSite returns a single site by ID.
// GET /api/v1/sites/:id
func (h *APIHandler) GetSite(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "site not found"})
		return
	}

	c.JSON(http.StatusOK, siteResponse{
		ID:        site.ID,
		Name:      site.Name,
		IsEnabled: site.IsEnabled,
		SSLEnabled: site.SSLEnabled,
	})
}

// DeleteSite deletes a site by ID.
// DELETE /api/v1/sites/:id
func (h *APIHandler) DeleteSite(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "site not found"})
		return
	}

	siteName := site.Name

	if err := h.siteService.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to delete site"})
		return
	}

	// Log via audit
	token := middleware.GetAPIToken(c)
	tokenName := ""
	if token != nil {
		tokenName = token.Name
	}
	h.auditService.LogAnonymous(services.ActionSiteDelete, services.EntitySite, map[string]string{
		"name":      siteName,
		"api_token": tokenName,
	}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "site deleted"})
}

// Deploy uploads and deploys an archive to a site.
// POST /api/v1/sites/:id/deploy
func (h *APIHandler) Deploy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "site not found"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "file is required"})
		return
	}
	defer file.Close()

	// Deploy using admin user ID (1) for API deploys
	deploy, err := h.deployService.Deploy(site.ID, 1, header.Filename, file, header.Size)
	if err != nil {
		status := http.StatusInternalServerError
		errMsg := "deploy failed"

		switch err {
		case services.ErrArchiveTooLarge:
			status = http.StatusRequestEntityTooLarge
			errMsg = "archive too large"
		case services.ErrTooManyFiles:
			status = http.StatusBadRequest
			errMsg = "too many files in archive"
		case services.ErrUnsupportedArchive:
			status = http.StatusBadRequest
			errMsg = "unsupported archive format (use .zip or .tar.gz)"
		case services.ErrPathTraversal, services.ErrSymlinkDetected:
			status = http.StatusBadRequest
			errMsg = "invalid archive content"
		}

		c.JSON(status, errorResponse{Error: errMsg})
		return
	}

	// Log via audit
	token := middleware.GetAPIToken(c)
	tokenName := ""
	if token != nil {
		tokenName = token.Name
	}
	h.auditService.LogAnonymous(services.ActionDeploy, services.EntitySite, map[string]string{
		"site_name": site.Name,
		"filename":  header.Filename,
		"api_token": tokenName,
	}, c.ClientIP())

	c.JSON(http.StatusOK, deployResponse{
		DeployID: deploy.ID,
		Status:   string(deploy.Status),
	})
}
