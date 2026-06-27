package services

import (
	"regexp"
	"strings"
	"unicode"
)

var sentenceSplitRe = regexp.MustCompile(`[.!?]+`)

func countWords(text string) int {
	tokens := tokenize(text)
	n := 0
	for _, t := range tokens {
		if t != "" {
			n++
		}
	}
	return n
}

func splitSentences(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	parts := sentenceSplitRe.Split(text, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{text}
	}
	return out
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '\''
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func isHinglishToken(token string) bool {
	hinglish := map[string]bool{
		"yaar": true, "matlab": true, "suno": true, "dekho": true, "bas": true,
		"achha": true, "accha": true, "nahi": true, "nahin": true, "kya": true,
		"hai": true, "hain": true, "bhai": true, "arre": true, "yeh": true,
		"woh": true, "tum": true, "hum": true, "mera": true, "tera": true,
	}
	return hinglish[token]
}
