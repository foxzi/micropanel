package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
)

type RedirectHandler struct {
	redirectService *services.RedirectService
	siteService     *services.SiteService
}

func NewRedirectHandler(redirectService *services.RedirectService, siteService *services.SiteService) *RedirectHandler {
	return &RedirectHandler{
		redirectService: redirectService,
		siteService:     siteService,
	}
}

func (h *RedirectHandler) Create(c *gin.Context) {
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

	sourcePath := c.PostForm("source_path")
	targetURL := c.PostForm("target_url")
	codeStr := c.PostForm("code")
	preservePath := c.PostForm("preserve_path") == "on"
	preserveQuery := c.PostForm("preserve_query") == "on"
	priorityStr := c.PostForm("priority")

	code := 301
	if codeStr != "" {
		if parsed, err := strconv.Atoi(codeStr); err == nil {
			code = parsed
		}
	}

	priority := 0
	if priorityStr != "" {
		if parsed, err := strconv.Atoi(priorityStr); err == nil {
			priority = parsed
		}
	}

	_, err = h.redirectService.Create(siteID, sourcePath, targetURL, code, preservePath, preserveQuery, priority)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}

func (h *RedirectHandler) Update(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	redirectID, err := strconv.ParseInt(c.Param("redirectId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid redirect ID")
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

	redirect, err := h.redirectService.GetByID(redirectID)
	if err != nil {
		c.String(http.StatusNotFound, "Redirect not found")
		return
	}

	if redirect.SiteID != siteID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	redirect.SourcePath = c.PostForm("source_path")
	redirect.TargetURL = c.PostForm("target_url")
	redirect.PreservePath = c.PostForm("preserve_path") == "on"
	redirect.PreserveQuery = c.PostForm("preserve_query") == "on"
	redirect.IsEnabled = c.PostForm("is_enabled") == "on"

	if codeStr := c.PostForm("code"); codeStr != "" {
		if parsed, err := strconv.Atoi(codeStr); err == nil {
			redirect.Code = parsed
		}
	}

	if priorityStr := c.PostForm("priority"); priorityStr != "" {
		if parsed, err := strconv.Atoi(priorityStr); err == nil {
			redirect.Priority = parsed
		}
	}

	if err := h.redirectService.Update(redirect); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}

func (h *RedirectHandler) Delete(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	redirectID, err := strconv.ParseInt(c.Param("redirectId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid redirect ID")
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

	redirect, err := h.redirectService.GetByID(redirectID)
	if err != nil {
		c.String(http.StatusNotFound, "Redirect not found")
		return
	}

	if redirect.SiteID != siteID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	if err := h.redirectService.Delete(redirectID); err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete redirect")
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}

func (h *RedirectHandler) Toggle(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	redirectID, err := strconv.ParseInt(c.Param("redirectId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid redirect ID")
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

	redirect, err := h.redirectService.GetByID(redirectID)
	if err != nil {
		c.String(http.StatusNotFound, "Redirect not found")
		return
	}

	if redirect.SiteID != siteID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	if err := h.redirectService.Toggle(redirectID); err != nil {
		c.String(http.StatusInternalServerError, "Failed to toggle redirect")
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}
