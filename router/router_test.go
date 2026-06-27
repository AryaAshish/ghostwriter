package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/handlers"
	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/ashisharyan/ghostwriter-prompt-engine/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.CreatorProfile{},
		&models.PersonaProfile{},
		&models.Prompt{},
		&models.Script{},
		&models.ScriptFeedback{},
	); err != nil {
		t.Fatal(err)
	}

	onboardingService := services.NewDefaultOnboardingService()
	voiceService := &mockVoiceService{}
	personaService := services.NewGormPersonaService(db, onboardingService, voiceService)
	promptRepo := services.NewGormPromptRepository(db)
	profileService := services.NewGormProfileService(db)
	promptService := services.NewDefaultPromptService(promptRepo, personaService)
	promptABService := services.NewDefaultPromptABService(personaService, promptRepo)
	scriptRepo := services.NewGormScriptRepository(db)
	scriptService := &mockScriptService{text: "generated script"}
	feedbackService := services.NewGormFeedbackService(db, personaService, scriptRepo)
	instagramService := services.NewDefaultInstagramService(&utils.Secrets{}, nil)

	r := gin.New()
	RegisterRoutes(r,
		handlers.NewOnboardingHandler(onboardingService),
		handlers.NewProfileHandler(profileService, promptService, personaService),
		handlers.NewPersonaHandler(personaService),
		handlers.NewPromptABHandler(profileService, personaService, promptABService),
		handlers.NewPromptHandler(promptRepo),
		handlers.NewScriptHandler(promptRepo, scriptService, scriptRepo),
		handlers.NewFeedbackHandler(feedbackService),
		handlers.NewInstagramHandler(instagramService, "http://localhost"),
	)
	return r
}

type mockVoiceService struct{}

func (m *mockVoiceService) AnalyzeVoice(samples []string, baseline models.PersonaScores, profile *models.CreatorProfile) (*services.VoiceAnalysisResult, error) {
	return &services.VoiceAnalysisResult{
		SuggestedScores: models.PersonaScores{Humor: 80},
		LexicalProfile:  models.LexicalProfile{SignaturePhrases: []string{"matlab"}},
		VoiceSummary:    "Mock voice",
	}, nil
}

type mockScriptService struct {
	text string
}

func (m *mockScriptService) GenerateScriptFromPrompt(systemPrompt, userPrompt string) (string, error) {
	return m.text, nil
}

func performRequest(r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRoutes(t *testing.T) {
	r := setupTestRouter(t)

	if w := performRequest(r, http.MethodGet, "/", nil); w.Code != http.StatusOK {
		t.Fatal("health failed")
	}
	if w := performRequest(r, http.MethodGet, "/api/v1/onboarding/questions", nil); w.Code != http.StatusOK {
		t.Fatal("questions failed")
	}

	submit := performRequest(r, http.MethodPost, "/api/v1/submit-profile", map[string]interface{}{
		"name": "Amit", "genre": "Comedy", "language": "Hindi",
		"voice_input_path": models.VoiceInputPathSkipCalibrate,
		"style_answers": map[string]interface{}{
			"humor_style":     "funny",
			"preferred_words": "yaar",
			"avoid_words":     "delve",
			"anti_voice":      "news anchor",
		},
	})
	if submit.Code != http.StatusOK {
		t.Fatalf("submit failed: %s", submit.Body.String())
	}
	if w := performRequest(r, http.MethodPost, "/api/v1/submit-profile", map[string]interface{}{"genre": "x"}); w.Code != http.StatusBadRequest {
		t.Fatal("expected bad submit")
	}

	if w := performRequest(r, http.MethodGet, "/api/v1/persona/1", nil); w.Code != http.StatusOK {
		t.Fatal("get persona failed")
	}
	if w := performRequest(r, http.MethodGet, "/api/v1/persona/abc", nil); w.Code != http.StatusBadRequest {
		t.Fatal("bad persona id")
	}
	if w := performRequest(r, http.MethodGet, "/api/v1/persona/999", nil); w.Code != http.StatusNotFound {
		t.Fatal("missing persona")
	}

	if w := performRequest(r, http.MethodPatch, "/api/v1/persona/1", map[string]interface{}{"voice_summary": "updated"}); w.Code != http.StatusOK {
		t.Fatalf("patch persona failed: %s", w.Body.String())
	}
	if w := performRequest(r, http.MethodPost, "/api/v1/persona/1/analyze-voice", map[string]interface{}{
		"writing_samples": []string{"new"},
	}); w.Code != http.StatusOK {
		t.Fatalf("analyze voice failed: %s", w.Body.String())
	}

	if w := performRequest(r, http.MethodPost, "/api/v1/generate-prompt", map[string]interface{}{
		"creator_id": 1, "topic": "viral",
	}); w.Code != http.StatusOK {
		t.Fatalf("generate prompt failed: %s", w.Body.String())
	}
	if w := performRequest(r, http.MethodPost, "/api/v1/generate-prompt", map[string]interface{}{
		"creator_id": 999, "topic": "x",
	}); w.Code != http.StatusNotFound {
		t.Fatal("missing profile for prompt")
	}

	if w := performRequest(r, http.MethodPost, "/api/v1/generate-prompt-ab", map[string]interface{}{
		"creator_id": 1, "topic": "viral",
	}); w.Code != http.StatusOK {
		t.Fatalf("generate ab failed: %s", w.Body.String())
	}
	if w := performRequest(r, http.MethodGet, "/api/v1/prompts/1", nil); w.Code != http.StatusOK {
		t.Fatal("list prompts failed")
	}
	if w := performRequest(r, http.MethodGet, "/api/v1/prompts/abc", nil); w.Code != http.StatusBadRequest {
		t.Fatal("bad prompts id")
	}

	if w := performRequest(r, http.MethodPost, "/api/v1/generate-script", map[string]interface{}{
		"creator_id": 1, "topic": "viral", "variant": "base",
	}); w.Code != http.StatusOK {
		t.Fatalf("generate script failed: %s", w.Body.String())
	}
	if w := performRequest(r, http.MethodPost, "/api/v1/generate-script", map[string]interface{}{
		"creator_id": 1, "topic": "missing",
	}); w.Code != http.StatusNotFound {
		t.Fatal("missing prompt for script")
	}
	if w := performRequest(r, http.MethodGet, "/api/v1/scripts/1", nil); w.Code != http.StatusOK {
		t.Fatal("list scripts failed")
	}

	if w := performRequest(r, http.MethodPost, "/api/v1/scripts/1/feedback", map[string]interface{}{
		"rating": "not_quite", "notes": "too formal",
	}); w.Code != http.StatusOK {
		t.Fatalf("feedback failed: %s", w.Body.String())
	}
	if w := performRequest(r, http.MethodPost, "/api/v1/scripts/999/feedback", map[string]interface{}{
		"rating": "no",
	}); w.Code != http.StatusInternalServerError {
		t.Fatal("missing script feedback")
	}
	if w := performRequest(r, http.MethodPost, "/api/v1/scripts/1/feedback", map[string]interface{}{}); w.Code != http.StatusBadRequest {
		t.Fatal("bad feedback body")
	}
}
