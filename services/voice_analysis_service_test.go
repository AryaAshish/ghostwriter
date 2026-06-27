package services

import (
	"strings"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

func TestVoiceAnalysisService(t *testing.T) {
	profile := testProfile()
	baseline := defaultScores()

	t.Run("missing client", func(t *testing.T) {
		svc := NewOpenAIVoiceAnalysisService(nil)
		if _, err := svc.AnalyzeVoice([]string{"sample"}, baseline, profile); err == nil {
			t.Fatal("expected missing client error")
		}
	})

	t.Run("missing samples", func(t *testing.T) {
		svc := NewOpenAIVoiceAnalysisService(&OpenAIClient{APIKey: "k"})
		if _, err := svc.AnalyzeVoice(nil, baseline, profile); err == nil {
			t.Fatal("expected missing samples error")
		}
	})

	t.Run("success", func(t *testing.T) {
		payload := `{
			"suggested_scores": {
				"formality": 20, "humor": 80, "energy": 70, "brevity": 60,
				"storytelling": 50, "directness": 40, "emotional_warmth": 55, "hinglish_mix": 65
			},
			"lexical_profile": {
				"signature_phrases": ["matlab"],
				"filler_words": ["so"],
				"sentence_starters": ["Dekho"],
				"preferred_words": ["yaar"],
				"avoid_words": ["delve"],
				"slang_register": "casual_hinglish",
				"filler_frequency": "moderate"
			},
			"voice_summary": "Casual Hinglish humor"
		}`
		server := newMockOpenAIServer(t, payload, 0)
		defer server.Close()
		client := newOpenAIClientForTest(t, server)
		svc := NewOpenAIVoiceAnalysisService(client)

		result, err := svc.AnalyzeVoice([]string{"Dekho yaar matlab"}, baseline, profile)
		if err != nil {
			t.Fatal(err)
		}
		if result.VoiceSummary != "Casual Hinglish humor" || result.SuggestedScores.Humor != 80 {
			t.Fatalf("unexpected result: %+v", result)
		}
		if result.LexicalProfile.FillerFrequency != "moderate" {
			t.Fatal("expected filler frequency")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := newMockOpenAIServer(t, "not json", 0)
		defer server.Close()
		client := newOpenAIClientForTest(t, server)
		svc := NewOpenAIVoiceAnalysisService(client)
		if _, err := svc.AnalyzeVoice([]string{"sample"}, baseline, profile); err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("default filler frequency", func(t *testing.T) {
		payload := `{
			"suggested_scores": {"formality":50,"humor":50,"energy":50,"brevity":50,"storytelling":50,"directness":50,"emotional_warmth":50,"hinglish_mix":50},
			"lexical_profile": {"signature_phrases":[]},
			"voice_summary": "neutral"
		}`
		server := newMockOpenAIServer(t, payload, 0)
		defer server.Close()
		svc := NewOpenAIVoiceAnalysisService(newOpenAIClientForTest(t, server))
		result, err := svc.AnalyzeVoice([]string{"sample"}, baseline, profile)
		if err != nil || result.LexicalProfile.FillerFrequency != "moderate" {
			t.Fatalf("expected default filler frequency, got %+v err=%v", result, err)
		}
	})

	t.Run("api error", func(t *testing.T) {
		server := newMockOpenAIServer(t, "", 400)
		defer server.Close()
		svc := NewOpenAIVoiceAnalysisService(newOpenAIClientForTest(t, server))
		if _, err := svc.AnalyzeVoice([]string{"sample"}, baseline, profile); err == nil {
			t.Fatal("expected api error")
		}
	})
}

func TestOnboardingServiceExtended(t *testing.T) {
	service := NewDefaultOnboardingService()
	questions := service.GetQuestions()
	if len(questions) < 8 {
		t.Fatalf("expected at least 8 questions, got %d", len(questions))
	}

	empty := service.MapAnswersToBaseline(nil)
	if empty.Humor != 50 {
		t.Fatalf("expected default scores, got %+v", empty)
	}

	multi := service.MapAnswersToBaseline(map[string]interface{}{
		"hook_style": []interface{}{"punch", "question"},
	})
	if multi.Brevity <= 50 {
		t.Fatalf("expected multi-select to affect brevity: %+v", multi)
	}

	freeText := service.ExtractLexicalHints(map[string]interface{}{
		"preferred_words": "yaar",
	})
	if len(freeText.PreferredWords) != 1 {
		t.Fatal("expected preferred words from free text")
	}

	unknown := service.MapAnswersToBaseline(map[string]interface{}{
		"humor_style": "unknown_option",
	})
	if unknown.Humor != 50 {
		t.Fatalf("unknown option should not change defaults unexpectedly: %+v", unknown)
	}

	scaleOnly := service.MapAnswersToBaseline(map[string]interface{}{"content_energy": 1})
	if scaleOnly.Energy != 0 {
		t.Fatalf("expected energy 0 for scale 1, got %d", scaleOnly.Energy)
	}
	if !strings.Contains(onboardingQuestionCatalog()[0].Text, "energetic") {
		t.Fatal("catalog question text changed unexpectedly")
	}
}

func TestReadWriteDimensionDefaults(t *testing.T) {
	scores := models.PersonaScores{}
	if readDimension(scores, "unknown") != defaultScore {
		t.Fatal("unknown dimension should default")
	}
	scores = writeDimension(scores, "unknown", 99)
	if scores.Humor != 0 {
		t.Fatal("unknown dimension write should no-op")
	}
}
