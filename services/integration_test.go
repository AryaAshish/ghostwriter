package services

import (
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

func TestProfileService(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGormProfileService(db)
	profile := testProfile()

	id, err := svc.CreateProfile(profile)
	if err != nil || id == 0 {
		t.Fatalf("create profile failed: id=%d err=%v", id, err)
	}

	got, err := svc.GetProfileByID(id)
	if err != nil || got.Name != "Amit" {
		t.Fatalf("get profile failed: %+v err=%v", got, err)
	}

	if _, err := svc.GetProfileByID(9999); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestPromptAndScriptRepositories(t *testing.T) {
	db := setupTestDB(t)
	profile, persona := seedCreator(t, db)
	_ = persona

	promptRepo := NewGormPromptRepository(db)
	prompt := &models.Prompt{
		CreatorID:    profile.ID,
		Topic:        "viral",
		Variant:      "A",
		PromptText:   "full",
		SystemPrompt: "system",
		UserPrompt:   "user",
	}
	if err := promptRepo.SavePrompt(prompt); err != nil {
		t.Fatal(err)
	}
	prompts, err := promptRepo.GetPromptsByCreatorID(profile.ID)
	if err != nil || len(prompts) != 1 {
		t.Fatalf("get prompts failed: len=%d err=%v", len(prompts), err)
	}

	scriptRepo := NewGormScriptRepository(db)
	script := &models.Script{
		CreatorID:  profile.ID,
		PromptID:   prompt.ID,
		ScriptText: "script body",
		Source:     "gpt-4",
	}
	if err := scriptRepo.SaveScript(script); err != nil {
		t.Fatal(err)
	}

	gotScript, err := scriptRepo.GetScriptByID(script.ID)
	if err != nil || gotScript.ScriptText != "script body" {
		t.Fatalf("get script by id failed: %+v err=%v", gotScript, err)
	}

	joined, err := scriptRepo.GetScriptsByCreatorIDWithPrompt(profile.ID)
	if err != nil || len(joined) != 1 || joined[0].Variant != "A" {
		t.Fatalf("joined scripts failed: %+v err=%v", joined, err)
	}

	if _, err := scriptRepo.GetScriptByID(999); err == nil {
		t.Fatal("expected missing script error")
	}
}

func TestPromptServices(t *testing.T) {
	db := setupTestDB(t)
	profile, persona := seedCreator(t, db)
	persona.WritingSamples = []string{"sample line"}

	personaSvc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	promptRepo := NewGormPromptRepository(db)
	promptSvc := NewDefaultPromptService(promptRepo, personaSvc)
	abSvc := NewDefaultPromptABService(personaSvc, promptRepo)

	summary := promptSvc.GeneratePersonaSummary(profile, persona)
	if summary == "" {
		t.Fatal("expected persona summary")
	}

	ctx, err := promptSvc.GeneratePrompt(profile, persona, "How to go viral")
	if err != nil || ctx.FullPromptText == "" {
		t.Fatalf("generate prompt failed: %+v err=%v", ctx, err)
	}

	variants, personaSummary := abSvc.GeneratePromptVariants(profile, persona, "topic")
	if len(variants) != 3 || personaSummary == "" {
		t.Fatalf("expected 3 variants, got %d", len(variants))
	}
	if err := abSvc.StorePromptVariants(variants, profile, "topic"); err != nil {
		t.Fatal(err)
	}

	prompts, _ := promptRepo.GetPromptsByCreatorID(profile.ID)
	if len(prompts) < 4 {
		t.Fatalf("expected base + 3 variants saved, got %d", len(prompts))
	}

	nilRepoAB := NewDefaultPromptABService(personaSvc, nil)
	if err := nilRepoAB.StorePromptVariants(variants, profile, "topic2"); err != nil {
		t.Fatal("nil repo should no-op")
	}
}

func TestScriptService(t *testing.T) {
	server := newMockOpenAIServer(t, "final script", 0)
	defer server.Close()
	client := newOpenAIClientForTest(t, server)
	svc := NewOpenAIScriptService(client)

	out, err := svc.GenerateScriptFromPrompt("system rules", "write about cats")
	if err != nil || out != "final script" {
		t.Fatalf("script generation failed: %q err=%v", out, err)
	}

	out2, err := svc.GenerateScriptFromPrompt("", "only user prompt")
	if err != nil || out2 != "final script" {
		t.Fatalf("user-only generation failed: %q err=%v", out2, err)
	}

	badSvc := NewOpenAIScriptService(nil)
	if _, err := badSvc.GenerateScriptFromPrompt("s", "u"); err == nil {
		t.Fatal("expected missing client error")
	}
}

func TestFeedbackService(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	personaSvc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	promptRepo := NewGormPromptRepository(db)
	scriptRepo := NewGormScriptRepository(db)
	feedbackSvc := NewGormFeedbackService(db, personaSvc, scriptRepo)

	prompt := &models.Prompt{CreatorID: profile.ID, Topic: "t", Variant: "base", PromptText: "p"}
	if err := promptRepo.SavePrompt(prompt); err != nil {
		t.Fatal(err)
	}
	script := &models.Script{CreatorID: profile.ID, PromptID: prompt.ID, ScriptText: "body", Source: "gpt-4"}
	if err := scriptRepo.SaveScript(script); err != nil {
		t.Fatal(err)
	}

	feedback, err := feedbackSvc.SubmitFeedback(script.ID, models.ScriptFeedbackRequest{
		Rating: "not_quite",
		Notes:  "too formal",
	})
	if err != nil || feedback.Feedback.ScriptID != script.ID || len(feedback.Deltas) == 0 {
		t.Fatalf("submit feedback failed: %+v err=%v", feedback, err)
	}

	if _, err := feedbackSvc.SubmitFeedback(999, models.ScriptFeedbackRequest{Rating: "no"}); err == nil {
		t.Fatal("expected missing script error")
	}
}
