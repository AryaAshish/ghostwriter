package handlers

import (
	"net/http"

	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/gin-gonic/gin"
)

type OnboardingHandler struct {
	OnboardingService services.OnboardingService
}

func NewOnboardingHandler(onboardingService services.OnboardingService) *OnboardingHandler {
	return &OnboardingHandler{OnboardingService: onboardingService}
}

func (h *OnboardingHandler) GetQuestions(c *gin.Context) {
	path := c.Query("voice_input_path")
	genre := c.Query("genre")
	var questions interface{}
	if path != "" {
		questions = h.OnboardingService.GetQuestionsForPath(path, genre)
	} else {
		questions = h.OnboardingService.GetQuestions()
	}
	c.JSON(http.StatusOK, gin.H{
		"questions": questions,
	})
}
