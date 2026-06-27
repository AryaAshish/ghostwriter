package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/gin-gonic/gin"
)

func TestSubmitProfileValidationAndWarning(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewProfileHandler(&stubProfileService{}, &stubPromptService{}, &stubPersonaService{createResult: &models.PersonaProfile{CreatorID: 1}})

	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{
		"name": "A", "genre": "G", "language": "L",
		"voice_input_path": models.VoiceInputPathPasteScripts,
		"writing_samples":  []string{"one"},
	}, h.SubmitProfile); w.Code != http.StatusBadRequest {
		t.Fatal("expected paste validation error")
	}

	h = NewProfileHandler(&stubProfileService{}, &stubPromptService{}, &stubPersonaService{createResult: &models.PersonaProfile{CreatorID: 1}})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{
		"name": "A", "genre": "G", "language": "L",
		"voice_input_path": models.VoiceInputPathPasteScripts,
		"writing_samples":  []string{"one two three four five six seven eight nine ten", "one two three four five six seven eight nine ten"},
	}, h.SubmitProfile); w.Code != http.StatusOK {
		t.Fatalf("expected warn submit ok, got %d %s", w.Code, w.Body.String())
	}
}

func TestOnboardingHandlerPathQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oh := NewOnboardingHandler(&stubOnboardingService{questions: []models.OnboardingQuestion{{ID: "q1"}}})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?voice_input_path=guided_write&genre=comedy", nil)
	oh.GetQuestions(c)
	if w.Code != http.StatusOK {
		t.Fatal("expected path questions")
	}
}

func TestScriptHandlerBadCreatorID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sh := NewScriptHandler(&stubPromptRepo{}, &stubScriptService{}, &stubScriptRepo{})
	if code := runHandler(func(c *gin.Context) {
		c.Params = gin.Params{{Key: "creator_id", Value: "bad"}}
		sh.GetScriptsByCreatorID(c)
	}); code != http.StatusBadRequest {
		t.Fatal("expected bad creator id for scripts list")
	}
}

func TestPersonaReanalyzeBadCreatorIDOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPersonaHandler(&stubPersonaService{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Params = gin.Params{{Key: "creator_id", Value: "bad"}}
	h.ReanalyzeVoice(c)
	if w.Code != http.StatusBadRequest {
		t.Fatal("expected bad id on reanalyze")
	}
}

func TestPersonaUpdateNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPersonaHandler(&stubPersonaService{updateErr: errors.New("missing")})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"voice_summary":"x"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
	h.UpdatePersona(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatal("expected update error")
	}
}

func TestGeneratePromptABPersonaError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ab := NewPromptABHandler(
		&stubProfileService{getResult: &models.CreatorProfile{}},
		&stubPersonaService{getErr: errors.New("persona fail"), getResult: nil},
		&stubPromptABService{},
	)
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "t"}, ab.GeneratePromptAB); w.Code != http.StatusInternalServerError {
		t.Fatal("expected persona error on ab")
	}
}

func TestPersonaUpdateBadCreatorID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPersonaHandler(&stubPersonaService{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"voice_summary":"x"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "creator_id", Value: "abc"}}
	h.UpdatePersona(c)
	if w.Code != http.StatusBadRequest {
		t.Fatal("expected bad id on update")
	}
}
