package utils

import "testing"

func TestLoadSecrets(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_MODEL", "gpt-4")
	secrets := LoadSecrets()
	if secrets.OpenAIAPIKey != "test-key" || secrets.OpenAIModel != "gpt-4" {
		t.Fatalf("unexpected secrets: %+v", secrets)
	}
}
