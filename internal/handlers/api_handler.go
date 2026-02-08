package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/models"
	"micropanel/internal/services"
)

type APIHandler struct {
	siteService   *services.SiteService
	deployService *services.DeployService
	nginxService  *services.NginxService
	sslService    *services.SSLService
	auditService  *services.AuditService
}

func NewAPIHandler(siteService *services.SiteService, deployService *services.DeployService, nginxService *services.NginxService, sslService *services.SSLService, auditService *services.AuditService) *APIHandler {
	return &APIHandler{
		siteService:   siteService,
		deployService: deployService,
		nginxService:  nginxService,
		sslService:    sslService,
		auditService:  auditService,
	}
}

type createSiteRequest struct {
	Name string `json:"name" binding:"required"`
	SSL  *bool  `json:"ssl"` // optional, default true
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

// getTokenUserID returns the user ID associated with the API token.
// Returns 0 if token has no user_id configured (caller must handle this).
func getTokenUserID(c *gin.Context) int64 {
	token := middleware.GetAPIToken(c)
	if token != nil && token.UserID > 0 {
		return token.UserID
	}
	return 0 // No user_id configured
}

// requireTokenUserID returns the user ID or aborts with error if not configured.
func requireTokenUserID(c *gin.Context) (int64, bool) {
	userID := getTokenUserID(c)
	if userID == 0 {
		c.JSON(http.StatusForbidden, errorResponse{Error: "API token must have user_id configured"})
		return 0, false
	}
	return userID, true
}

// CreateSite creates a new site via API.
// POST /api/v1/sites
func (h *APIHandler) CreateSite(c *gin.Context) {
	// Require token with user_id
	ownerID, ok := requireTokenUserID(c)
	if !ok {
		return
	}

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

	site, err := h.siteService.Create(req.Name, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to create site"})
		return
	}

	// Generate nginx config
	h.nginxService.WriteConfig(site.ID)

	// Issue SSL certificate if requested (default: true)
	issueSSL := req.SSL == nil || *req.SSL
	if issueSSL {
		go h.sslService.IssueCertificate(site.ID)
	}

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

// ListSites returns sites owned by the token's user.
// GET /api/v1/sites
func (h *APIHandler) ListSites(c *gin.Context) {
	userID := getTokenUserID(c)

	// Token without user_id cannot list sites
	if userID == 0 {
		c.JSON(http.StatusForbidden, errorResponse{Error: "API token must have user_id configured"})
		return
	}

	// User ID 1 (admin) can see all sites, others see only their own
	var sites []*models.Site
	var err error
	if userID == 1 {
		sites, err = h.siteService.ListAll()
	} else {
		sites, err = h.siteService.ListByOwner(userID)
	}

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

// canAccessSite checks if the API token user can access the site.
func (h *APIHandler) canAccessSite(c *gin.Context, site *models.Site) bool {
	userID := getTokenUserID(c)
	// Token without user_id has no access
	if userID == 0 {
		return false
	}
	// Admin (user_id=1) can access all sites
	if userID == 1 {
		return true
	}
	return site.OwnerID == userID
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

	// Check access
	if !h.canAccessSite(c, site) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
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

	// Check access
	if !h.canAccessSite(c, site) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
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

	// Check access
	if !h.canAccessSite(c, site) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "file is required"})
		return
	}
	defer file.Close()

	// Deploy using token's user ID
	userID := getTokenUserID(c)
	deploy, err := h.deployService.Deploy(site.ID, userID, header.Filename, file, header.Size)
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
