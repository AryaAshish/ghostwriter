package services

import (
	"math"
	"strings"
	"time"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

var functionWords = []string{
	"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for",
	"of", "with", "is", "are", "was", "were", "be", "been", "it", "this",
	"that", "you", "your", "i", "my", "we", "so", "just", "like", "really",
}

var fillerWords = []string{
	"basically", "literally", "actually", "like", "so", "okay", "ok",
	"yaar", "matlab", "dekho", "suno", "arre", "bhai", "right",
}

const minWordsForDerived = 300
const minWordsForCalibrated = 150

// ExtractFingerprint builds a voice fingerprint from sample texts.
func ExtractFingerprint(texts []string) models.VoiceFingerprint {
	combined := strings.Join(texts, "\n")
	tokens := tokenize(combined)
	totalWords := len(tokens)
	if totalWords == 0 {
		return models.VoiceFingerprint{}
	}

	sentences := splitSentences(combined)
	lengths := make([]float64, 0, len(sentences))
	for _, s := range sentences {
		lengths = append(lengths, float64(countWords(s)))
	}
	avgLen, stdLen := meanStd(lengths)

	fillerCount := 0
	hinglishCount := 0
	for _, t := range tokens {
		if isFiller(t) {
			fillerCount++
		}
		if isHinglishToken(t) {
			hinglishCount++
		}
	}

	exclam := strings.Count(combined, "!")
	questions := strings.Count(combined, "?")

	hook := classifyHook(firstLine(combined))

	return models.VoiceFingerprint{
		FunctionWordFreq:  functionWordFreq(tokens),
		AvgSentenceLength: avgLen,
		SentenceLengthStd: stdLen,
		FillerDensity:     float64(fillerCount) / float64(totalWords),
		HinglishRatio:     float64(hinglishCount) / float64(totalWords),
		HookPattern:       hook,
		ExclamationRate:   float64(exclam) / float64(len(sentences)),
		QuestionRate:      float64(questions) / float64(len(sentences)),
		TotalWords:        totalWords,
	}
}

// ShiftScore returns 0–100 similarity (100 = identical style). Lower distance = higher score.
func ShiftScore(output string, fp models.VoiceFingerprint) int {
	if fp.TotalWords == 0 {
		return 0
	}
	outFP := ExtractFingerprint([]string{output})
	if outFP.TotalWords == 0 {
		return 0
	}

	dist := 0.0
	dist += math.Abs(outFP.AvgSentenceLength-fp.AvgSentenceLength) / math.Max(fp.AvgSentenceLength, 1)
	dist += math.Abs(outFP.FillerDensity-fp.FillerDensity) * 10
	dist += math.Abs(outFP.HinglishRatio-fp.HinglishRatio) * 10
	if outFP.HookPattern != fp.HookPattern {
		dist += 0.5
	}

	score := 100 - int(dist*20)
	if score < 0 {
		return 0
	}
	return score
}

// ComputeVoiceConfidence estimates how reliable the voice profile is.
func ComputeVoiceConfidence(totalWords, sampleCount, feedbackCount int, mode string) int {
	score := 0
	switch {
	case totalWords >= minWordsForDerived:
		score = 75
	case totalWords >= minWordsForCalibrated:
		score = 55
	case totalWords > 0:
		score = 35
	default:
		score = 15
	}
	score += sampleCount * 5
	score += feedbackCount * 8
	if mode == models.PersonaModeDerived {
		score += 10
	}
	if score > 100 {
		return 100
	}
	return score
}

// ResolvePersonaMode picks initial mode from path and word count.
func ResolvePersonaMode(path string, totalWords int) string {
	switch path {
	case models.VoiceInputPathSkipCalibrate:
		return models.PersonaModeDeclared
	case models.VoiceInputPathPasteScripts:
		if totalWords >= minWordsForDerived {
			return models.PersonaModeDerived
		}
		if totalWords >= minWordsForCalibrated {
			return models.PersonaModeCalibrated
		}
		return models.PersonaModeDeclared
	case models.VoiceInputPathGuidedWrite:
		if totalWords >= minWordsForDerived {
			return models.PersonaModeDerived
		}
		if totalWords >= minWordsForCalibrated {
			return models.PersonaModeCalibrated
		}
		return models.PersonaModeDeclared
	default:
		if totalWords >= minWordsForDerived {
			return models.PersonaModeDerived
		}
		if totalWords >= minWordsForCalibrated {
			return models.PersonaModeCalibrated
		}
		return models.PersonaModeDeclared
	}
}

// PromotePersonaMode upgrades mode after calibration.
func PromotePersonaMode(current string, totalWords int) string {
	if totalWords >= minWordsForDerived {
		return models.PersonaModeDerived
	}
	if totalWords >= minWordsForCalibrated && current == models.PersonaModeDeclared {
		return models.PersonaModeCalibrated
	}
	return current
}

// LexicalHintsFromFingerprint derives lexical profile hints from a fingerprint.
func LexicalHintsFromFingerprint(texts []string, fp models.VoiceFingerprint) models.LexicalProfile {
	profile := models.LexicalProfile{FillerFrequency: "moderate"}
	combined := strings.ToLower(strings.Join(texts, "\n"))

	foundFillers := make([]string, 0)
	for _, f := range fillerWords {
		if strings.Contains(combined, f) {
			foundFillers = append(foundFillers, f)
		}
	}
	profile.FillerWords = uniqueStrings(foundFillers)

	first := firstLine(strings.Join(texts, "\n"))
	if first != "" {
		profile.SentenceStarters = []string{firstLineStarter(first)}
	}
	if fp.HinglishRatio > 0.08 {
		profile.SlangRegister = "casual_hinglish"
	}
	if fp.FillerDensity > 0.06 {
		profile.FillerFrequency = "high"
	} else if fp.FillerDensity < 0.02 {
		profile.FillerFrequency = "low"
	}
	return profile
}

func functionWordFreq(tokens []string) map[string]float64 {
	if len(tokens) == 0 {
		return map[string]float64{}
	}
	counts := map[string]int{}
	for _, t := range tokens {
		counts[t]++
	}
	out := map[string]float64{}
	for _, fw := range functionWords {
		out[fw] = float64(counts[fw]) / float64(len(tokens))
	}
	return out
}

func isFiller(token string) bool {
	for _, f := range fillerWords {
		if token == f {
			return true
		}
	}
	return false
}

func meanStd(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	if len(values) == 1 {
		return mean, 0
	}
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	return mean, math.Sqrt(variance)
}

func firstLine(text string) string {
	lines := strings.Split(text, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			return l
		}
	}
	return strings.TrimSpace(text)
}

func firstLineStarter(line string) string {
	tokens := tokenize(line)
	if len(tokens) == 0 {
		return line
	}
	if len(tokens) > 4 {
		tokens = tokens[:4]
	}
	return strings.Join(tokens, " ")
}

func classifyHook(line string) string {
	lower := strings.ToLower(strings.TrimSpace(line))
	if lower == "" {
		return "unknown"
	}
	if strings.HasSuffix(lower, "?") || strings.HasPrefix(lower, "what") || strings.HasPrefix(lower, "why") || strings.HasPrefix(lower, "how") {
		return "question"
	}
	if strings.HasPrefix(lower, "okay so") || strings.HasPrefix(lower, "suno") || strings.HasPrefix(lower, "dekho") || strings.HasPrefix(lower, "real talk") {
		return "punch"
	}
	if strings.Contains(lower, "story") || strings.HasPrefix(lower, "once") || strings.HasPrefix(lower, "when i") {
		return "story"
	}
	if countWords(line) >= 18 {
		return "slow_setup"
	}
	return "bold_claim"
}

// NormalizeSamples merges legacy string samples with structured samples.
func NormalizeSamples(legacy []string, structured []models.WritingSample) []models.WritingSample {
	out := make([]models.WritingSample, 0, len(legacy)+len(structured))
	for _, s := range structured {
		if strings.TrimSpace(s.Text) != "" {
			out = append(out, s)
		}
	}
	seen := map[string]bool{}
	for _, s := range out {
		seen[s.Text] = true
	}
	for _, text := range legacy {
		text = strings.TrimSpace(text)
		if text == "" || seen[text] {
			continue
		}
		out = append(out, models.WritingSample{
			Text:      text,
			Source:    models.SampleSourcePastScript,
			CreatedAt: time.Now(),
		})
	}
	return out
}

func sampleTexts(samples []models.WritingSample) []string {
	out := make([]string, 0, len(samples))
	for _, s := range samples {
		if strings.TrimSpace(s.Text) != "" {
			out = append(out, s.Text)
		}
	}
	return out
}

func totalSampleWords(samples []models.WritingSample) int {
	total := 0
	for _, s := range samples {
		total += countWords(s.Text)
	}
	return total
}

func BuildSamplesFromSubmit(req models.SubmitProfileRequest) []models.WritingSample {
	path := req.VoiceInputPath
	if path == "" {
		path = models.VoiceInputPathPasteScripts
	}

	switch path {
	case models.VoiceInputPathGuidedWrite:
		keys := []string{"guided_hook", "guided_hot_take", "guided_mini_story"}
		out := make([]models.WritingSample, 0, len(keys))
		for _, k := range keys {
			if text, ok := req.GuidedWrites[k]; ok && strings.TrimSpace(text) != "" {
				out = append(out, models.WritingSample{
					Text:      strings.TrimSpace(text),
					Source:    models.SampleSourceGuidedWrite,
					CreatedAt: time.Now(),
				})
			}
		}
		return out
	case models.VoiceInputPathSkipCalibrate:
		return nil
	case models.VoiceInputPathImportInstagram:
		return BuildSamplesFromInstagramReels(req.InstagramReels)
	default:
		out := make([]models.WritingSample, 0, len(req.WritingSamples))
		for _, text := range req.WritingSamples {
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			out = append(out, models.WritingSample{
				Text:      text,
				Source:    models.SampleSourcePastScript,
				CreatedAt: time.Now(),
			})
		}
		return out
	}
}
