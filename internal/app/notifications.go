package app

import (
	"fmt"
	"log"
	"mine-parser/internal/service"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// PlayerLoginEvent событие входа игрока на сервер
type PlayerLoginEvent struct {
	PlayerID  string
	Username  string
	Timestamp int64
}

// Глобальный канал для событий входа игроков
var playerLoginEvents chan PlayerLoginEvent

// InitPlayerLoginEvents инициализирует канал событий
func InitPlayerLoginEvents() chan PlayerLoginEvent {
	if playerLoginEvents == nil {
		playerLoginEvents = make(chan PlayerLoginEvent, 100) // Буферизованный канал
	}
	return playerLoginEvents
}

// SendPlayerLoginEvent отправляет событие входа игрока (безопасно для использования из parser)
func SendPlayerLoginEvent(playerID, username string) {
	if playerLoginEvents != nil {
		select {
		case playerLoginEvents <- PlayerLoginEvent{
			PlayerID:  playerID,
			Username:  username,
			Timestamp: 0, // Можно добавить timestamp если нужно
		}:
		default:
			// Если канал переполнен, просто игнорируем (не блокируем parser)
			log.Printf("Канал событий переполнен, событие пропущено")
		}
	}
}

// init устанавливает глобальную функцию отправки событий
func init() {
	service.SetGlobalLoginEventSender(SendPlayerLoginEvent)
}

// NotificationSender отправляет уведомления о входе игроков
type NotificationSender struct {
	bot             *tgbotapi.BotAPI
	notificationSvc service.NotificationService
	eventChan       chan PlayerLoginEvent
	wg              sync.WaitGroup
	stopChan        chan struct{}
}

// StartNotificationSender запускает сервис отправки уведомлений
func StartNotificationSender(
	bot *tgbotapi.BotAPI,
	notificationSvc service.NotificationService,
) {
	eventChan := InitPlayerLoginEvents()

	sender := &NotificationSender{
		bot:             bot,
		notificationSvc: notificationSvc,
		eventChan:       eventChan,
		stopChan:        make(chan struct{}),
	}

	sender.wg.Add(1)
	go sender.processEvents()
}

// processEvents обрабатывает события входа игроков
func (ns *NotificationSender) processEvents() {
	defer ns.wg.Done()

	for {
		select {
		case event := <-ns.eventChan:
			ns.handlePlayerLogin(event)
		case <-ns.stopChan:
			return
		}
	}
}

// handlePlayerLogin обрабатывает событие входа игрока
func (ns *NotificationSender) handlePlayerLogin(event PlayerLoginEvent) {
	// Получаем всех активных подписчиков
	subscribers, err := ns.notificationSvc.GetAllSubscribers()
	if err != nil {
		log.Printf("Ошибка при получении подписчиков: %v", err)
		return
	}

	if len(subscribers) == 0 {
		return
	}

	// Формируем сообщение
	message := fmt.Sprintf("🟢 Игрок %s зашел на сервер", event.Username)

	// Отправляем уведомления асинхронно каждому подписчику
	for _, subscriber := range subscribers {
		go func(chatID int64) {
			// Проверяем, нужно ли отправлять уведомление (не в черном списке)
			shouldNotify, err := ns.notificationSvc.ShouldNotify(chatID, event.PlayerID)
			if err != nil {
				log.Printf("Ошибка при проверке черного списка для чата %d: %v", chatID, err)
				return
			}

			if !shouldNotify {
				return // Игрок в черном списке или уведомления выключены
			}

			msg := tgbotapi.NewMessage(chatID, message)
			if _, err := ns.bot.Send(msg); err != nil {
				log.Printf("Ошибка при отправке уведомления в чат %d: %v", chatID, err)
			}
		}(subscriber.ChatID)
	}
}

// Stop останавливает сервис отправки уведомлений
func (ns *NotificationSender) Stop() {
	close(ns.stopChan)
	ns.wg.Wait()
}

// GetNotificationSender возвращает экземпляр отправителя (для использования в tg.go)
func GetNotificationSender() *NotificationSender {
	return nil // Не используется, но может понадобиться для graceful shutdown
}
