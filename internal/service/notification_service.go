package service

import (
	"mine-parser/internal/models"
	"mine-parser/internal/repo"
)

type NotificationService interface {
	ToggleSubscription(chatID int64) (bool, error)
	GetSubscriptionStatus(chatID int64) (bool, error)
	GetAllSubscribers() ([]models.NotificationSubscription, error)
	// Методы для черного списка
	ToggleBlacklist(chatID int64, playerID string) (bool, error) // возвращает true если добавлен в черный список
	IsInBlacklist(chatID int64, playerID string) (bool, error)
	GetBlacklist(chatID int64) ([]models.NotificationBlacklist, error)
	ShouldNotify(chatID int64, playerID string) (bool, error) // проверяет, нужно ли отправлять уведомление
}

type notificationService struct {
	notificationRepo repo.NotificationRepository
}

func NewNotificationService(notificationRepo repo.NotificationRepository) NotificationService {
	return &notificationService{
		notificationRepo: notificationRepo,
	}
}

func (s *notificationService) ToggleSubscription(chatID int64) (bool, error) {
	subscription, err := s.notificationRepo.GetByChatID(chatID)
	if err != nil {
		return false, err
	}

	// Если подписки нет, создаем с enabled=true
	if subscription == nil {
		newEnabled := true
		err = s.notificationRepo.CreateOrUpdate(chatID, newEnabled)
		if err != nil {
			return false, err
		}
		return newEnabled, nil
	}

	// Переключаем статус
	newEnabled := !subscription.Enabled
	err = s.notificationRepo.CreateOrUpdate(chatID, newEnabled)
	if err != nil {
		return false, err
	}
	return newEnabled, nil
}

func (s *notificationService) GetSubscriptionStatus(chatID int64) (bool, error) {
	subscription, err := s.notificationRepo.GetByChatID(chatID)
	if err != nil {
		return false, err
	}
	// Если подписки нет, считаем что выключена
	if subscription == nil {
		return false, nil
	}
	return subscription.Enabled, nil
}

func (s *notificationService) GetAllSubscribers() ([]models.NotificationSubscription, error) {
	return s.notificationRepo.GetAllEnabled()
}

func (s *notificationService) ToggleBlacklist(chatID int64, playerID string) (bool, error) {
	isInBlacklist, err := s.notificationRepo.IsInBlacklist(chatID, playerID)
	if err != nil {
		return false, err
	}

	if isInBlacklist {
		// Удаляем из черного списка
		err = s.notificationRepo.RemoveFromBlacklist(chatID, playerID)
		return false, err
	} else {
		// Добавляем в черный список
		err = s.notificationRepo.AddToBlacklist(chatID, playerID)
		return true, err
	}
}

func (s *notificationService) IsInBlacklist(chatID int64, playerID string) (bool, error) {
	return s.notificationRepo.IsInBlacklist(chatID, playerID)
}

func (s *notificationService) GetBlacklist(chatID int64) ([]models.NotificationBlacklist, error) {
	return s.notificationRepo.GetBlacklist(chatID)
}

func (s *notificationService) ShouldNotify(chatID int64, playerID string) (bool, error) {
	// Проверяем, включены ли уведомления
	enabled, err := s.GetSubscriptionStatus(chatID)
	if err != nil || !enabled {
		return false, err
	}

	// Проверяем, не в черном списке ли игрок
	isInBlacklist, err := s.IsInBlacklist(chatID, playerID)
	if err != nil {
		return false, err
	}

	return !isInBlacklist, nil
}
