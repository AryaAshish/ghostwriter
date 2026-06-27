package router

import (
	"github.com/ashisharyan/ghostwriter-prompt-engine/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	r *gin.Engine,
	onboardingHandler *handlers.OnboardingHandler,
	profileHandler *handlers.ProfileHandler,
	personaHandler *handlers.PersonaHandler,
	promptABHandler *handlers.PromptABHandler,
	promptHandler *handlers.PromptHandler,
	scriptHandler *handlers.ScriptHandler,
	feedbackHandler *handlers.FeedbackHandler,
	instagramHandler *handlers.InstagramHandler,
) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.GET("/onboarding/questions", onboardingHandler.GetQuestions)
		api.POST("/submit-profile", profileHandler.SubmitProfile)
		api.GET("/persona/:creator_id", personaHandler.GetPersona)
		api.PATCH("/persona/:creator_id", personaHandler.UpdatePersona)
		api.POST("/persona/:creator_id/analyze-voice", personaHandler.ReanalyzeVoice)
		api.POST("/generate-prompt", profileHandler.GeneratePrompt)
		api.POST("/generate-prompt-ab", promptABHandler.GeneratePromptAB)
		api.GET("/prompts/:creator_id", promptHandler.GetPromptsByCreatorID)
		api.POST("/generate-script", scriptHandler.GenerateScript)
		api.GET("/scripts/:creator_id", scriptHandler.GetScriptsByCreatorID)
		api.POST("/scripts/:script_id/feedback", feedbackHandler.SubmitFeedback)

		api.GET("/instagram/status", instagramHandler.Status)
		api.GET("/instagram/auth-url", instagramHandler.AuthURL)
		api.GET("/instagram/callback", instagramHandler.Callback)
		api.GET("/instagram/reels", instagramHandler.Reels)
		api.POST("/instagram/prepare", instagramHandler.Prepare)
	}
}
