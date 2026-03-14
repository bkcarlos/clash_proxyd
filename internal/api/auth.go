package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LoginRequest represents login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	token, expiresAt, err := h.authManager.Login(req.Username, req.Password)
	if err != nil {
		h.auditLog(c, "login", "auth", "Failed login attempt for user: "+req.Username)
		h.respondError(c, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	h.auditLog(c, "login", "auth", "Successful login for user: "+req.Username)
	h.respondJSON(c, http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	})
}

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	// Get user from context
	user := h.getUser(c)

	// Logout (client-side token deletion)
	if err := h.authManager.Logout(""); err != nil {
		h.respondError(c, http.StatusInternalServerError, "Logout failed")
		return
	}

	h.auditLog(c, "logout", "auth", "User logged out: "+user)
	h.respondSuccess(c, "Logged out successfully", nil)
}

// RefreshToken handles token refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		h.respondError(c, http.StatusBadRequest, "Token required")
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	newToken, expiresAt, err := h.authManager.RefreshToken(token)
	if err != nil {
		h.respondError(c, http.StatusUnauthorized, "Token refresh failed: "+err.Error())
		return
	}

	h.respondJSON(c, http.StatusOK, LoginResponse{
		Token:     newToken,
		ExpiresAt: expiresAt,
	})
}

// GetProfile returns current user profile
func (h *Handler) GetProfile(c *gin.Context) {
	user := h.getUser(c)
	h.respondJSON(c, http.StatusOK, gin.H{
		"username": user,
		"role":     "admin",
	})
}

// UpdatePassword updates user password
func (h *Handler) UpdatePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	user := h.getUser(c)

	// Verify old password (simplified for MVP)
	// In production, verify old password hash
	if err := h.authManager.SetCredentials(user, req.NewPassword); err != nil {
		h.respondError(c, http.StatusInternalServerError, "Password update failed")
		return
	}

	h.auditLog(c, "password_update", "auth", "Password updated for user: "+user)
	h.respondSuccess(c, "Password updated successfully", nil)
}
