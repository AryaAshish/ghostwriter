package models

// InstagramReel is one reel imported for voice fingerprinting.
type InstagramReel struct {
	ID            string `json:"id"`
	Permalink     string `json:"permalink,omitempty"`
	Caption       string `json:"caption,omitempty"`
	Transcript    string `json:"transcript,omitempty"`
	Text          string `json:"text,omitempty"`
	TextSource    string `json:"text_source,omitempty"` // caption | transcript
	Timestamp     string `json:"timestamp,omitempty"`
	LikeCount     int    `json:"like_count,omitempty"`
	CommentsCount int    `json:"comments_count,omitempty"`
	Selected      bool   `json:"selected"`
}

// InstagramProfileSnapshot is public profile metadata from Graph API.
type InstagramProfileSnapshot struct {
	Username       string `json:"username"`
	Name           string `json:"name"`
	Biography      string `json:"biography"`
	FollowersCount int    `json:"followers_count"`
	MediaCount     int    `json:"media_count"`
}

// InstagramImportBundle groups profile + reels for onboarding submit.
type InstagramImportBundle struct {
	Profile InstagramProfileSnapshot `json:"profile"`
	Reels   []InstagramReel          `json:"reels"`
}
