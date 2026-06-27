package models

import "time"

// ScriptFeedback records user feedback on a generated script.
type ScriptFeedback struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ScriptID      uint      `json:"script_id"`
	CreatorID     uint      `json:"creator_id"`
	Rating        string    `json:"rating"`
	Notes         string    `json:"notes"`
	AppliedDeltas string    `gorm:"type:text" json:"applied_deltas"`
	CreatedAt     time.Time `json:"created_at"`
}
