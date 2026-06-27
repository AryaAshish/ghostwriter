package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/gin-gonic/gin"
)

type stubProfileService struct {
	createErr error
	getResult *models.CreatorProfile
	getErr    error
}

func (s *stubProfileService) CreateProfile(profile *models.CreatorProfile) (uint, error) {
	if s.createErr != nil {
		return 0, s.createErr
	}
	profile.ID = 1
	return 1, nil
}

func (s *stubProfileService) GetProfileByID(id uint) (*models.CreatorProfile, error) {
	return s.getResult, s.getErr
}

type stubPersonaService struct {
	createResult *models.PersonaProfile
	createErr    error
	getResult    *models.PersonaProfile
	getErr       error
	updateResult *models.PersonaProfile
	updateErr    error
	reanalyzeErr error
	promptCtx    services.PromptContext
}

func (s *stubPersonaService) CreatePersonaFromOnboarding(creatorID uint, profile *models.CreatorProfile, req models.SubmitProfileRequest) (*models.PersonaProfile, error) {
	if s.createResult == nil {
		return &models.PersonaProfile{CreatorID: creatorID, VoiceMode: models.PersonaModeDeclared}, s.createErr
	}
	return s.createResult, s.createErr
}
func (s *stubPersonaService) GetPersona(creatorID uint) (*models.PersonaProfile, error) {
	return s.getResult, s.getErr
}
func (s *stubPersonaService) UpdatePersona(creatorID uint, req models.UpdatePersonaRequest) (*models.PersonaProfile, error) {
	return s.updateResult, s.updateErr
}
func (s *stubPersonaService) ReanalyzeVoice(creatorID uint, samples []string) (*models.PersonaProfile, error) {
	return s.getResult, s.reanalyzeErr
}
func (s *stubPersonaService) BuildPersonaSummary(profile *models.CreatorProfile, persona *models.PersonaProfile) string {
	return "summary"
}
func (s *stubPersonaService) BuildLexicalRules(lexical models.LexicalProfile) string { return "rules" }
func (s *stubPersonaService) BuildPromptContext(profile *models.CreatorProfile, persona *models.PersonaProfile, topic, variant string) services.PromptContext {
	if s.promptCtx.FullPromptText == "" {
		return services.PromptContext{PersonaSummary: "summary", FullPromptText: "prompt", Scores: models.PersonaScores{Humor: 70}}
	}
	return s.promptCtx
}
func (s *stubPersonaService) ApplyFeedback(creatorID uint, req models.ScriptFeedbackRequest) (services.FeedbackResult, error) {
	return services.FeedbackResult{Deltas: map[string]int{"humor": 5}, VoiceMode: models.PersonaModeDeclared, VoiceConfidence: 20}, nil
}
func (s *stubPersonaService) GetOrDefaultPersona(creatorID uint, profile *models.CreatorProfile) (*models.PersonaProfile, error) {
	if s.getResult != nil {
		return s.getResult, nil
	}
	return &models.PersonaProfile{CreatorID: creatorID}, s.getErr
}

type stubPromptService struct {
	ctx services.PromptContext
	err error
}

func (s *stubPromptService) GeneratePersonaSummary(profile *models.CreatorProfile, persona *models.PersonaProfile) string {
	return "summary"
}
func (s *stubPromptService) GeneratePrompt(profile *models.CreatorProfile, persona *models.PersonaProfile, topic string) (services.PromptContext, error) {
	if s.ctx.FullPromptText == "" {
		return services.PromptContext{PersonaSummary: "summary", FullPromptText: "prompt", Scores: models.PersonaScores{}}, s.err
	}
	return s.ctx, s.err
}

type stubPromptABService struct {
	variants map[string]services.PromptContext
	summary  string
	storeErr error
}

func (s *stubPromptABService) GeneratePromptVariants(profile *models.CreatorProfile, persona *models.PersonaProfile, topic string) (map[string]services.PromptContext, string) {
	if s.variants == nil {
		return map[string]services.PromptContext{
			"A": {FullPromptText: "a"},
			"B": {FullPromptText: "b"},
			"C": {FullPromptText: "c", Scores: models.PersonaScores{Humor: 1}},
		}, "summary"
	}
	return s.variants, s.summary
}
func (s *stubPromptABService) StorePromptVariants(contexts map[string]services.PromptContext, profile *models.CreatorProfile, topic string) error {
	return s.storeErr
}

type stubPromptRepo struct {
	prompts []models.Prompt
	err     error
}

func (s *stubPromptRepo) SavePrompt(prompt *models.Prompt) error { return s.err }
func (s *stubPromptRepo) GetPromptsByCreatorID(creatorID uint) ([]models.Prompt, error) {
	return s.prompts, s.err
}

type stubScriptRepo struct {
	scripts []services.ScriptWithPrompt
	script  *models.Script
	err     error
}

func (s *stubScriptRepo) SaveScript(script *models.Script) error {
	script.ID = 1
	return s.err
}
func (s *stubScriptRepo) GetScriptByID(scriptID uint) (*models.Script, error) {
	return s.script, s.err
}
func (s *stubScriptRepo) GetScriptsByCreatorIDWithPrompt(creatorID uint) ([]services.ScriptWithPrompt, error) {
	return s.scripts, s.err
}

type stubScriptService struct {
	text string
	err  error
}

func (s *stubScriptService) GenerateScriptFromPrompt(systemPrompt, userPrompt string) (string, error) {
	return s.text, s.err
}

type stubFeedbackService struct {
	feedback *models.ScriptFeedback
	deltas   map[string]int
	err      error
}

func (s *stubFeedbackService) SubmitFeedback(scriptID uint, req models.ScriptFeedbackRequest) (*services.FeedbackSubmitResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &services.FeedbackSubmitResult{
		Feedback:        s.feedback,
		Deltas:          s.deltas,
		VoiceMode:       models.PersonaModeDeclared,
		VoiceConfidence: 25,
		ShiftScore:      50,
	}, nil
}

type stubOnboardingService struct {
	questions []models.OnboardingQuestion
}

func (s *stubOnboardingService) GetQuestions() []models.OnboardingQuestion {
	return s.questions
}
func (s *stubOnboardingService) GetQuestionsForPath(voiceInputPath, genre string) []models.OnboardingQuestion {
	return s.questions
}
func (s *stubOnboardingService) MapAnswersToBaseline(answers map[string]interface{}) models.PersonaScores {
	return models.PersonaScores{Humor: 70}
}
func (s *stubOnboardingService) ExtractLexicalHints(answers map[string]interface{}) models.LexicalProfile {
	return models.LexicalProfile{}
}

func performHandlerRequest(method, path string, body interface{}, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, &buf)
	c.Request.Header.Set("Content-Type", "application/json")
	if path != "" {
		c.Params = gin.Params{{Key: "creator_id", Value: "1"}, {Key: "script_id", Value: "1"}}
	}
	handler(c)
	return w
}

func TestProfileHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	profile := &models.CreatorProfile{Name: "A", Genre: "G", Language: "L"}

	h := NewProfileHandler(&stubProfileService{}, &stubPromptService{}, &stubPersonaService{})
	if w := performHandlerRequest(http.MethodPost, "/", nil, h.SubmitProfile); w.Code != http.StatusBadRequest {
		t.Fatal("expected bad json")
	}

	h = NewProfileHandler(&stubProfileService{createErr: errors.New("fail")}, &stubPromptService{}, &stubPersonaService{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{
		"name": "A", "genre": "G", "language": "L",
		"voice_input_path": models.VoiceInputPathSkipCalibrate,
		"style_answers": map[string]interface{}{
			"preferred_words": "yaar",
			"avoid_words":     "delve",
			"anti_voice":      "news anchor",
		},
	}, h.SubmitProfile); w.Code != http.StatusInternalServerError {
		t.Fatal("expected create profile error")
	}

	h = NewProfileHandler(&stubProfileService{}, &stubPromptService{}, &stubPersonaService{createErr: errors.New("persona fail")})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{
		"name": "A", "genre": "G", "language": "L",
		"voice_input_path": models.VoiceInputPathSkipCalibrate,
		"style_answers": map[string]interface{}{
			"preferred_words": "yaar",
			"avoid_words":     "delve",
			"anti_voice":      "news anchor",
		},
	}, h.SubmitProfile); w.Code != http.StatusInternalServerError {
		t.Fatal("expected persona error")
	}

	h = NewProfileHandler(&stubProfileService{}, &stubPromptService{}, &stubPersonaService{createResult: &models.PersonaProfile{CreatorID: 1}})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{
		"name": "A", "genre": "G", "language": "L",
		"voice_input_path": models.VoiceInputPathSkipCalibrate,
		"style_answers": map[string]interface{}{
			"preferred_words": "yaar",
			"avoid_words":     "delve",
			"anti_voice":      "news anchor",
		},
	}, h.SubmitProfile); w.Code != http.StatusOK {
		t.Fatal("expected success submit")
	}

	h = NewProfileHandler(&stubProfileService{getResult: profile}, &stubPromptService{}, &stubPersonaService{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": "bad", "topic": "x"}, h.GeneratePrompt); w.Code != http.StatusBadRequest {
		t.Fatal("expected bad prompt request")
	}

	h = NewProfileHandler(&stubProfileService{getErr: errors.New("missing")}, &stubPromptService{}, &stubPersonaService{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "x"}, h.GeneratePrompt); w.Code != http.StatusNotFound {
		t.Fatal("expected missing profile")
	}

	h = NewProfileHandler(&stubProfileService{getResult: profile}, &stubPromptService{err: errors.New("save fail")}, &stubPersonaService{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "x"}, h.GeneratePrompt); w.Code != http.StatusInternalServerError {
		t.Fatal("expected prompt save error")
	}

	h = NewProfileHandler(&stubProfileService{getResult: profile}, &stubPromptService{}, &stubPersonaService{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "x"}, h.GeneratePrompt); w.Code != http.StatusOK {
		t.Fatal("expected prompt success")
	}

	h = NewProfileHandler(&stubProfileService{getResult: profile}, &stubPromptService{}, &stubPersonaService{getErr: errors.New("persona fail")})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "x"}, h.GeneratePrompt); w.Code != http.StatusInternalServerError {
		t.Fatal("expected persona default error")
	}
}

func TestPersonaHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	run := func(fn func(c *gin.Context)) int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		fn(c)
		return w.Code
	}

	h := NewPersonaHandler(&stubPersonaService{getResult: &models.PersonaProfile{CreatorID: 1}})
	if code := run(func(c *gin.Context) {
		c.Params = gin.Params{{Key: "creator_id", Value: "abc"}}
		h.GetPersona(c)
	}); code != http.StatusBadRequest {
		t.Fatal("bad id")
	}

	if code := run(func(c *gin.Context) {
		c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
		h.GetPersona(c)
	}); code != http.StatusOK {
		t.Fatal("get persona")
	}

	h = NewPersonaHandler(&stubPersonaService{getErr: errors.New("missing")})
	if code := run(func(c *gin.Context) {
		c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
		h.GetPersona(c)
	}); code != http.StatusNotFound {
		t.Fatal("missing persona")
	}

	h = NewPersonaHandler(&stubPersonaService{updateResult: &models.PersonaProfile{CreatorID: 1}})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"voice_summary":"x"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
	h.UpdatePersona(c)
	if w.Code != http.StatusOK {
		t.Fatal("update persona")
	}

	h = NewPersonaHandler(&stubPersonaService{updateErr: errors.New("fail")})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"voice_summary":"x"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
	h.UpdatePersona(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatal("update persona error")
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
	h.UpdatePersona(c)
	if w.Code != http.StatusBadRequest {
		t.Fatal("update persona bad json")
	}

	h = NewPersonaHandler(&stubPersonaService{getResult: &models.PersonaProfile{CreatorID: 1}})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"writing_samples":["s"]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
	h.ReanalyzeVoice(c)
	if w.Code != http.StatusOK {
		t.Fatalf("reanalyze: %s", w.Body.String())
	}

	h = NewPersonaHandler(&stubPersonaService{reanalyzeErr: errors.New("fail")})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
	h.ReanalyzeVoice(c)
	if w.Code != http.StatusBadGateway {
		t.Fatal("reanalyze error")
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
	h.ReanalyzeVoice(c)
	if w.Code != http.StatusBadRequest {
		t.Fatal("reanalyze bad json with valid id")
	}
}

func TestOnboardingAndPromptHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oh := NewOnboardingHandler(&stubOnboardingService{questions: []models.OnboardingQuestion{{ID: "q1", Text: "t"}}})
	if w := performHandlerRequest(http.MethodGet, "/", nil, oh.GetQuestions); w.Code != http.StatusOK {
		t.Fatal("questions")
	}

	ph := NewPromptHandler(&stubPromptRepo{prompts: []models.Prompt{{ID: 1}}})
	if code := runHandler(func(c *gin.Context) {
		c.Params = gin.Params{{Key: "creator_id", Value: "bad"}}
		ph.GetPromptsByCreatorID(c)
	}); code != http.StatusBadRequest {
		t.Fatal("bad creator")
	}

	if code := runHandler(func(c *gin.Context) {
		c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
		ph.GetPromptsByCreatorID(c)
	}); code != http.StatusOK {
		t.Fatal("prompts ok")
	}

	ph = NewPromptHandler(&stubPromptRepo{err: errors.New("fail")})
	if code := runHandler(func(c *gin.Context) {
		c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
		ph.GetPromptsByCreatorID(c)
	}); code != http.StatusInternalServerError {
		t.Fatal("prompt repo error")
	}

	ab := NewPromptABHandler(&stubProfileService{getResult: &models.CreatorProfile{}}, &stubPersonaService{}, &stubPromptABService{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "t"}, ab.GeneratePromptAB); w.Code != http.StatusOK {
		t.Fatal("prompt ab")
	}

	ab = NewPromptABHandler(&stubProfileService{getErr: errors.New("x")}, &stubPersonaService{}, &stubPromptABService{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "t"}, ab.GeneratePromptAB); w.Code != http.StatusNotFound {
		t.Fatal("missing profile ab")
	}

	ab = NewPromptABHandler(&stubProfileService{getResult: &models.CreatorProfile{}}, &stubPersonaService{}, &stubPromptABService{storeErr: errors.New("fail")})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "t"}, ab.GeneratePromptAB); w.Code != http.StatusInternalServerError {
		t.Fatal("store ab error")
	}

	if w := performHandlerRequest(http.MethodPost, "/", map[string]string{"topic": "t"}, ab.GeneratePromptAB); w.Code != http.StatusBadRequest {
		t.Fatal("bad ab json")
	}
}

func TestScriptAndFeedbackHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sh := NewScriptHandler(&stubPromptRepo{prompts: []models.Prompt{{Topic: "t", Variant: "base", PromptText: "p", UserPrompt: "u"}}}, &stubScriptService{text: "script"}, &stubScriptRepo{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "t"}, sh.GenerateScript); w.Code != http.StatusOK {
		t.Fatal("generate script")
	}

	sh = NewScriptHandler(&stubPromptRepo{prompts: []models.Prompt{{Topic: "t", Variant: "base", PromptText: "full only"}}}, &stubScriptService{text: "script"}, &stubScriptRepo{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "t"}, sh.GenerateScript); w.Code != http.StatusOK {
		t.Fatal("generate script with prompt text fallback")
	}

	sh = NewScriptHandler(&stubPromptRepo{prompts: []models.Prompt{}}, &stubScriptService{}, &stubScriptRepo{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "missing"}, sh.GenerateScript); w.Code != http.StatusNotFound {
		t.Fatal("missing prompt")
	}

	sh = NewScriptHandler(&stubPromptRepo{err: errors.New("prompt err")}, &stubScriptService{}, &stubScriptRepo{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "t"}, sh.GenerateScript); w.Code != http.StatusInternalServerError {
		t.Fatal("prompt fetch error")
	}

	sh = NewScriptHandler(&stubPromptRepo{prompts: []models.Prompt{{Topic: "t", Variant: "base"}}}, &stubScriptService{err: errors.New("openai")}, &stubScriptRepo{})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "t"}, sh.GenerateScript); w.Code != http.StatusBadGateway {
		t.Fatal("openai error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
	sh = NewScriptHandler(&stubPromptRepo{}, &stubScriptService{}, &stubScriptRepo{scripts: []services.ScriptWithPrompt{{ScriptText: "s"}}})
	sh.GetScriptsByCreatorID(c)
	if w.Code != http.StatusOK {
		t.Fatal("list scripts")
	}

	fh := NewFeedbackHandler(&stubFeedbackService{feedback: &models.ScriptFeedback{ID: 1}, deltas: map[string]int{"humor": 5}})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"rating":"no","notes":"too formal"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "script_id", Value: "1"}}
	fh.SubmitFeedback(c)
	if w.Code != http.StatusOK {
		t.Fatal("feedback ok")
	}

	fh = NewFeedbackHandler(&stubFeedbackService{err: errors.New("fail")})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"rating":"no"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "script_id", Value: "1"}}
	fh.SubmitFeedback(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatal("feedback error")
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"rating":"no"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "script_id", Value: "bad"}}
	fh.SubmitFeedback(c)
	if w.Code != http.StatusBadRequest {
		t.Fatal("feedback bad script id")
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "script_id", Value: "1"}}
	fh.SubmitFeedback(c)
	if w.Code != http.StatusBadRequest {
		t.Fatal("feedback missing rating")
	}

	sh = NewScriptHandler(&stubPromptRepo{prompts: []models.Prompt{{Topic: "t", Variant: "base"}}}, &stubScriptService{text: "x"}, &stubScriptRepo{err: errors.New("save")})
	if w := performHandlerRequest(http.MethodPost, "/", map[string]interface{}{"creator_id": 1, "topic": "t"}, sh.GenerateScript); w.Code != http.StatusInternalServerError {
		t.Fatal("script save error")
	}

	sh = NewScriptHandler(&stubPromptRepo{}, &stubScriptService{}, &stubScriptRepo{err: errors.New("list")})
	if code := runHandler(func(c *gin.Context) {
		c.Params = gin.Params{{Key: "creator_id", Value: "1"}}
		sh.GetScriptsByCreatorID(c)
	}); code != http.StatusInternalServerError {
		t.Fatal("script list error")
	}

	if w := performHandlerRequest(http.MethodPost, "/", map[string]string{"topic": "t"}, sh.GenerateScript); w.Code != http.StatusBadRequest {
		t.Fatal("bad script request")
	}
}

func runHandler(fn func(c *gin.Context)) int {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	fn(c)
	return w.Code
}

func TestParseCreatorID(t *testing.T) {
	if _, err := parseCreatorID("abc"); err == nil {
		t.Fatal("expected parse error")
	}
	if id, err := parseCreatorID("12"); err != nil || id != 12 {
		t.Fatalf("expected id 12, got %d err=%v", id, err)
	}
}
