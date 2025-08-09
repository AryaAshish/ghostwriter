package handlers

import (
	"net/http"
	"os"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

type ScriptHandler struct {
	PromptRepo    services.PromptRepository
	ScriptService services.ScriptService
	ScriptRepo    services.ScriptRepository
}

// GetScriptsByCreatorID godoc
// @Summary Get scripts by creator
// @Description Retrieve all scripts for a given creator, joined with prompt data
// @Tags script
// @Produce json
// @Param creator_id path string true "Creator ID"
// @Success 200 {array} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /scripts/{creator_id} [get]
func (h *ScriptHandler) GetScriptsByCreatorID(c *gin.Context) {
	creatorID := c.Param("creator_id")
	if creatorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "creator_id required"})
		return
	}
	scripts, err := h.ScriptRepo.GetScriptsByCreatorIDWithPrompt(creatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scripts: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, scripts)
}

func NewScriptHandler(promptRepo services.PromptRepository, scriptService services.ScriptService, scriptRepo services.ScriptRepository) *ScriptHandler {
	return &ScriptHandler{PromptRepo: promptRepo, ScriptService: scriptService, ScriptRepo: scriptRepo}
}

// GenerateScript godoc
// @Summary Generate script from prompt
// @Description Generate a final script using OpenAI for a given creator, topic, and variant
// @Tags script
// @Accept json
// @Produce json
// @Param request body models.GenerateScriptRequest true "Script request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /generate-script [post]
func (h *ScriptHandler) GenerateScript(c *gin.Context) {
	var req struct {
		CreatorID string `json:"creator_id" binding:"required"`
		Topic     string `json:"topic" binding:"required"`
		Variant   string `json:"variant"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Default variant
	if req.Variant == "" {
		req.Variant = "base"
	}
	// Fetch prompt
	prompts, err := h.PromptRepo.GetPromptsByCreatorID(req.CreatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prompts: " + err.Error()})
		return
	}
	var prompt *models.Prompt
	for i := range prompts {
		if prompts[i].Topic == req.Topic && prompts[i].Variant == req.Variant {
			prompt = &prompts[i]
			break
		}
	}
	if prompt == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prompt not found for creator/topic/variant"})
		return
	}
	// Call OpenAI
	scriptText, err := h.ScriptService.GenerateScriptFromPrompt(prompt.PromptText)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "OpenAI error: " + err.Error()})
		return
	}
	// Save script
	script := &models.Script{
		CreatorID:  req.CreatorID,
		PromptID:   prompt.ID,
		ScriptText: scriptText,
		Source:     os.Getenv("OPENAI_MODEL"),
		CreatedAt:  time.Now(),
	}
	if err := h.ScriptRepo.SaveScript(script); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save script: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"prompt_id": prompt.ID,
		"prompt":    prompt.PromptText,
		"script":    script.ScriptText,
		"source":    script.Source,
		"created_at": script.CreatedAt,
	})
}
