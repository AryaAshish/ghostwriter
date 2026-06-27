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
	PersonaService services.PersonaService
}

func NewProfileHandler(profileService services.ProfileService, promptService services.PromptService, personaService services.PersonaService) *ProfileHandler {
	return &ProfileHandler{
		ProfileService: profileService,
		PromptService:  promptService,
		PersonaService: personaService,
	}
}

func (h *ProfileHandler) SubmitProfile(c *gin.Context) {
	var req models.SubmitProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}

	validation := services.ValidateSubmitProfile(req)
	if !validation.OK {
		c.JSON(http.StatusBadRequest, gin.H{"error": validation.Message})
		return
	}

	applyInstagramProfileHints(&req)

	id, err := h.ProfileService.CreateProfile(&req.CreatorProfile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save profile: " + err.Error()})
		return
	}

	persona, err := h.PersonaService.CreatePersonaFromOnboarding(id, &req.CreatorProfile, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Profile saved but persona creation failed: " + err.Error()})
		return
	}

	resp := gin.H{
		"creator_id": id,
		"message":    "Profile submitted successfully",
		"persona":    persona,
	}
	if validation.Warning != "" {
		resp["warning"] = validation.Warning
	}
	c.JSON(http.StatusOK, resp)
}

func applyInstagramProfileHints(req *models.SubmitProfileRequest) {
	if req.Instagram == nil {
		return
	}
	hints := services.ProfileHintsFromInstagram(*req.Instagram)
	if req.Name == "" && hints["name"] != "" {
		req.Name = hints["name"]
	}
	if req.Bio == "" && hints["bio"] != "" {
		req.Bio = hints["bio"]
	}
	if req.Platform == "" && hints["platform"] != "" {
		req.Platform = hints["platform"]
	}
	if req.Channel == "" && hints["channel"] != "" {
		req.Channel = hints["channel"]
	}
	if req.ContentType == "" && hints["content_type"] != "" {
		req.ContentType = hints["content_type"]
	}
}

func (h *ProfileHandler) GeneratePrompt(c *gin.Context) {
	var req models.GeneratePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profile, err := h.ProfileService.GetProfileByID(req.CreatorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}
	persona, err := h.PersonaService.GetOrDefaultPersona(req.CreatorID, profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx, perr := h.PromptService.GeneratePrompt(profile, persona, req.Topic)
	if perr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save prompt: " + perr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"persona":           ctx.PersonaSummary,
		"persona_summary":   ctx.PersonaSummary,
		"score_snapshot":    ctx.Scores,
		"prompt":            ctx.FullPromptText,
		"system_prompt":     ctx.SystemPrompt,
		"user_prompt":       ctx.UserPrompt,
		"voice_mode":        ctx.VoiceMode,
		"voice_confidence":  ctx.VoiceConfidence,
	})
}
