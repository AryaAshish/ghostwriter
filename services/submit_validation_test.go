package services

import (
	"strings"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

func TestValidateSubmitProfilePaths(t *testing.T) {
	long := strings.Repeat("word ", 60)

	t.Run("paste invalid", func(t *testing.T) {
		r := ValidateSubmitProfile(models.SubmitProfileRequest{
			VoiceInputPath: models.VoiceInputPathPasteScripts,
			WritingSamples: []string{"short"},
		})
		if r.OK {
			t.Fatal("expected paste validation failure")
		}
	})

	t.Run("paste warn", func(t *testing.T) {
		r := ValidateSubmitProfile(models.SubmitProfileRequest{
			VoiceInputPath: models.VoiceInputPathPasteScripts,
			WritingSamples: []string{long, long},
		})
		if !r.OK || r.Warning == "" {
			t.Fatal("expected warning for low volume with 2 samples")
		}
	})

	t.Run("guided invalid", func(t *testing.T) {
		r := ValidateSubmitProfile(models.SubmitProfileRequest{
			VoiceInputPath: models.VoiceInputPathGuidedWrite,
			GuidedWrites:   map[string]string{"guided_hook": "too short"},
			StyleAnswers: map[string]interface{}{
				"preferred_words": "yaar",
				"avoid_words":     "delve",
			},
		})
		if r.OK {
			t.Fatal("expected guided validation failure")
		}
	})

	t.Run("skip ok", func(t *testing.T) {
		r := ValidateSubmitProfile(models.SubmitProfileRequest{
			VoiceInputPath: models.VoiceInputPathSkipCalibrate,
			StyleAnswers: map[string]interface{}{
				"preferred_words": "yaar",
				"avoid_words":     "delve",
				"anti_voice":      "news anchor",
			},
		})
		if !r.OK {
			t.Fatalf("expected skip ok, got %s", r.Message)
		}
	})
}

func TestGetQuestionsForPath(t *testing.T) {
	svc := NewDefaultOnboardingService()
	paste := svc.GetQuestionsForPath(models.VoiceInputPathPasteScripts, "comedy")
	guided := svc.GetQuestionsForPath(models.VoiceInputPathGuidedWrite, "finance")
	skip := svc.GetQuestionsForPath(models.VoiceInputPathSkipCalibrate, "")
	if len(guided) <= len(paste) {
		t.Fatal("guided should have extra questions")
	}
	if len(skip) <= len(paste) {
		t.Fatal("skip should have extra questions")
	}
	found := false
	for _, q := range guided {
		if q.Type == "guided_write" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected guided_write question")
	}
}

func TestMapAnswersComparativeChoice(t *testing.T) {
	svc := NewDefaultOnboardingService()
	scores := svc.MapAnswersToBaseline(map[string]interface{}{
		"comparative_hook": "punch",
	})
	if scores.Brevity <= defaultScore {
		t.Fatalf("expected brevity bump from comparative choice, got %d", scores.Brevity)
	}
}

func TestToIntDefaultBranch(t *testing.T) {
	if toInt("bad") != 0 {
		t.Fatal("expected 0 for bad type")
	}
	if toStringSlice(123) != nil {
		t.Fatal("expected nil for bad slice type")
	}
}
