package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAIClientChatCompletion(t *testing.T) {
	server := newMockOpenAIServer(t, "generated script", 0)
	defer server.Close()
	client := newOpenAIClientForTest(t, server)

	out, err := client.ChatCompletion("system", "user", 100, 0.5)
	if err != nil || out != "generated script" {
		t.Fatalf("chat completion failed: %q err=%v", out, err)
	}

	out2, err := client.ChatCompletionMessages([]openAIMessage{{Role: "user", Content: "hello"}}, 50, 0.3)
	if err != nil || out2 != "generated script" {
		t.Fatalf("messages completion failed: %q err=%v", out2, err)
	}
}

func TestOpenAIClientErrors(t *testing.T) {
	server := newMockOpenAIServer(t, "", 400)
	defer server.Close()
	client := newOpenAIClientForTest(t, server)
	if _, err := client.ChatCompletion("s", "u", 10, 0.1); err == nil {
		t.Fatal("expected API error")
	}

	emptyChoices := httptestNewEmptyChoicesServer(t)
	defer emptyChoices.Close()
	client2 := &OpenAIClient{
		APIKey:     "k",
		Model:      "m",
		BaseURL:    emptyChoices.URL,
		HTTPClient: emptyChoices.Client(),
	}
	if _, err := client2.ChatCompletion("s", "u", 10, 0.1); err == nil {
		t.Fatal("expected no choices error")
	}

	retryServer := httptestNewRetryServer(t)
	defer retryServer.Close()
	client3 := &OpenAIClient{
		APIKey:     "k",
		Model:      "m",
		BaseURL:    retryServer.URL,
		HTTPClient: retryServer.Client(),
	}
	out, err := client3.ChatCompletion("s", "u", 10, 0.1)
	if err != nil || out != "ok after retry" {
		t.Fatalf("retry failed: %q err=%v", out, err)
	}

	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer badJSON.Close()
	client4 := &OpenAIClient{APIKey: "k", Model: "m", BaseURL: badJSON.URL, HTTPClient: badJSON.Client()}
	if _, err := client4.ChatCompletion("s", "u", 10, 0.1); err == nil {
		t.Fatal("expected json unmarshal error")
	}

	netFail := &OpenAIClient{APIKey: "k", Model: "m", BaseURL: "http://127.0.0.1:1", HTTPClient: &http.Client{Timeout: time.Millisecond}}
	if _, err := netFail.ChatCompletion("s", "u", 10, 0.1); err == nil {
		t.Fatal("expected network failure after retries")
	}
}

func httptestNewEmptyChoicesServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
}

func httptestNewRetryServer(t *testing.T) *httptest.Server {
	t.Helper()
	attempts := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"retry"}`))
			return
		}
		resp := openAIChatResponse{}
		resp.Choices = append(resp.Choices, struct {
			Message openAIMessage `json:"message"`
		}{Message: openAIMessage{Content: "ok after retry"}})
		_ = json.NewEncoder(w).Encode(resp)
	}))
}
