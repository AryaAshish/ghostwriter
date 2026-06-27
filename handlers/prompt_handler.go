package handlers

import (
	"net/http"

	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/gin-gonic/gin"
)

type PromptHandler struct {
	PromptRepo services.PromptRepository
}

func NewPromptHandler(promptRepo services.PromptRepository) *PromptHandler {
	return &PromptHandler{PromptRepo: promptRepo}
}

func (h *PromptHandler) GetPromptsByCreatorID(c *gin.Context) {
	creatorID, err := parseCreatorID(c.Param("creator_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid creator_id"})
		return
	}
	prompts, err := h.PromptRepo.GetPromptsByCreatorID(creatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prompts: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prompts": prompts})
}
