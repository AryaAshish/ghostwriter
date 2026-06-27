package main

// @title Prompt Engine API
// @version 1.0
// @description AI-powered backend for personalized script generation for creators.
// @host localhost:8080
// @BasePath /api/v1
// @schemes http
// @contact.name Ashish Aryan
// @contact.email ashish@example.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

import (
	"log"
	"net/http"
	"os"

	"github.com/ashisharyan/ghostwriter-prompt-engine/db"
	"github.com/ashisharyan/ghostwriter-prompt-engine/handlers"
	"github.com/ashisharyan/ghostwriter-prompt-engine/router"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/ashisharyan/ghostwriter-prompt-engine/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbConn, err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE, UPDATE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	secrets := utils.LoadSecrets()
	openAIClient := services.NewOpenAIClient(secrets.OpenAIAPIKey, secrets.OpenAIModel)

	onboardingService := services.NewDefaultOnboardingService()
	voiceAnalysisService := services.NewOpenAIVoiceAnalysisService(openAIClient)
	personaService := services.NewGormPersonaService(dbConn, onboardingService, voiceAnalysisService)

	var promptRepo services.PromptRepository = services.NewGormPromptRepository(dbConn)
	var profileService services.ProfileService = services.NewGormProfileService(dbConn)
	var promptService services.PromptService = services.NewDefaultPromptService(promptRepo, personaService)
	var promptABService services.PromptABService = services.NewDefaultPromptABService(personaService, promptRepo)
	var scriptRepo services.ScriptRepository = services.NewGormScriptRepository(dbConn)
	var scriptService services.ScriptService = services.NewOpenAIScriptService(openAIClient)
	feedbackService := services.NewGormFeedbackService(dbConn, personaService, scriptRepo)

	onboardingHandler := handlers.NewOnboardingHandler(onboardingService)
	profileHandler := handlers.NewProfileHandler(profileService, promptService, personaService)
	personaHandler := handlers.NewPersonaHandler(personaService)
	promptABHandler := handlers.NewPromptABHandler(profileService, personaService, promptABService)
	promptHandler := handlers.NewPromptHandler(promptRepo)
	scriptHandler := handlers.NewScriptHandler(promptRepo, scriptService, scriptRepo)
	feedbackHandler := handlers.NewFeedbackHandler(feedbackService)
	transcriber := services.NewOpenAITranscriptionService(secrets.OpenAIAPIKey)
	instagramService := services.NewDefaultInstagramService(secrets, transcriber)
	instagramHandler := handlers.NewInstagramHandler(instagramService, secrets.AppBaseURL)

	router.RegisterRoutes(r, onboardingHandler, profileHandler, personaHandler, promptABHandler, promptHandler, scriptHandler, feedbackHandler, instagramHandler)

	r.Static("/planning", "./planning")

	r.Static("/app", "./web")
	r.GET("/playground", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/app/")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
