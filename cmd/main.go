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
	"os"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/ashisharyan/ghostwriter-prompt-engine/db"
	"github.com/ashisharyan/ghostwriter-prompt-engine/handlers"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/ashisharyan/ghostwriter-prompt-engine/router"
	"github.com/ashisharyan/ghostwriter-prompt-engine/utils"

	// Swagger
	_ "github.com/ashisharyan/ghostwriter-prompt-engine/docs"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize DB
	dbConn, err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	r := gin.Default()

	// Swagger route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Dependency injection using interfaces
	var promptRepo services.PromptRepository = services.NewGormPromptRepository(dbConn)
	var profileService services.ProfileService = services.NewGormProfileService(dbConn)
	var promptService services.PromptService = services.NewDefaultPromptService(promptRepo)
	var promptABService services.PromptABService = services.NewDefaultPromptABService(promptService, promptRepo)
	promptABHandler := handlers.NewPromptABHandler(profileService, promptABService)
	profileHandler := handlers.NewProfileHandler(profileService, promptService)
	promptHandler := handlers.NewPromptHandler(promptRepo)
	var scriptRepo services.ScriptRepository = services.NewGormScriptRepository(dbConn)
	secrets := utils.LoadSecrets()
	var scriptService services.ScriptService = services.NewOpenAIScriptService(secrets.OpenAIAPIKey, secrets.OpenAIModel)
	scriptHandler := handlers.NewScriptHandler(promptRepo, scriptService, scriptRepo)

	// Register routes using router/routes.go
	router.RegisterRoutes(r, profileHandler, promptABHandler, promptHandler, scriptHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
