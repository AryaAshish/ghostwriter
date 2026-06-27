package services

import "testing"

func TestNicheVoiceRules(t *testing.T) {
	for _, genre := range []string{"comedy", "finance", "lifestyle", "unknown"} {
		rules := NicheVoiceRules(genre)
		if rules == "" {
			t.Fatalf("expected rules for genre %s", genre)
		}
	}
}
