package services

import "testing"

func TestCountWordsAndTokenize(t *testing.T) {
	if countWords("") != 0 {
		t.Fatal("empty should be 0 words")
	}
	if countWords("Hello world") != 2 {
		t.Fatal("expected 2 words")
	}
	tokens := tokenize("Yaar, matlab — okay!")
	if len(tokens) < 3 {
		t.Fatalf("expected tokens, got %v", tokens)
	}
}

func TestSplitSentences(t *testing.T) {
	sents := splitSentences("First line. Second line! Third?")
	if len(sents) != 3 {
		t.Fatalf("expected 3 sentences, got %d", len(sents))
	}
	if len(splitSentences("")) != 0 {
		t.Fatal("empty text should have no sentences")
	}
	if len(splitSentences("no punctuation")) != 1 {
		t.Fatal("expected single sentence fallback")
	}
}

func TestIsHinglishToken(t *testing.T) {
	if !isHinglishToken("yaar") || isHinglishToken("hello") {
		t.Fatal("hinglish detection failed")
	}
}
