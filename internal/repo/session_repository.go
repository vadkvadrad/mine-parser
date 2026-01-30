package repo

import (
	"mine-parser/internal/models"
	"time"

	"gorm.io/gorm"
)

type SessionRepository interface {
	Create(session *models.Session) error
	CloseSession(sessionID uint, leaveTime time.Time) error
	GetActiveSessionByPlayer(playerID string) (*models.Session, error)
	ListByPlayer(playerID string) ([]models.Session, error)
	ListActive() ([]models.Session, error)
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(session *models.Session) error {
	return r.db.Create(session).Error
}

func (r *sessionRepository) CloseSession(sessionID uint, leaveTime time.Time) error {
	return r.db.Model(&models.Session{}).
		Where("id = ?", sessionID).
		Update("leave_time", leaveTime).Error
}

func (r *sessionRepository) GetActiveSessionByPlayer(playerID string) (*models.Session, error) {
	var sessions []models.Session
	err := r.db.Where("player_id = ? AND leave_time IS NULL", playerID).
		Order("join_time DESC").
		Limit(1).
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	return &sessions[0], nil
}

func (r *sessionRepository) ListByPlayer(playerID string) ([]models.Session, error) {
	var sessions []models.Session
	err := r.db.Where("player_id = ?", playerID).
		Order("join_time DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *sessionRepository) ListActive() ([]models.Session, error) {
	var sessions []models.Session
	err := r.db.Where("leave_time IS NULL").
		Order("join_time DESC").
		Find(&sessions).Error
	return sessions, err
}
