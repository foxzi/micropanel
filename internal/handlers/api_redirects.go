package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/models"
	"micropanel/internal/repository"
	"micropanel/internal/services"
)

type createRedirectRequest struct {
	SourcePath    string `json:"source_path" binding:"required"`
	TargetURL     string `json:"target_url" binding:"required"`
	Code          *int   `json:"code"` // 301 (default) or 302
	PreservePath  bool   `json:"preserve_path"`
	PreserveQuery bool   `json:"preserve_query"`
	Priority      int    `json:"priority"`
}

// accessibleSite loads the site of the :id parameter and checks that the
// token may use it; on failure it answers and returns false.
func (h *APIHandler) accessibleSite(c *gin.Context) (*models.Site, bool) {
	if _, ok := requireTokenUserID(c); !ok {
		return nil, false
	}
	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid site ID"})
		return nil, false
	}
	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Error: "site not found"})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to load site"})
		return nil, false
	}
	if !h.canAccessSite(c, site) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
		return nil, false
	}
	return site, true
}

func apiTokenName(c *gin.Context) string {
	if token := middleware.GetAPIToken(c); token != nil {
		return token.Name
	}
	return ""
}

// CreateRedirect adds a redirect to a site.
// POST /api/v1/sites/:id/redirects
//
// Idempotent: repeating a redirect that exists on the site with the same
// source_path, target_url, code and preserve flags answers 200 with it; the
// same source_path with other settings answers 409.
func (h *APIHandler) CreateRedirect(c *gin.Context) {
	site, ok := h.accessibleSite(c)
	if !ok {
		return
	}

	var req createRedirectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "source_path and target_url are required"})
		return
	}
	code := 301
	if req.Code != nil {
		code = *req.Code
	}

	existing, err := h.redirectService.ListBySite(site.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list redirects"})
		return
	}
	for _, r := range existing {
		if r.SourcePath != req.SourcePath {
			continue
		}
		if r.TargetURL == req.TargetURL && r.Code == code && r.PreservePath == req.PreservePath && r.PreserveQuery == req.PreserveQuery {
			c.JSON(http.StatusOK, r)
			return
		}
		c.JSON(http.StatusConflict, errorResponse{Error: "redirect for this source_path already exists"})
		return
	}

	redirect, err := h.redirectService.Create(site.ID, req.SourcePath, req.TargetURL, code, req.PreservePath, req.PreserveQuery, req.Priority)
	if err != nil {
		if redirect == nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		// The row is saved but nginx refused (and rolled back) the config:
		// drop the row so the API and the served config agree.
		slog.Error("failed to apply nginx config after adding redirect", "site_id", site.ID, "error", err)
		if delErr := h.redirectService.Delete(redirect.ID); delErr != nil {
			slog.Error("failed to remove redirect after nginx failure", "site_id", site.ID, "redirect_id", redirect.ID, "error", delErr)
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to apply nginx config"})
		return
	}

	h.auditService.LogAnonymous(services.ActionRedirectAdd, services.EntityRedirect, map[string]string{
		"source_path": req.SourcePath,
		"target_url":  req.TargetURL,
		"site_id":     strconv.FormatInt(site.ID, 10),
		"api_token":   apiTokenName(c),
	}, c.ClientIP())

	c.JSON(http.StatusCreated, redirect)
}

// ListRedirects returns the redirects of a site, highest priority first.
// GET /api/v1/sites/:id/redirects
func (h *APIHandler) ListRedirects(c *gin.Context) {
	site, ok := h.accessibleSite(c)
	if !ok {
		return
	}

	redirects, err := h.redirectService.ListBySite(site.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list redirects"})
		return
	}
	if redirects == nil {
		redirects = []*models.Redirect{}
	}
	c.JSON(http.StatusOK, redirects)
}

// DeleteRedirect removes a redirect from a site.
// DELETE /api/v1/sites/:id/redirects/:redirectId
func (h *APIHandler) DeleteRedirect(c *gin.Context) {
	site, ok := h.accessibleSite(c)
	if !ok {
		return
	}

	redirectID, err := strconv.ParseInt(c.Param("redirectId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid redirect ID"})
		return
	}
	redirect, err := h.redirectService.GetByID(redirectID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "redirect not found"})
		return
	}
	if redirect.SiteID != site.ID {
		c.JSON(http.StatusForbidden, errorResponse{Error: "redirect does not belong to this site"})
		return
	}

	if err := h.redirectService.Delete(redirectID); err != nil {
		slog.Error("failed to delete redirect", "site_id", site.ID, "redirect_id", redirectID, "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to delete redirect"})
		return
	}

	h.auditService.LogAnonymous(services.ActionRedirectDel, services.EntityRedirect, map[string]string{
		"source_path": redirect.SourcePath,
		"site_id":     strconv.FormatInt(site.ID, 10),
		"api_token":   apiTokenName(c),
	}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "redirect deleted"})
}
