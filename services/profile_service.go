package services

import (
	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"gorm.io/gorm"
)

type ProfileService interface {
	CreateProfile(profile *models.CreatorProfile) (uint, error)
	GetProfileByID(id uint) (*models.CreatorProfile, error)
}

type GormProfileService struct {
	DB *gorm.DB
}

func NewGormProfileService(db *gorm.DB) ProfileService {
	return &GormProfileService{DB: db}
}

func (s *GormProfileService) CreateProfile(profile *models.CreatorProfile) (uint, error) {
	if err := s.DB.Create(profile).Error; err != nil {
		return 0, err
	}
	return profile.ID, nil
}

func (s *GormProfileService) GetProfileByID(id uint) (*models.CreatorProfile, error) {
	var profile models.CreatorProfile
	if err := s.DB.First(&profile, id).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

