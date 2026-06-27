package services

import "strings"

// NicheVoiceRules returns genre-specific voice scaffolding for declared/cold-start personas.
func NicheVoiceRules(genre string) string {
	switch strings.ToLower(strings.TrimSpace(genre)) {
	case "comedy", "entertainment", "meme":
		return strings.Join([]string{
			"Niche voice pack (comedy):",
			"- Use playful exaggeration and punchy one-liners.",
			"- Break the fourth wall occasionally; react to absurdity.",
			"- Keep setups short; land jokes fast.",
			"- Avoid corporate or news-anchor tone.",
		}, "\n")
	case "finance", "business", "money", "investing":
		return strings.Join([]string{
			"Niche voice pack (finance):",
			"- Lead with a clear takeaway; use concrete numbers when possible.",
			"- Sound confident but not preachy; explain jargon in plain words.",
			"- Use cautionary examples; avoid hype language.",
			"- No get-rich-quick clichés.",
		}, "\n")
	case "lifestyle", "fitness", "beauty", "fashion":
		return strings.Join([]string{
			"Niche voice pack (lifestyle):",
			"- Sound relatable and aspirational without being salesy.",
			"- Use sensory details and personal micro-stories.",
			"- Keep energy warm; invite the viewer in.",
			"- Avoid lecture mode.",
		}, "\n")
	default:
		return strings.Join([]string{
			"Niche voice pack (general creator):",
			"- Sound like a real person talking to camera, not an essay.",
			"- Hook in the first line; keep momentum.",
			"- Use the creator's stated preferences over generic creator tropes.",
		}, "\n")
	}
}
