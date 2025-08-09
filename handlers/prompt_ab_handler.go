package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
)

type PromptABHandler struct {
	ProfileService services.ProfileService
	PromptABService services.PromptABService
}

func NewPromptABHandler(profileService services.ProfileService, promptABService services.PromptABService) *PromptABHandler {
	return &PromptABHandler{ProfileService: profileService, PromptABService: promptABService}
}

// GeneratePromptAB godoc
// @Summary Generate A/B/C prompt variants
// @Description Generate three prompt variants (A/B/C) for a creator and topic
// @Tags prompt
// @Accept json
// @Produce json
// @Param request body models.GeneratePromptRequest true "Prompt AB request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /generate-prompt-ab [post]
func (h *PromptABHandler) GeneratePromptAB(c *gin.Context) {
	var req struct {
		CreatorID uint   `json:"creator_id"`
		Topic     string `json:"topic"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}
	profile, err := h.ProfileService.GetProfileByID(req.CreatorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}
	variants, persona := h.PromptABService.GeneratePromptVariants(profile, req.Topic)
	err = h.PromptABService.StorePromptVariants(variants, profile, req.Topic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store prompt variants: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"persona_summary": persona,
		"topic": req.Topic,
		"variants": variants,
	})
}
