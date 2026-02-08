package handlers

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"micropanel/internal/middleware"
	"micropanel/internal/services"
)

type FileHandler struct {
	fileService  *services.FileService
	siteService  *services.SiteService
	auditService *services.AuditService
}

func NewFileHandler(fileService *services.FileService, siteService *services.SiteService, auditService *services.AuditService) *FileHandler {
	return &FileHandler{
		fileService:  fileService,
		siteService:  siteService,
		auditService: auditService,
	}
}

// List returns files at the given path
func (h *FileHandler) List(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	path := c.Query("path")
	if path == "" {
		path = "/"
	}

	files, err := h.fileService.List(siteID, path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"path":  path,
		"files": files,
	})
}

// Read returns file content
func (h *FileHandler) Read(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	content, err := h.fileService.Read(siteID, path)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrFileNotFound {
			status = http.StatusNotFound
		} else if err == services.ErrFileTooBig {
			status = http.StatusRequestEntityTooLarge
		} else if err == services.ErrIsDirectory {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	info, _ := h.fileService.GetFileInfo(siteID, path)

	c.JSON(http.StatusOK, gin.H{
		"path":     path,
		"content":  string(content),
		"info":     info,
		"is_text":  h.fileService.IsTextFile(path),
		"is_image": h.fileService.IsImageFile(path),
	})
}

// Write saves file content
func (h *FileHandler) Write(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	if err := h.fileService.Write(siteID, req.Path, []byte(req.Content)); err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrFileTooBig {
			status = http.StatusRequestEntityTooLarge
		} else if err == services.ErrFilePathTraversal {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File saved"})
}

// Create creates a new file or directory
func (h *FileHandler) Create(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Path  string `json:"path"`
		IsDir bool   `json:"is_dir"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	if req.IsDir {
		err = h.fileService.CreateDirectory(siteID, req.Path)
	} else {
		err = h.fileService.CreateFile(siteID, req.Path)
	}

	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrFileExists {
			status = http.StatusConflict
		} else if err == services.ErrFilePathTraversal {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Created"})
}

// Delete removes a file or directory
func (h *FileHandler) Delete(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	if err := h.fileService.Delete(siteID, path); err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrFileNotFound {
			status = http.StatusNotFound
		} else if err == services.ErrCannotDelete {
			status = http.StatusForbidden
		} else if err == services.ErrFilePathTraversal {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// Rename moves/renames a file or directory
func (h *FileHandler) Rename(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.OldPath == "" || req.NewPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both paths are required"})
		return
	}

	if err := h.fileService.Rename(siteID, req.OldPath, req.NewPath); err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrFileNotFound {
			status = http.StatusNotFound
		} else if err == services.ErrFileExists {
			status = http.StatusConflict
		} else if err == services.ErrFilePathTraversal {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Renamed"})
}

// Upload handles file upload
func (h *FileHandler) Upload(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	dir := c.PostForm("path")
	if dir == "" {
		dir = "/"
	}

	filename := filepath.Join(dir, header.Filename)

	if err := h.fileService.Upload(siteID, filename, file, header.Size); err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrFileTooBig {
			status = http.StatusRequestEntityTooLarge
		} else if err == services.ErrFilePathTraversal {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Uploaded", "path": filename})
}

// Download serves a file for download
func (h *FileHandler) Download(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	content, err := h.fileService.Read(siteID, path)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrFileNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	filename := filepath.Base(path)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/octet-stream", content)
}

// Preview serves a file for preview (images)
func (h *FileHandler) Preview(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	content, err := h.fileService.Read(siteID, path)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrFileNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Detect content type
	ext := filepath.Ext(path)
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	case ".svg":
		// Serve SVG as text/plain to prevent XSS via embedded scripts
		contentType = "text/plain; charset=utf-8"
	case ".ico":
		contentType = "image/x-icon"
	}

	// Add security headers for user-uploaded content
	c.Header("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, contentType, content)
}

// Info returns file information
func (h *FileHandler) Info(c *gin.Context) {
	user := middleware.GetUser(c)

	siteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site ID"})
		return
	}

	site, err := h.siteService.GetByID(siteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
		return
	}

	if !h.siteService.CanAccess(site, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	info, err := h.fileService.GetFileInfo(siteID, path)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrFileNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"info":     info,
		"is_text":  h.fileService.IsTextFile(path),
		"is_image": h.fileService.IsImageFile(path),
	})
}
