package repo

import (
	"mine-parser/internal/models"

	"gorm.io/gorm"
)

type AdvancementRepository interface {
	Create(adv *models.Advancement) error
	ListByPlayer(playerID string) ([]models.Advancement, error)
	HasPlayerCompleted(advancementName string, playerID string) (bool, error)
	CountAdvancementsByPlayer(playerID string) (int64, error)
}

type advancementRepository struct {
	db *gorm.DB
}

func NewAdvancementRepository(db *gorm.DB) AdvancementRepository {
	return &advancementRepository{db: db}
}

func (r *advancementRepository) Create(adv *models.Advancement) error {
	return r.db.Create(adv).Error
}

func (r *advancementRepository) ListByPlayer(playerID string) ([]models.Advancement, error) {
	var advancements []models.Advancement
	err := r.db.Where("player_id = ?", playerID).
		Order("timestamp DESC").
		Find(&advancements).Error
	return advancements, err
}

func (r *advancementRepository) HasPlayerCompleted(advancementName string, playerID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Advancement{}).
		Where("player_id = ? AND advancement_name = ?", playerID, advancementName).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *advancementRepository) CountAdvancementsByPlayer(playerID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Advancement{}).
		Where("player_id = ?", playerID).
		Count(&count).Error
	return count, err
}
