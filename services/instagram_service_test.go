package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/ashisharyan/ghostwriter-prompt-engine/utils"
)

func TestInstagramSessionStore(t *testing.T) {
	store := newInstagramSessionStore(time.Millisecond)
	store.Put("s1", InstagramSession{IGUserID: "123"})
	if _, ok := store.Get("s1"); !ok {
		t.Fatal("expected session")
	}
	time.Sleep(2 * time.Millisecond)
	if _, ok := store.Get("s1"); ok {
		t.Fatal("expected expired session")
	}
}

func TestBuildSamplesFromInstagramReels(t *testing.T) {
	samples := BuildSamplesFromInstagramReels([]models.InstagramReel{
		{ID: "1", Text: "okay so suno yaar", TextSource: models.ReelTextSourceCaption, Selected: true},
		{ID: "2", Text: "second reel caption here", TextSource: models.ReelTextSourceTranscript, Selected: true},
		{ID: "3", Caption: "skipped", Selected: false},
	})
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}
	if samples[1].Source != models.SampleSourceInstagramTranscript {
		t.Fatal("expected transcript source tag")
	}
}

func TestProfileHintsFromInstagram(t *testing.T) {
	hints := ProfileHintsFromInstagram(models.InstagramProfileSnapshot{
		Username:  "amit",
		Name:      "Amit",
		Biography: "Comic",
	})
	if hints["channel"] != "@amit" || hints["platform"] != "Instagram" {
		t.Fatalf("unexpected hints: %+v", hints)
	}
}

func TestValidateSubmitProfileInstagram(t *testing.T) {
	ok := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathImportInstagram,
		InstagramReels: []models.InstagramReel{
			{Selected: true, Text: strings.Repeat("word ", 50)},
			{Selected: true, Text: strings.Repeat("word ", 50)},
		},
	})
	if !ok.OK {
		t.Fatalf("expected ok, got %+v", ok)
	}
	bad := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathImportInstagram,
		InstagramReels: []models.InstagramReel{{Selected: true, Text: "one"}},
	})
	if bad.OK {
		t.Fatal("expected validation error for single reel")
	}
}

func TestInstagramServiceAuthURLAndCallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/oauth/access_token"):
			w.Write([]byte(`{"access_token":"user-token"}`))
		case strings.Contains(r.URL.Path, "/me/accounts"):
			w.Write([]byte(`{"data":[{"access_token":"page-token","instagram_business_account":{"id":"ig-1"}}]}`))
		case strings.Contains(r.URL.Path, "/ig-1/media"):
			w.Write([]byte(`{"data":[{"id":"r1","caption":"` + strings.Repeat("word ", 30) + `","media_type":"VIDEO","media_product_type":"REELS","permalink":"https://instagram.com/reel/1","timestamp":"2026-01-01T00:00:00+0000","like_count":10,"comments_count":2}]}`))
		case strings.Contains(r.URL.Path, "/ig-1"):
			w.Write([]byte(`{"username":"amit","name":"Amit","biography":"Comic","followers_count":100,"media_count":5}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	secrets := &utils.Secrets{
		MetaAppID:        "app",
		MetaAppSecret:    "secret",
		MetaRedirectURI:  "http://localhost/callback",
		MetaGraphVersion: "v21.0",
		AppBaseURL:       "http://localhost",
	}
	ig := &DefaultInstagramService{
		Secrets:      secrets,
		HTTPClient:   srv.Client(),
		Sessions:     newInstagramSessionStore(time.Hour),
		MediaURLs:    newReelMediaStore(),
		GraphBaseURL: srv.URL,
	}

	url, err := ig.AuthURL("state-1")
	if err != nil || !strings.Contains(url, "facebook.com") {
		t.Fatalf("auth url: %v %s", err, url)
	}
	sessionID, err := ig.HandleCallback("code-1", "state-1")
	if err != nil || sessionID != "state-1" {
		t.Fatalf("callback: %v", err)
	}
	bundle, err := ig.FetchReels("state-1")
	if err != nil {
		t.Fatalf("fetch reels: %v", err)
	}
	if bundle.Profile.Username != "amit" || len(bundle.Reels) != 1 {
		t.Fatalf("unexpected bundle: %+v", bundle)
	}
	prepared, err := ig.PrepareReels("state-1", []string{"r1"}, false)
	if err != nil || len(prepared) != 1 || prepared[0].TextSource != models.ReelTextSourceCaption {
		t.Fatalf("prepare: %v %+v", err, prepared)
	}
}

func TestInstagramServiceNotConfigured(t *testing.T) {
	svc := NewDefaultInstagramService(&utils.Secrets{}, nil)
	if svc.Configured() {
		t.Fatal("expected not configured")
	}
	if _, err := svc.AuthURL("x"); err == nil {
		t.Fatal("expected error when not configured")
	}
}

func TestBuildSamplesFromSubmitInstagram(t *testing.T) {
	samples := BuildSamplesFromSubmit(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathImportInstagram,
		InstagramReels: []models.InstagramReel{
			{Selected: true, Text: "reel one text"},
			{Selected: true, Text: "reel two text"},
		},
	})
	if len(samples) != 2 {
		t.Fatalf("expected 2, got %d", len(samples))
	}
}

func TestIsReelMedia(t *testing.T) {
	if !isReelMedia("REELS", "VIDEO") || isReelMedia("FEED", "IMAGE") {
		t.Fatal("reel media filter")
	}
}
