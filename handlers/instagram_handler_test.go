package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/gin-gonic/gin"
)

type stubInstagramService struct {
	configured bool
	authURL    string
	authErr    error
}

func (s *stubInstagramService) Configured() bool { return s.configured }
func (s *stubInstagramService) AuthURL(state string) (string, error) {
	if s.authErr != nil {
		return "", s.authErr
	}
	return s.authURL, nil
}
func (s *stubInstagramService) HandleCallback(code, state string) (string, error) { return state, nil }
func (s *stubInstagramService) FetchReels(sessionID string) (*models.InstagramImportBundle, error) {
	return &models.InstagramImportBundle{}, nil
}
func (s *stubInstagramService) PrepareReels(sessionID string, reelIDs []string, transcribe bool) ([]models.InstagramReel, error) {
	return nil, nil
}

func TestInstagramHandlerStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInstagramHandler(&stubInstagramService{configured: true}, "http://localhost:8080")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	h.Status(c)
	if w.Code != http.StatusOK {
		t.Fatal("expected ok status")
	}
}

func TestInstagramHandlerAuthURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInstagramHandler(&stubInstagramService{configured: true, authURL: "https://facebook.com/oauth"}, "http://localhost:8080")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	h.AuthURL(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected ok, got %d", w.Code)
	}
}
