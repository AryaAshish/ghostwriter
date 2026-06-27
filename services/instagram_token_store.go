package services

import (
	"sync"
	"time"
)

// InstagramSession holds OAuth tokens for a connect flow.
type InstagramSession struct {
	UserAccessToken string
	PageAccessToken string
	IGUserID        string
	ExpiresAt       time.Time
}

type instagramSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]InstagramSession
	ttl      time.Duration
}

func newInstagramSessionStore(ttl time.Duration) *instagramSessionStore {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &instagramSessionStore{
		sessions: make(map[string]InstagramSession),
		ttl:      ttl,
	}
}

func (s *instagramSessionStore) Put(id string, sess InstagramSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.ExpiresAt = time.Now().Add(s.ttl)
	s.sessions[id] = sess
}

func (s *instagramSessionStore) Get(id string) (InstagramSession, bool) {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok || time.Now().After(sess.ExpiresAt) {
		if ok {
			s.Delete(id)
		}
		return InstagramSession{}, false
	}
	return sess, true
}

func (s *instagramSessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// reelMediaURL maps reel ID to downloadable media URL (server-side only).
type reelMediaStore struct {
	mu   sync.RWMutex
	urls map[string]map[string]string // sessionID -> reelID -> mediaURL
}

func newReelMediaStore() *reelMediaStore {
	return &reelMediaStore{urls: make(map[string]map[string]string)}
}

func (s *reelMediaStore) Put(sessionID, reelID, mediaURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.urls[sessionID] == nil {
		s.urls[sessionID] = make(map[string]string)
	}
	s.urls[sessionID][reelID] = mediaURL
}

func (s *reelMediaStore) Get(sessionID, reelID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.urls[sessionID] == nil {
		return "", false
	}
	u, ok := s.urls[sessionID][reelID]
	return u, ok
}

func (s *reelMediaStore) ClearSession(sessionID string) {
	s.mu.Lock()
	delete(s.urls, sessionID)
	s.mu.Unlock()
}
