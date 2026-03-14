package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/clash-proxyd/proxyd/internal/policy"
)

// GenerateGroups generates proxy groups
func (h *Handler) GenerateGroups(c *gin.Context) {
	var req struct {
		ProxyNames []string `json:"proxy_names" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	generator := policy.NewGenerator()
	groups := generator.GenerateGroups(req.ProxyNames)

	h.respondJSON(c, http.StatusOK, gin.H{
		"groups": groups,
	})
}

// GenerateRules generates rule configurations
func (h *Handler) GenerateRules(c *gin.Context) {
	var req struct {
		CustomRules []string `json:"custom_rules"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	generator := policy.NewGenerator()
	rules := generator.GenerateRules(req.CustomRules)

	h.respondJSON(c, http.StatusOK, gin.H{
		"rules": rules,
	})
}

// ValidateRule validates a rule
func (h *Handler) ValidateRule(c *gin.Context) {
	var req struct {
		Rule string `json:"rule" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	generator := policy.NewGenerator()
	if err := generator.ValidateRule(req.Rule); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid rule: "+err.Error())
		return
	}

	h.respondSuccess(c, "Rule is valid", nil)
}

// CreateCustomGroup creates a custom proxy group
func (h *Handler) CreateCustomGroup(c *gin.Context) {
	var req policy.GroupConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	generator := policy.NewGenerator()
	group, err := generator.GenerateCustomGroup(req)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(c, http.StatusCreated, group)
}
