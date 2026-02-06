package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"micropanel/internal/middleware"
	"micropanel/internal/models"
	"micropanel/internal/repository"
	"micropanel/internal/services"
	"micropanel/internal/templates/pages"
)

type UserHandler struct {
	userRepo     *repository.UserRepository
	auditService *services.AuditService
}

func NewUserHandler(userRepo *repository.UserRepository, auditService *services.AuditService) *UserHandler {
	return &UserHandler{
		userRepo:     userRepo,
		auditService: auditService,
	}
}

// List shows all users (admin only)
func (h *UserHandler) List(c *gin.Context) {
	user := middleware.GetUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	if !user.IsAdmin() {
		c.Redirect(http.StatusFound, "/")
		return
	}

	users, err := h.userRepo.List()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error loading users")
		return
	}

	component := pages.Users(user, users, csrfToken)
	component.Render(c.Request.Context(), c.Writer)
}

// Create creates a new user (admin only)
func (h *UserHandler) Create(c *gin.Context) {
	currentUser := middleware.GetUser(c)
	ip := c.ClientIP()

	if !currentUser.IsAdmin() {
		c.String(http.StatusForbidden, "Admin access required")
		return
	}

	email := c.PostForm("email")
	password := c.PostForm("password")
	role := models.Role(c.PostForm("role"))

	if email == "" || password == "" {
		c.String(http.StatusBadRequest, "Email and password are required")
		return
	}

	if role != models.RoleAdmin && role != models.RoleUser {
		role = models.RoleUser
	}

	// Check if user already exists
	if _, err := h.userRepo.GetByEmail(email); err == nil {
		c.String(http.StatusConflict, "User with this email already exists")
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating user")
		return
	}

	newUser := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		IsActive:     true,
	}

	if err := h.userRepo.Create(newUser); err != nil {
		c.String(http.StatusInternalServerError, "Error creating user")
		return
	}

	h.auditService.LogUser(currentUser.ID, services.ActionUserCreate, services.EntityUser, &newUser.ID, map[string]string{
		"email": email,
		"role":  string(role),
	}, ip)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/users")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

// Update updates a user (admin only)
func (h *UserHandler) Update(c *gin.Context) {
	currentUser := middleware.GetUser(c)
	ip := c.ClientIP()

	if !currentUser.IsAdmin() {
		c.String(http.StatusForbidden, "Admin access required")
		return
	}

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	user.Email = c.PostForm("email")
	if role := models.Role(c.PostForm("role")); role == models.RoleAdmin || role == models.RoleUser {
		user.Role = role
	}

	// Update password if provided
	if password := c.PostForm("password"); password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error updating user")
			return
		}
		user.PasswordHash = string(hash)
	}

	if err := h.userRepo.Update(user); err != nil {
		c.String(http.StatusInternalServerError, "Error updating user")
		return
	}

	h.auditService.LogUser(currentUser.ID, services.ActionUserUpdate, services.EntityUser, &userID, nil, ip)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/users")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

// ToggleActive toggles user active status (admin only)
func (h *UserHandler) ToggleActive(c *gin.Context) {
	currentUser := middleware.GetUser(c)
	ip := c.ClientIP()

	if !currentUser.IsAdmin() {
		c.String(http.StatusForbidden, "Admin access required")
		return
	}

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Prevent self-deactivation
	if userID == currentUser.ID {
		c.String(http.StatusBadRequest, "Cannot deactivate yourself")
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	user.IsActive = !user.IsActive

	if err := h.userRepo.Update(user); err != nil {
		c.String(http.StatusInternalServerError, "Error updating user")
		return
	}

	action := services.ActionUserBlock
	if user.IsActive {
		action = services.ActionUserUnblock
	}
	h.auditService.LogUser(currentUser.ID, action, services.EntityUser, &userID, nil, ip)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/users")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

// Delete deletes a user (admin only)
func (h *UserHandler) Delete(c *gin.Context) {
	currentUser := middleware.GetUser(c)
	ip := c.ClientIP()

	if !currentUser.IsAdmin() {
		c.String(http.StatusForbidden, "Admin access required")
		return
	}

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Prevent self-deletion
	if userID == currentUser.ID {
		c.String(http.StatusBadRequest, "Cannot delete yourself")
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	if err := h.userRepo.Delete(userID); err != nil {
		c.String(http.StatusInternalServerError, "Error deleting user")
		return
	}

	h.auditService.LogUser(currentUser.ID, services.ActionUserDelete, services.EntityUser, &userID, map[string]string{
		"email": user.Email,
	}, ip)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/users")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

// Profile shows current user profile
func (h *UserHandler) Profile(c *gin.Context) {
	user := middleware.GetUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	component := pages.Profile(user, csrfToken, "")
	component.Render(c.Request.Context(), c.Writer)
}

// ChangePassword changes current user's password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	user := middleware.GetUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	currentPassword := c.PostForm("current_password")
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		component := pages.Profile(user, csrfToken, "Current password is incorrect")
		c.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	// Validate new password
	if len(newPassword) < 6 {
		component := pages.Profile(user, csrfToken, "New password must be at least 6 characters")
		c.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	if newPassword != confirmPassword {
		component := pages.Profile(user, csrfToken, "Passwords do not match")
		c.Writer.WriteHeader(http.StatusBadRequest)
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		component := pages.Profile(user, csrfToken, "Error changing password")
		c.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	user.PasswordHash = string(hash)
	if err := h.userRepo.Update(user); err != nil {
		component := pages.Profile(user, csrfToken, "Error changing password")
		c.Writer.WriteHeader(http.StatusInternalServerError)
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/profile?success=1")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/profile?success=1")
}
