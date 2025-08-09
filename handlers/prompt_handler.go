package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
)

type PromptHandler struct {
	PromptRepo services.PromptRepository
}

func NewPromptHandler(promptRepo services.PromptRepository) *PromptHandler {
	return &PromptHandler{PromptRepo: promptRepo}
}

// GetPromptsByCreatorID godoc
// @Summary Get prompts by creator
// @Description Retrieve all prompts for a given creator
// @Tags prompt
// @Produce json
// @Param creator_id path string true "Creator ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /prompts/{creator_id} [get]
func (h *PromptHandler) GetPromptsByCreatorID(c *gin.Context) {
	creatorID := c.Param("creator_id")
	prompts, err := h.PromptRepo.GetPromptsByCreatorID(creatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prompts: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prompts": prompts})
}
