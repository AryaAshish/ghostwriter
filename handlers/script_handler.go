package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/gin-gonic/gin"
)

type ScriptHandler struct {
	PromptRepo    services.PromptRepository
	ScriptService services.ScriptService
	ScriptRepo    services.ScriptRepository
}

func NewScriptHandler(promptRepo services.PromptRepository, scriptService services.ScriptService, scriptRepo services.ScriptRepository) *ScriptHandler {
	return &ScriptHandler{PromptRepo: promptRepo, ScriptService: scriptService, ScriptRepo: scriptRepo}
}

func (h *ScriptHandler) GetScriptsByCreatorID(c *gin.Context) {
	creatorID, err := parseCreatorID(c.Param("creator_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid creator_id"})
		return
	}
	scripts, err := h.ScriptRepo.GetScriptsByCreatorIDWithPrompt(creatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scripts: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, scripts)
}

func (h *ScriptHandler) GenerateScript(c *gin.Context) {
	var req models.GenerateScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Variant == "" {
		req.Variant = "base"
	}

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

	systemPrompt := prompt.SystemPrompt
	userPrompt := prompt.UserPrompt
	if userPrompt == "" {
		userPrompt = prompt.PromptText
	}

	scriptText, err := h.ScriptService.GenerateScriptFromPrompt(systemPrompt, userPrompt)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "OpenAI error: " + err.Error()})
		return
	}

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
		"script_id":    script.ID,
		"prompt_id":    prompt.ID,
		"prompt":       prompt.PromptText,
		"script":       script.ScriptText,
		"source":       script.Source,
		"created_at":   script.CreatedAt,
	})
}
