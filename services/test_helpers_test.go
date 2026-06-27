package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.CreatorProfile{},
		&models.PersonaProfile{},
		&models.Prompt{},
		&models.Script{},
		&models.ScriptFeedback{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}

func testProfile() *models.CreatorProfile {
	return &models.CreatorProfile{
		Name:        "Amit",
		Genre:       "Comedy",
		Language:    "Hindi",
		Region:      "Delhi",
		Bio:         "Standup comic",
		Goal:        "Grow audience",
		Audience:    "18-25",
		Tone:        "Witty",
		Style:       "Relatable",
		Inspiration: "Vir Das",
		ContentType: "Shorts",
		Platform:    "YouTube",
		Experience:  "2 years",
		USP:         "Quick punchlines",
	}
}

func seedCreator(t *testing.T, db *gorm.DB) (*models.CreatorProfile, *models.PersonaProfile) {
	t.Helper()
	profile := testProfile()
	if err := db.Create(profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
	onboarding := NewDefaultOnboardingService()
	personaSvc := NewGormPersonaService(db, onboarding, nil)
	persona, err := personaSvc.CreatePersonaFromOnboarding(profile.ID, profile, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathSkipCalibrate,
		StyleAnswers: map[string]interface{}{
			"humor_style":     "funny",
			"formality_level": "casual",
			"preferred_words": "yaar",
			"avoid_words":     "delve",
			"anti_voice":      "corporate trainer",
		},
	})
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	return profile, persona
}

type mockVoiceAnalysisService struct {
	result *VoiceAnalysisResult
	err    error
}

func (m *mockVoiceAnalysisService) AnalyzeVoice(samples []string, baseline models.PersonaScores, profile *models.CreatorProfile) (*VoiceAnalysisResult, error) {
	return m.result, m.err
}

func newMockOpenAIServer(t *testing.T, content string, statusCode int) *httptest.Server {
	t.Helper()
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if statusCode != http.StatusOK {
			_, _ = w.Write([]byte(`{"error":"fail"}`))
			return
		}
		resp := openAIChatResponse{}
		resp.Choices = append(resp.Choices, struct {
			Message openAIMessage `json:"message"`
		}{Message: openAIMessage{Role: "assistant", Content: content}})
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func newOpenAIClientForTest(t *testing.T, server *httptest.Server) *OpenAIClient {
	t.Helper()
	return &OpenAIClient{
		APIKey:     "test-key",
		Model:      "gpt-4",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}
}
