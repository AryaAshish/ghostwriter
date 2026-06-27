package services

import (
	"encoding/json"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"gorm.io/gorm"
)

type FeedbackSubmitResult struct {
	Feedback        *models.ScriptFeedback
	Deltas          map[string]int
	VoiceMode       string
	VoiceConfidence int
	ShiftScore      int
}

type FeedbackService interface {
	SubmitFeedback(scriptID uint, req models.ScriptFeedbackRequest) (*FeedbackSubmitResult, error)
}

type GormFeedbackService struct {
	DB             *gorm.DB
	PersonaService PersonaService
	ScriptRepo     ScriptRepository
}

func NewGormFeedbackService(db *gorm.DB, persona PersonaService, scriptRepo ScriptRepository) FeedbackService {
	return &GormFeedbackService{DB: db, PersonaService: persona, ScriptRepo: scriptRepo}
}

func (s *GormFeedbackService) SubmitFeedback(scriptID uint, req models.ScriptFeedbackRequest) (*FeedbackSubmitResult, error) {
	script, err := s.ScriptRepo.GetScriptByID(scriptID)
	if err != nil {
		return nil, err
	}

	result, err := s.PersonaService.ApplyFeedback(script.CreatorID, req)
	if err != nil {
		return nil, err
	}

	deltaJSON, _ := json.Marshal(result.Deltas)
	feedback := &models.ScriptFeedback{
		ScriptID:      scriptID,
		CreatorID:     script.CreatorID,
		Rating:        req.Rating,
		Notes:         req.Notes,
		AppliedDeltas: string(deltaJSON),
	}
	if err := s.DB.Create(feedback).Error; err != nil {
		return nil, err
	}
	return &FeedbackSubmitResult{
		Feedback:        feedback,
		Deltas:          result.Deltas,
		VoiceMode:       result.VoiceMode,
		VoiceConfidence: result.VoiceConfidence,
		ShiftScore:      result.ShiftScore,
	}, nil
}
