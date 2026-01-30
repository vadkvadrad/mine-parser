package service

import (
	"mine-parser/internal/models"
	"mine-parser/internal/repo"
	"time"
)

type AdvancementService interface {
	GrantAdvancement(playerID, advancementName string, timestamp time.Time) error
	GetPlayerAdvancements(playerID string) ([]models.Advancement, error)
	IsAdvancementUnlocked(playerID, advancementName string) (bool, error)
}

type advancementService struct {
	advanceRepo repo.AdvancementRepository
}

func NewAdvancementService(advanceRepo repo.AdvancementRepository) AdvancementService {
	return &advancementService{
		advanceRepo: advanceRepo,
	}
}

func (s *advancementService) GrantAdvancement(playerID, advancementName string, timestamp time.Time) error {
	// Проверяем, не получено ли уже это достижение
	hasCompleted, err := s.advanceRepo.HasPlayerCompleted(advancementName, playerID)
	if err != nil {
		return err
	}

	// Если достижение уже получено, не создаем дубликат
	if hasCompleted {
		return nil
	}

	// Создаем новое достижение
	advancement := &models.Advancement{
		PlayerID:        playerID,
		Timestamp:       timestamp,
		AdvancementName: advancementName,
	}

	return s.advanceRepo.Create(advancement)
}

func (s *advancementService) GetPlayerAdvancements(playerID string) ([]models.Advancement, error) {
	return s.advanceRepo.ListByPlayer(playerID)
}

func (s *advancementService) IsAdvancementUnlocked(playerID, advancementName string) (bool, error) {
	return s.advanceRepo.HasPlayerCompleted(advancementName, playerID)
}
