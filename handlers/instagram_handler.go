package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/gin-gonic/gin"
)

type InstagramHandler struct {
	InstagramService services.InstagramService
	AppBaseURL       string
}

func NewInstagramHandler(instagramService services.InstagramService, appBaseURL string) *InstagramHandler {
	return &InstagramHandler{InstagramService: instagramService, AppBaseURL: appBaseURL}
}

func (h *InstagramHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"configured": h.InstagramService.Configured(),
		"scopes":     "instagram_basic,pages_show_list,pages_read_engagement",
		"setup_doc":  "/planning/meta-instagram-setup.md",
	})
}

func (h *InstagramHandler) AuthURL(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		state = newOAuthState()
	}
	url, err := h.InstagramService.AuthURL(state)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"auth_url": url, "state": state})
}

func (h *InstagramHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		errMsg := c.Query("error_description")
		if errMsg == "" {
			errMsg = c.Query("error")
		}
		if errMsg == "" {
			errMsg = "instagram authorization denied"
		}
		c.Redirect(http.StatusFound, h.appURL("/app/?instagram_error="+urlQuery(errMsg)))
		return
	}
	sessionID, err := h.InstagramService.HandleCallback(code, state)
	if err != nil {
		c.Redirect(http.StatusFound, h.appURL("/app/?instagram_error="+urlQuery(err.Error())))
		return
	}
	c.Redirect(http.StatusFound, h.appURL("/app/?ig_session="+sessionID))
}

func (h *InstagramHandler) Reels(c *gin.Context) {
	sessionID := c.Query("session")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session query param required"})
		return
	}
	bundle, err := h.InstagramService.FetchReels(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bundle)
}

func (h *InstagramHandler) Prepare(c *gin.Context) {
	var req models.InstagramPrepareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reels, err := h.InstagramService.PrepareReels(req.SessionID, req.ReelIDs, req.Transcribe)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reels": reels})
}

func (h *InstagramHandler) appURL(path string) string {
	base := h.AppBaseURL
	if base == "" {
		base = "http://localhost:8080"
	}
	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}
	return base + path
}

func newOAuthState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func urlQuery(s string) string {
	return url.QueryEscape(s)
}
