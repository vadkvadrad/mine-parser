package repo

import (
	"mine-parser/internal/models"
	"time"

	"gorm.io/gorm"
)

type PlayerRepository interface {
	GetOrCreate(playerID string, username string, timestamp time.Time) (*models.Player, error)
	UpdateLastSeen(playerID string, lastSeen time.Time) error
	FindByID(playerID string) (*models.Player, error)
	ListAll() ([]models.Player, error)
	FindByUsername(username string) (*models.Player, error)
}

type playerRepository struct {
	db *gorm.DB
}

func NewPlayerRepository(db *gorm.DB) PlayerRepository {
	return &playerRepository{db: db}
}

func (r *playerRepository) GetOrCreate(playerID string, username string, timestamp time.Time) (*models.Player, error) {
	player := &models.Player{
		ID:        playerID,
		Username:  username,
		FirstSeen: timestamp,
		LastSeen:  timestamp,
	}

	err := r.db.FirstOrCreate(player, models.Player{ID: playerID}).Error
	if err != nil {
		return nil, err
	}

	// Если игрок уже существовал, обновляем username и last_seen
	if player.FirstSeen.Before(timestamp) {
		player.Username = username
		player.LastSeen = timestamp
		if err := r.db.Save(player).Error; err != nil {
			return nil, err
		}
	}

	return player, nil
}

func (r *playerRepository) UpdateLastSeen(playerID string, lastSeen time.Time) error {
	return r.db.Model(&models.Player{}).
		Where("id = ?", playerID).
		Update("last_seen", lastSeen).Error
}

func (r *playerRepository) FindByID(playerID string) (*models.Player, error) {
	var player models.Player
	err := r.db.Where("id = ?", playerID).First(&player).Error
	if err != nil {
		return nil, err
	}
	return &player, nil
}

func (r *playerRepository) ListAll() ([]models.Player, error) {
	var players []models.Player
	err := r.db.Find(&players).Error
	return players, err
}

func (r *playerRepository) FindByUsername(username string) (*models.Player, error) {
	var player models.Player
	err := r.db.Where("username = ?", username).First(&player).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil // не ошибка, просто нет такого
	}
	return &player, err
}
