package handlers

import (
	"net/http"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	ProfileService services.ProfileService
	PromptService  services.PromptService
}

func NewProfileHandler(profileService services.ProfileService, promptService services.PromptService) *ProfileHandler {
	return &ProfileHandler{ProfileService: profileService, PromptService: promptService}
}

// SubmitProfile godoc
// @Summary Submit creator profile
// @Description Onboard a new creator and store their profile
// @Tags profile
// @Accept json
// @Produce json
// @Param profile body models.CreatorProfile true "Creator profile"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /submit-profile [post]
func (h *ProfileHandler) SubmitProfile(c *gin.Context) {
	var profile models.CreatorProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}
	id, err := h.ProfileService.CreateProfile(&profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save profile: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"creator_id": id, "message": "Profile submitted successfully"})
}

// GeneratePrompt godoc
// @Summary Generate prompt for creator
// @Description Generate a persona and script prompt for a creator and topic
// @Tags prompt
// @Accept json
// @Produce json
// @Param request body models.GeneratePromptRequest true "Prompt request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /generate-prompt [post]
func (h *ProfileHandler) GeneratePrompt(c *gin.Context) {
	var req struct {
		CreatorID uint   `json:"creator_id" binding:"required"`
		Topic     string `json:"topic" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profile, err := h.ProfileService.GetProfileByID(req.CreatorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}
	persona := h.PromptService.GeneratePersonaSummary(profile)
	prompt, perr := h.PromptService.GeneratePrompt(profile, req.Topic)
	if perr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save prompt: " + perr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"persona": persona,
		"prompt":  prompt,
	})
}
