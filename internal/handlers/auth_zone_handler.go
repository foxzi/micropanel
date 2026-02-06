package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
)

type AuthZoneHandler struct {
	authZoneService *services.AuthZoneService
	siteService     *services.SiteService
}

func NewAuthZoneHandler(authZoneService *services.AuthZoneService, siteService *services.SiteService) *AuthZoneHandler {
	return &AuthZoneHandler{
		authZoneService: authZoneService,
		siteService:     siteService,
	}
}

func (h *AuthZoneHandler) Create(c *gin.Context) {
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

	pathPrefix := c.PostForm("path_prefix")
	realm := c.PostForm("realm")
	if realm == "" {
		realm = "Restricted"
	}

	_, err = h.authZoneService.Create(siteID, pathPrefix, realm)
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

func (h *AuthZoneHandler) Update(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	zoneID, err := strconv.ParseInt(c.Param("zoneId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid zone ID")
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

	zone, err := h.authZoneService.GetByID(zoneID)
	if err != nil {
		c.String(http.StatusNotFound, "Auth zone not found")
		return
	}

	if zone.SiteID != siteID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	zone.PathPrefix = c.PostForm("path_prefix")
	zone.Realm = c.PostForm("realm")
	zone.IsEnabled = c.PostForm("is_enabled") == "on"

	if err := h.authZoneService.Update(zone); err != nil {
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

func (h *AuthZoneHandler) Delete(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	zoneID, err := strconv.ParseInt(c.Param("zoneId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid zone ID")
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

	zone, err := h.authZoneService.GetByID(zoneID)
	if err != nil {
		c.String(http.StatusNotFound, "Auth zone not found")
		return
	}

	if zone.SiteID != siteID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	if err := h.authZoneService.Delete(zoneID); err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete auth zone")
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}

func (h *AuthZoneHandler) Toggle(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	zoneID, err := strconv.ParseInt(c.Param("zoneId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid zone ID")
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

	zone, err := h.authZoneService.GetByID(zoneID)
	if err != nil {
		c.String(http.StatusNotFound, "Auth zone not found")
		return
	}

	if zone.SiteID != siteID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	if err := h.authZoneService.Toggle(zoneID); err != nil {
		c.String(http.StatusInternalServerError, "Failed to toggle auth zone")
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}

// User management

func (h *AuthZoneHandler) CreateUser(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	zoneID, err := strconv.ParseInt(c.Param("zoneId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid zone ID")
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

	zone, err := h.authZoneService.GetByID(zoneID)
	if err != nil {
		c.String(http.StatusNotFound, "Auth zone not found")
		return
	}

	if zone.SiteID != siteID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")

	_, err = h.authZoneService.CreateUser(zoneID, username, password)
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

func (h *AuthZoneHandler) DeleteUser(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	zoneID, err := strconv.ParseInt(c.Param("zoneId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid zone ID")
		return
	}

	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
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

	zone, err := h.authZoneService.GetByID(zoneID)
	if err != nil {
		c.String(http.StatusNotFound, "Auth zone not found")
		return
	}

	if zone.SiteID != siteID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	if err := h.authZoneService.DeleteUser(userID); err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete user")
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/sites/"+strconv.FormatInt(siteID, 10))
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/sites/"+strconv.FormatInt(siteID, 10))
}
