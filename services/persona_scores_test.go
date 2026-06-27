package services

import (
	"strings"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

func TestPersonaScoresHelpers(t *testing.T) {
	base := defaultScores()
	merged := mergeScoresWeighted(base, models.PersonaScores{Humor: 100}, 60, 40)
	if merged.Humor <= base.Humor {
		t.Fatalf("expected merged humor to increase")
	}

	clamped := clampScores(models.PersonaScores{Formality: -10, Humor: 150})
	if clamped.Formality != 0 || clamped.Humor != 100 {
		t.Fatalf("clamp failed: %+v", clamped)
	}

	lexical := mergeLexicalProfiles(
		models.LexicalProfile{PreferredWords: []string{"yaar"}},
		models.LexicalProfile{PreferredWords: []string{"yaar", "matlab"}, SlangRegister: "casual"},
	)
	if len(lexical.PreferredWords) != 2 || lexical.SlangRegister != "casual" {
		t.Fatalf("unexpected lexical merge: %+v", lexical)
	}

	words := uniqueStrings([]string{"Yaar", "yaar", " ", "matlab"})
	if len(words) != 2 {
		t.Fatalf("expected 2 unique words, got %v", words)
	}

	split := splitWords("yaar, matlab; bro")
	if len(split) != 3 {
		t.Fatalf("expected 3 split words, got %v", split)
	}
	if trimSpace("  hi  ") != "hi" || trimSpace("") != "" {
		t.Fatal("trimSpace failed")
	}
	if len(toStringSlice([]string{"a", "b"})) != 2 {
		t.Fatal("toStringSlice string slice failed")
	}
}

func TestMergeLegacyProfileHints(t *testing.T) {
	profile := testProfile()
	scores := mergeLegacyProfileHints(defaultScores(), profile)
	if scores.Humor <= 50 || scores.HinglishMix <= 50 {
		t.Fatalf("expected legacy hints to adjust scores: %+v", scores)
	}
	if mergeLegacyProfileHints(defaultScores(), nil).Humor != 50 {
		t.Fatalf("nil profile should return unchanged defaults")
	}
}

func TestBandLabelAndVariantInstructions(t *testing.T) {
	if bandLabel(10) != "low" || bandLabel(50) != "medium" || bandLabel(90) != "high" {
		t.Fatal("band labels incorrect")
	}
	for _, variant := range []string{"A", "B", "C", "base", ""} {
		if variantStructureInstruction(variant) == "" {
			t.Fatalf("empty structure for variant %q", variant)
		}
	}
}

func TestBuildScoreInstructionBlock(t *testing.T) {
	block := buildScoreInstructionBlock(models.PersonaScores{
		Formality: 10, Humor: 50, Energy: 90, Brevity: 10,
		Storytelling: 50, Directness: 90, EmotionalWarmth: 10, HinglishMix: 50,
	})
	if !strings.Contains(block, "Style instructions:") {
		t.Fatal("missing style instructions")
	}
	for _, fn := range []func() [3]string{
		formalityInstructions, humorInstructions, energyInstructions, brevityInstructions,
		storytellingInstructions, directnessInstructions, warmthInstructions, hinglishInstructions,
	} {
		bands := fn()
		if bands[0] == "" || bands[1] == "" || bands[2] == "" {
			t.Fatal("empty instruction band")
		}
	}
}

func TestInferDeltasFromFeedbackExtended(t *testing.T) {
	cases := map[string]string{
		"more formal":   "formality",
		"not funny":     "humor",
		"too long":      "brevity",
		"more energy":   "energy",
		"too soft":      "directness",
		"more personal": "emotional_warmth",
		"more story":    "storytelling",
		"less hindi":    "hinglish_mix",
	}
	for phrase, dim := range cases {
		deltas := inferDeltasFromFeedback("", phrase)
		if deltas[dim] == 0 {
			t.Fatalf("expected delta for %q on %s", phrase, dim)
		}
	}

	noRating := inferDeltasFromFeedback("no", "")
	if noRating["formality"] == 0 || noRating["humor"] == 0 {
		t.Fatalf("expected default no-rating deltas: %+v", noRating)
	}
}

func TestExtractJSON(t *testing.T) {
	raw := "```json\n{\"voice_summary\":\"test\"}\n```"
	out := extractJSON(raw)
	if !strings.Contains(out, "voice_summary") {
		t.Fatalf("extractJSON failed: %s", out)
	}
}
