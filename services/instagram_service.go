package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/ashisharyan/ghostwriter-prompt-engine/utils"
)

const instagramOAuthScopes = "instagram_basic,pages_show_list,pages_read_engagement"
const minCaptionWordsForTranscriptSkip = 25

// InstagramService handles Meta OAuth and Graph API reel import.
type InstagramService interface {
	Configured() bool
	AuthURL(state string) (string, error)
	HandleCallback(code, state string) (sessionID string, err error)
	FetchReels(sessionID string) (*models.InstagramImportBundle, error)
	PrepareReels(sessionID string, reelIDs []string, transcribe bool) ([]models.InstagramReel, error)
}

type DefaultInstagramService struct {
	Secrets      *utils.Secrets
	HTTPClient   *http.Client
	Sessions     *instagramSessionStore
	MediaURLs    *reelMediaStore
	Transcriber  TranscriptionService
	GraphBaseURL string // test override
}

func NewDefaultInstagramService(secrets *utils.Secrets, transcriber TranscriptionService) InstagramService {
	return &DefaultInstagramService{
		Secrets:     secrets,
		HTTPClient:  &http.Client{Timeout: 60 * time.Second},
		Sessions:    newInstagramSessionStore(time.Hour),
		MediaURLs:   newReelMediaStore(),
		Transcriber: transcriber,
	}
}

func (s *DefaultInstagramService) Configured() bool {
	return s.Secrets != nil && s.Secrets.MetaAppID != "" && s.Secrets.MetaAppSecret != ""
}

func (s *DefaultInstagramService) redirectURI() string {
	if s.Secrets.MetaRedirectURI != "" {
		return s.Secrets.MetaRedirectURI
	}
	return strings.TrimRight(s.Secrets.AppBaseURL, "/") + "/api/v1/instagram/callback"
}

func (s *DefaultInstagramService) graphBase() string {
	if s.GraphBaseURL != "" {
		return s.GraphBaseURL
	}
	return "https://graph.facebook.com/" + s.Secrets.MetaGraphVersion
}

func (s *DefaultInstagramService) AuthURL(state string) (string, error) {
	if !s.Configured() {
		return "", fmt.Errorf("instagram connect is not configured: set META_APP_ID and META_APP_SECRET (see planning/meta-instagram-setup.md)")
	}
	q := url.Values{}
	q.Set("client_id", s.Secrets.MetaAppID)
	q.Set("redirect_uri", s.redirectURI())
	q.Set("scope", instagramOAuthScopes)
	q.Set("response_type", "code")
	q.Set("state", state)
	return "https://www.facebook.com/" + s.Secrets.MetaGraphVersion + "/dialog/oauth?" + q.Encode(), nil
}

func (s *DefaultInstagramService) HandleCallback(code, state string) (string, error) {
	if state == "" {
		return "", fmt.Errorf("missing oauth state")
	}
	token, err := s.exchangeCode(code)
	if err != nil {
		return "", err
	}
	longLived, err := s.exchangeLongLived(token)
	if err != nil {
		longLived = token
	}
	pageToken, igUserID, err := s.resolveIGAccount(longLived)
	if err != nil {
		return "", err
	}
	s.Sessions.Put(state, InstagramSession{
		UserAccessToken: longLived,
		PageAccessToken: pageToken,
		IGUserID:        igUserID,
	})
	return state, nil
}

func (s *DefaultInstagramService) FetchReels(sessionID string) (*models.InstagramImportBundle, error) {
	sess, ok := s.Sessions.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("instagram session expired; connect again")
	}
	profile, err := s.fetchProfile(sess.PageAccessToken, sess.IGUserID)
	if err != nil {
		return nil, err
	}
	reels, err := s.fetchMedia(sess.PageAccessToken, sess.IGUserID, sessionID)
	if err != nil {
		return nil, err
	}
	return &models.InstagramImportBundle{Profile: profile, Reels: reels}, nil
}

func (s *DefaultInstagramService) PrepareReels(sessionID string, reelIDs []string, transcribe bool) ([]models.InstagramReel, error) {
	bundle, err := s.FetchReels(sessionID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]models.InstagramReel, len(bundle.Reels))
	for _, r := range bundle.Reels {
		byID[r.ID] = r
	}
	out := make([]models.InstagramReel, 0, len(reelIDs))
	for _, id := range reelIDs {
		reel, ok := byID[id]
		if !ok {
			continue
		}
		resolved, err := s.resolveReelText(sessionID, reel, transcribe)
		if err != nil {
			return nil, err
		}
		resolved.Selected = true
		out = append(out, resolved)
	}
	return out, nil
}

func (s *DefaultInstagramService) resolveReelText(sessionID string, reel models.InstagramReel, transcribe bool) (models.InstagramReel, error) {
	caption := strings.TrimSpace(reel.Caption)
	if countWords(caption) >= minCaptionWordsForTranscriptSkip {
		reel.Text = caption
		reel.TextSource = models.ReelTextSourceCaption
		return reel, nil
	}
	if transcribe && s.Transcriber != nil && s.Transcriber.Available() {
		mediaURL, ok := s.MediaURLs.Get(sessionID, reel.ID)
		if ok && mediaURL != "" {
			text, err := s.Transcriber.TranscribeURL(mediaURL)
			if err == nil && strings.TrimSpace(text) != "" {
				reel.Transcript = strings.TrimSpace(text)
				reel.Text = reel.Transcript
				reel.TextSource = models.ReelTextSourceTranscript
				return reel, nil
			}
		}
	}
	if caption != "" {
		reel.Text = caption
		reel.TextSource = models.ReelTextSourceCaption
		return reel, nil
	}
	return reel, fmt.Errorf("reel %s has no caption; enable transcribe with OPENAI_API_KEY or pick another reel", reel.ID)
}

func (s *DefaultInstagramService) exchangeCode(code string) (string, error) {
	q := url.Values{}
	q.Set("client_id", s.Secrets.MetaAppID)
	q.Set("client_secret", s.Secrets.MetaAppSecret)
	q.Set("redirect_uri", s.redirectURI())
	q.Set("code", code)
	var resp struct {
		AccessToken string `json:"access_token"`
		Error       *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := s.getJSON(s.graphBase()+"/oauth/access_token?"+q.Encode(), "", &resp); err != nil {
		return "", err
	}
	if resp.AccessToken == "" {
		msg := "token exchange failed"
		if resp.Error != nil {
			msg = resp.Error.Message
		}
		return "", fmt.Errorf(msg)
	}
	return resp.AccessToken, nil
}

func (s *DefaultInstagramService) exchangeLongLived(shortToken string) (string, error) {
	q := url.Values{}
	q.Set("grant_type", "fb_exchange_token")
	q.Set("client_id", s.Secrets.MetaAppID)
	q.Set("client_secret", s.Secrets.MetaAppSecret)
	q.Set("fb_exchange_token", shortToken)
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := s.getJSON(s.graphBase()+"/oauth/access_token?"+q.Encode(), "", &resp); err != nil {
		return "", err
	}
	return resp.AccessToken, nil
}

func (s *DefaultInstagramService) resolveIGAccount(userToken string) (pageToken, igUserID string, err error) {
	var resp struct {
		Data []struct {
			AccessToken              string `json:"access_token"`
			InstagramBusinessAccount *struct {
				ID string `json:"id"`
			} `json:"instagram_business_account"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	fields := "access_token,instagram_business_account"
	if err := s.getJSON(s.graphBase()+"/me/accounts?fields="+url.QueryEscape(fields), userToken, &resp); err != nil {
		return "", "", err
	}
	for _, page := range resp.Data {
		if page.InstagramBusinessAccount != nil && page.InstagramBusinessAccount.ID != "" {
			return page.AccessToken, page.InstagramBusinessAccount.ID, nil
		}
	}
	if resp.Error != nil {
		return "", "", fmt.Errorf(resp.Error.Message)
	}
	return "", "", fmt.Errorf("no Instagram Business or Creator account linked to your Facebook Page; convert your Instagram account and link a Page in Meta settings")
}

func (s *DefaultInstagramService) fetchProfile(pageToken, igUserID string) (models.InstagramProfileSnapshot, error) {
	var resp struct {
		Username       string `json:"username"`
		Name           string `json:"name"`
		Biography      string `json:"biography"`
		FollowersCount int    `json:"followers_count"`
		MediaCount     int    `json:"media_count"`
	}
	fields := "username,name,biography,followers_count,media_count"
	path := fmt.Sprintf("%s/%s?fields=%s", s.graphBase(), igUserID, url.QueryEscape(fields))
	if err := s.getJSON(path, pageToken, &resp); err != nil {
		return models.InstagramProfileSnapshot{}, err
	}
	return models.InstagramProfileSnapshot{
		Username:       resp.Username,
		Name:           resp.Name,
		Biography:      resp.Biography,
		FollowersCount: resp.FollowersCount,
		MediaCount:     resp.MediaCount,
	}, nil
}

func (s *DefaultInstagramService) fetchMedia(pageToken, igUserID, sessionID string) ([]models.InstagramReel, error) {
	fields := "id,caption,media_type,media_product_type,media_url,permalink,timestamp,like_count,comments_count"
	path := fmt.Sprintf("%s/%s/media?fields=%s&limit=25", s.graphBase(), igUserID, url.QueryEscape(fields))
	var resp struct {
		Data []struct {
			ID               string `json:"id"`
			Caption          string `json:"caption"`
			MediaType        string `json:"media_type"`
			MediaProductType string `json:"media_product_type"`
			MediaURL         string `json:"media_url"`
			Permalink        string `json:"permalink"`
			Timestamp        string `json:"timestamp"`
			LikeCount        int    `json:"like_count"`
			CommentsCount    int    `json:"comments_count"`
		} `json:"data"`
	}
	if err := s.getJSON(path, pageToken, &resp); err != nil {
		return nil, err
	}
	out := make([]models.InstagramReel, 0)
	for _, item := range resp.Data {
		if !isReelMedia(item.MediaProductType, item.MediaType) {
			continue
		}
		if item.MediaURL != "" {
			s.MediaURLs.Put(sessionID, item.ID, item.MediaURL)
		}
		out = append(out, models.InstagramReel{
			ID:            item.ID,
			Permalink:     item.Permalink,
			Caption:       item.Caption,
			Timestamp:     item.Timestamp,
			LikeCount:     item.LikeCount,
			CommentsCount: item.CommentsCount,
		})
	}
	return out, nil
}

func isReelMedia(productType, mediaType string) bool {
	pt := strings.ToUpper(productType)
	mt := strings.ToUpper(mediaType)
	return pt == "REELS" || mt == "VIDEO"
}

func (s *DefaultInstagramService) getJSON(urlStr, accessToken string, dest interface{}) error {
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return err
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := s.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("instagram api error: %s", string(body))
	}
	return json.Unmarshal(body, dest)
}

func (s *DefaultInstagramService) httpClient() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return http.DefaultClient
}

// BuildSamplesFromInstagramReels converts selected reels to writing samples.
func BuildSamplesFromInstagramReels(reels []models.InstagramReel) []models.WritingSample {
	out := make([]models.WritingSample, 0, len(reels))
	now := time.Now()
	for _, reel := range reels {
		if !reel.Selected {
			continue
		}
		text := strings.TrimSpace(reel.Text)
		if text == "" {
			text = strings.TrimSpace(reel.Caption)
		}
		if text == "" {
			text = strings.TrimSpace(reel.Transcript)
		}
		if text == "" {
			continue
		}
		source := models.SampleSourceInstagramReel
		if reel.TextSource == models.ReelTextSourceTranscript {
			source = models.SampleSourceInstagramTranscript
		}
		out = append(out, models.WritingSample{
			Text:      text,
			Source:    source,
			CreatedAt: now,
		})
	}
	return out
}

// ProfileHintsFromInstagram maps IG profile into creator profile fields.
func ProfileHintsFromInstagram(p models.InstagramProfileSnapshot) map[string]string {
	hints := map[string]string{"platform": "Instagram"}
	if p.Username != "" {
		hints["channel"] = "@" + p.Username
	}
	if p.Name != "" {
		hints["name"] = p.Name
	}
	if p.Biography != "" {
		hints["bio"] = p.Biography
	}
	hints["content_type"] = "Reels"
	return hints
}

// TranscriptionService transcribes video audio from a URL.
type TranscriptionService interface {
	Available() bool
	TranscribeURL(mediaURL string) (string, error)
}

type OpenAITranscriptionService struct {
	APIKey     string
	HTTPClient *http.Client
	Model      string
}

func NewOpenAITranscriptionService(apiKey string) TranscriptionService {
	if apiKey == "" {
		return nil
	}
	return &OpenAITranscriptionService{APIKey: apiKey, Model: "whisper-1"}
}

func (t *OpenAITranscriptionService) Available() bool {
	return t != nil && t.APIKey != ""
}

func (t *OpenAITranscriptionService) TranscribeURL(mediaURL string) (string, error) {
	client := t.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Get(mediaURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("failed to download reel audio: status %d", resp.StatusCode)
	}
	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return t.transcribeBytes(audioBytes)
}

func (t *OpenAITranscriptionService) transcribeBytes(audio []byte) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("model", t.Model)
	part, err := w.CreateFormFile("file", "reel.mp4")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(audio); err != nil {
		return "", err
	}
	_ = w.Close()

	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+t.APIKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := t.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper api error: %s", string(respBody))
	}
	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	return strings.TrimSpace(parsed.Text), nil
}
