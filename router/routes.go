package router

import (
	"github.com/gin-gonic/gin"
	"github.com/ashisharyan/ghostwriter-prompt-engine/handlers"
)

func RegisterRoutes(r *gin.Engine, profileHandler *handlers.ProfileHandler, promptABHandler *handlers.PromptABHandler, promptHandler *handlers.PromptHandler, scriptHandler *handlers.ScriptHandler) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/submit-profile", profileHandler.SubmitProfile)
		api.POST("/generate-prompt", profileHandler.GeneratePrompt)
		api.POST("/generate-prompt-ab", promptABHandler.GeneratePromptAB)
		api.GET("/prompts/:creator_id", promptHandler.GetPromptsByCreatorID)
		api.POST("/generate-script", scriptHandler.GenerateScript)
		api.GET("/scripts/:creator_id", scriptHandler.GetScriptsByCreatorID)
	}
}

