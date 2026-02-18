package repo

import (
	"mine-parser/internal/models"

	"gorm.io/gorm"
)

type NotificationRepository interface {
	CreateOrUpdate(chatID int64, enabled bool) error
	GetByChatID(chatID int64) (*models.NotificationSubscription, error)
	GetAllEnabled() ([]models.NotificationSubscription, error)
	Delete(chatID int64) error
	// Методы для черного списка
	AddToBlacklist(chatID int64, playerID string) error
	RemoveFromBlacklist(chatID int64, playerID string) error
	IsInBlacklist(chatID int64, playerID string) (bool, error)
	GetBlacklist(chatID int64) ([]models.NotificationBlacklist, error)
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) CreateOrUpdate(chatID int64, enabled bool) error {
	var subscription models.NotificationSubscription
	err := r.db.Where("chat_id = ?", chatID).First(&subscription).Error

	if err == gorm.ErrRecordNotFound {
		// Создаем новую запись
		subscription = models.NotificationSubscription{
			ChatID:  chatID,
			Enabled: enabled,
		}
		return r.db.Create(&subscription).Error
	} else if err != nil {
		return err
	}

	// Обновляем существующую запись
	subscription.Enabled = enabled
	return r.db.Save(&subscription).Error
}

func (r *notificationRepository) GetByChatID(chatID int64) (*models.NotificationSubscription, error) {
	var subscription models.NotificationSubscription
	err := r.db.Where("chat_id = ?", chatID).First(&subscription).Error
	if err != nil {
		// Если запись не найдена, это нормально - подписки еще нет
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &subscription, nil
}

func (r *notificationRepository) GetAllEnabled() ([]models.NotificationSubscription, error) {
	var subscriptions []models.NotificationSubscription
	err := r.db.Where("enabled = ?", true).Find(&subscriptions).Error
	return subscriptions, err
}

func (r *notificationRepository) Delete(chatID int64) error {
	return r.db.Where("chat_id = ?", chatID).Delete(&models.NotificationSubscription{}).Error
}

func (r *notificationRepository) AddToBlacklist(chatID int64, playerID string) error {
	blacklist := &models.NotificationBlacklist{
		ChatID:   chatID,
		PlayerID: playerID,
	}
	// Используем FirstOrCreate, чтобы не создавать дубликаты
	return r.db.Where("chat_id = ? AND player_id = ?", chatID, playerID).
		FirstOrCreate(blacklist).Error
}

func (r *notificationRepository) RemoveFromBlacklist(chatID int64, playerID string) error {
	return r.db.Where("chat_id = ? AND player_id = ?", chatID, playerID).
		Delete(&models.NotificationBlacklist{}).Error
}

func (r *notificationRepository) IsInBlacklist(chatID int64, playerID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.NotificationBlacklist{}).
		Where("chat_id = ? AND player_id = ?", chatID, playerID).
		Count(&count).Error
	return count > 0, err
}

func (r *notificationRepository) GetBlacklist(chatID int64) ([]models.NotificationBlacklist, error) {
	var blacklist []models.NotificationBlacklist
	err := r.db.Where("chat_id = ?", chatID).Find(&blacklist).Error
	return blacklist, err
}
