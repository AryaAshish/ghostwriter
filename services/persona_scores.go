package services

import "github.com/ashisharyan/ghostwriter-prompt-engine/models"

const defaultScore = 50

func clampScore(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func clampScores(s models.PersonaScores) models.PersonaScores {
	return models.PersonaScores{
		Formality:       clampScore(s.Formality),
		Humor:           clampScore(s.Humor),
		Energy:          clampScore(s.Energy),
		Brevity:         clampScore(s.Brevity),
		Storytelling:    clampScore(s.Storytelling),
		Directness:      clampScore(s.Directness),
		EmotionalWarmth: clampScore(s.EmotionalWarmth),
		HinglishMix:     clampScore(s.HinglishMix),
	}
}

func defaultScores() models.PersonaScores {
	return models.PersonaScores{
		Formality:       defaultScore,
		Humor:           defaultScore,
		Energy:          defaultScore,
		Brevity:         defaultScore,
		Storytelling:    defaultScore,
		Directness:      defaultScore,
		EmotionalWarmth: defaultScore,
		HinglishMix:     defaultScore,
	}
}

func mergeScoresWeighted(baseline, suggested models.PersonaScores, baselineWeight, suggestedWeight int) models.PersonaScores {
	total := baselineWeight + suggestedWeight
	return clampScores(models.PersonaScores{
		Formality:       (baseline.Formality*baselineWeight + suggested.Formality*suggestedWeight) / total,
		Humor:           (baseline.Humor*baselineWeight + suggested.Humor*suggestedWeight) / total,
		Energy:          (baseline.Energy*baselineWeight + suggested.Energy*suggestedWeight) / total,
		Brevity:         (baseline.Brevity*baselineWeight + suggested.Brevity*suggestedWeight) / total,
		Storytelling:    (baseline.Storytelling*baselineWeight + suggested.Storytelling*suggestedWeight) / total,
		Directness:      (baseline.Directness*baselineWeight + suggested.Directness*suggestedWeight) / total,
		EmotionalWarmth: (baseline.EmotionalWarmth*baselineWeight + suggested.EmotionalWarmth*suggestedWeight) / total,
		HinglishMix:     (baseline.HinglishMix*baselineWeight + suggested.HinglishMix*suggestedWeight) / total,
	})
}

func applyScoreDeltas(scores models.PersonaScores, deltas map[string]int) models.PersonaScores {
	result := scores
	for dim, delta := range deltas {
		switch dim {
		case "formality":
			result.Formality += delta
		case "humor":
			result.Humor += delta
		case "energy":
			result.Energy += delta
		case "brevity":
			result.Brevity += delta
		case "storytelling":
			result.Storytelling += delta
		case "directness":
			result.Directness += delta
		case "emotional_warmth":
			result.EmotionalWarmth += delta
		case "hinglish_mix":
			result.HinglishMix += delta
		}
	}
	return clampScores(result)
}

func scoresFromAdjustments(adjustments map[string]int) models.PersonaScores {
	base := defaultScores()
	return applyScoreDeltas(base, adjustments)
}

func addDeltasToScores(base models.PersonaScores, adjustments map[string]int) models.PersonaScores {
	return applyScoreDeltas(base, adjustments)
}

func mergeLexicalProfiles(base, extra models.LexicalProfile) models.LexicalProfile {
	return models.LexicalProfile{
		SignaturePhrases: uniqueStrings(append(base.SignaturePhrases, extra.SignaturePhrases...)),
		FillerWords:      uniqueStrings(append(base.FillerWords, extra.FillerWords...)),
		SentenceStarters: uniqueStrings(append(base.SentenceStarters, extra.SentenceStarters...)),
		PreferredWords:   uniqueStrings(append(base.PreferredWords, extra.PreferredWords...)),
		AvoidWords:       uniqueStrings(append(base.AvoidWords, extra.AvoidWords...)),
		SlangRegister:    pickNonEmpty(extra.SlangRegister, base.SlangRegister),
		FillerFrequency:  pickNonEmpty(extra.FillerFrequency, base.FillerFrequency),
	}
}

func pickNonEmpty(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = trimSpace(item)
		if item == "" {
			continue
		}
		key := lowerString(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

func lowerString(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func splitWords(input string) []string {
	if input == "" {
		return nil
	}
	parts := make([]string, 0)
	current := ""
	for _, r := range input {
		if r == ',' || r == ';' || r == '\n' {
			if w := trimSpace(current); w != "" {
				parts = append(parts, w)
			}
			current = ""
			continue
		}
		current += string(r)
	}
	if w := trimSpace(current); w != "" {
		parts = append(parts, w)
	}
	return parts
}
