package handlers

import (
	"net/http"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/gin-gonic/gin"
)

type PromptABHandler struct {
	ProfileService  services.ProfileService
	PersonaService  services.PersonaService
	PromptABService services.PromptABService
}

func NewPromptABHandler(profileService services.ProfileService, personaService services.PersonaService, promptABService services.PromptABService) *PromptABHandler {
	return &PromptABHandler{
		ProfileService:  profileService,
		PersonaService:  personaService,
		PromptABService: promptABService,
	}
}

func (h *PromptABHandler) GeneratePromptAB(c *gin.Context) {
	var req models.GeneratePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
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
	contexts, personaSummary := h.PromptABService.GeneratePromptVariants(profile, persona, req.Topic)
	err = h.PromptABService.StorePromptVariants(contexts, profile, req.Topic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store prompt variants: " + err.Error()})
		return
	}

	variants := map[string]string{}
	scoreSnapshot := models.PersonaScores{}
	for key, ctx := range contexts {
		variants[key] = ctx.FullPromptText
		scoreSnapshot = ctx.Scores
	}

	c.JSON(http.StatusOK, gin.H{
		"persona":         personaSummary,
		"persona_summary": personaSummary,
		"score_snapshot":  scoreSnapshot,
		"topic":           req.Topic,
		"variants":        variants,
	})
}
