package services

import (
	"fmt"
	"strings"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

const guidedWriteMinWords = 25

var guidedWriteLabels = map[string]string{
	"guided_hook":       "Video opener exercise",
	"guided_hot_take":   "Hot take exercise",
	"guided_mini_story": "Personal story exercise",
}

// SubmitValidationResult holds validation outcome for profile submit.
type SubmitValidationResult struct {
	OK      bool
	Message string
	Warning string
}

// ValidateSubmitProfile checks path-specific requirements.
func ValidateSubmitProfile(req models.SubmitProfileRequest) SubmitValidationResult {
	path := req.VoiceInputPath
	if path == "" {
		path = models.VoiceInputPathPasteScripts
	}

	switch path {
	case models.VoiceInputPathPasteScripts:
		wordCount := 0
		sampleCount := 0
		for _, s := range req.WritingSamples {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			sampleCount++
			wordCount += countWords(s)
		}
		if sampleCount < 2 && wordCount < 300 {
			return SubmitValidationResult{
				OK:      false,
				Message: "paste_scripts path requires at least 2 samples or 300 total words",
			}
		}
		if wordCount < 300 {
			return SubmitValidationResult{OK: true, Warning: "Low sample volume; voice confidence will be limited until you add more text."}
		}
		return SubmitValidationResult{OK: true}

	case models.VoiceInputPathGuidedWrite:
		required := []string{"guided_hook", "guided_hot_take", "guided_mini_story"}
		for _, key := range required {
			text, ok := req.GuidedWrites[key]
			words := countWords(text)
			if !ok || words < guidedWriteMinWords {
				label := guidedWriteLabels[key]
				if label == "" {
					label = key
				}
				return SubmitValidationResult{
					OK:      false,
					Message: fmt.Sprintf("%s needs at least %d words (you have %d)", label, guidedWriteMinWords, words),
				}
			}
		}
		if len(splitWords(toString(req.StyleAnswers["preferred_words"]))) == 0 {
			return SubmitValidationResult{OK: false, Message: "guided_write path requires preferred_words"}
		}
		if len(splitWords(toString(req.StyleAnswers["avoid_words"]))) == 0 {
			return SubmitValidationResult{OK: false, Message: "guided_write path requires avoid_words"}
		}
		return SubmitValidationResult{OK: true}

	case models.VoiceInputPathImportInstagram:
		selected := 0
		wordCount := 0
		for _, reel := range req.InstagramReels {
			if !reel.Selected {
				continue
			}
			text := strings.TrimSpace(reel.Text)
			if text == "" {
				text = strings.TrimSpace(reel.Caption)
			}
			if text == "" {
				text = strings.TrimSpace(reel.Transcript)
			}
			if text == "" {
				continue
			}
			selected++
			wordCount += countWords(text)
		}
		if selected < 2 {
			return SubmitValidationResult{
				OK:      false,
				Message: "import_instagram requires at least 2 selected reels with caption or transcript text",
			}
		}
		if wordCount < minWordsForCalibrated {
			return SubmitValidationResult{
				OK:      true,
				Warning: "Low reel text volume; voice confidence will improve after more reels or calibration edits.",
			}
		}
		return SubmitValidationResult{OK: true}

	case models.VoiceInputPathSkipCalibrate:
		if len(splitWords(toString(req.StyleAnswers["preferred_words"]))) == 0 {
			return SubmitValidationResult{OK: false, Message: "skip_calibrate path requires preferred_words"}
		}
		if len(splitWords(toString(req.StyleAnswers["avoid_words"]))) == 0 {
			return SubmitValidationResult{OK: false, Message: "skip_calibrate path requires avoid_words"}
		}
		if strings.TrimSpace(toString(req.StyleAnswers["anti_voice"])) == "" {
			return SubmitValidationResult{OK: false, Message: "skip_calibrate path requires anti_voice description"}
		}
		return SubmitValidationResult{OK: true}

	default:
		return SubmitValidationResult{OK: true}
	}
}
