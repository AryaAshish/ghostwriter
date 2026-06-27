package handlers

import (
	"net/http"
	"strconv"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/gin-gonic/gin"
)

type FeedbackHandler struct {
	FeedbackService services.FeedbackService
}

func NewFeedbackHandler(feedbackService services.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{FeedbackService: feedbackService}
}

func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	scriptID, err := strconv.ParseUint(c.Param("script_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid script_id"})
		return
	}
	var req models.ScriptFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.FeedbackService.SubmitFeedback(uint(scriptID), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"feedback":          result.Feedback,
		"applied_deltas":    result.Deltas,
		"voice_mode":        result.VoiceMode,
		"voice_confidence":  result.VoiceConfidence,
		"shift_score":       result.ShiftScore,
		"message":           "Feedback recorded and persona updated",
	})
}
