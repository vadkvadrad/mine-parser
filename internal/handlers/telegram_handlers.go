package handlers

import (
	"fmt"
	"log"
	"mine-parser/internal/models"
	"mine-parser/internal/service"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// sendEditMessage отправляет редактирование сообщения, игнорируя ошибку "message is not modified"
func (h *TelegramHandlers) sendEditMessage(edit tgbotapi.EditMessageTextConfig) {
	_, err := h.bot.Send(edit)
	if err != nil {
		// Игнорируем ошибку "message is not modified" - это нормальная ситуация
		errStr := err.Error()
		if !strings.Contains(errStr, "message is not modified") {
			log.Printf("Ошибка при отправке сообщения: %v", err)
		}
	}
}

type TelegramHandlers struct {
	bot             *tgbotapi.BotAPI
	playerSvc       service.PlayerService
	commandSvc      service.CommandService
	advanceSvc      service.AdvancementService
	notificationSvc service.NotificationService
}

func NewTelegramHandlers(
	bot *tgbotapi.BotAPI,
	playerSvc service.PlayerService,
	commandSvc service.CommandService,
	advanceSvc service.AdvancementService,
	notificationSvc service.NotificationService,
) *TelegramHandlers {
	return &TelegramHandlers{
		bot:             bot,
		playerSvc:       playerSvc,
		commandSvc:      commandSvc,
		advanceSvc:      advanceSvc,
		notificationSvc: notificationSvc,
	}
}

func (h *TelegramHandlers) HandleMessage(message *tgbotapi.Message) {
	if !message.IsCommand() {
		return
	}

	switch message.Command() {
	case "start":
		h.sendMainMenu(message.Chat.ID, 0)
	}
}

func (h *TelegramHandlers) HandleCallback(callback *tgbotapi.CallbackQuery) {
	// Отвечаем на callback query немедленно, чтобы убрать индикатор загрузки
	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	if _, err := h.bot.Request(callbackConfig); err != nil {
		log.Printf("Ошибка при ответе на callback: %v", err)
	}

	data := callback.Data
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	// Обрабатываем callback асинхронно, чтобы не блокировать основной поток
	go func() {
		if data == "back:main" {
			h.sendMainMenu(chatID, messageID)
			return
		}

		if strings.HasPrefix(data, "player:") {
			playerID := strings.TrimPrefix(data, "player:")
			h.showPlayerInfo(chatID, messageID, playerID)
		} else if strings.HasPrefix(data, "advancements:") {
			playerID := strings.TrimPrefix(data, "advancements:")
			h.showAdvancements(chatID, messageID, playerID)
		} else if strings.HasPrefix(data, "commands:") {
			playerID := strings.TrimPrefix(data, "commands:")
			h.showCommands(chatID, messageID, playerID)
		} else if data == "online" {
			h.showOnlinePlayers(chatID, messageID)
		} else if data == "all_players" {
			h.showAllPlayers(chatID, messageID)
		} else if data == "connection_guide" {
			h.showConnectionGuide(chatID, messageID)
		} else if data == "world_map" {
			h.showWorldMap(chatID, messageID)
		} else if data == "notifications" {
			h.showNotificationsMenu(chatID, messageID)
		} else if data == "enable_notifications" {
			h.enableNotifications(chatID, messageID)
		} else if data == "disable_notifications" {
			h.disableNotifications(chatID, messageID)
		} else if data == "blacklist" {
			h.showBlacklist(chatID, messageID)
		} else if strings.HasPrefix(data, "blacklist_toggle:") {
			playerID := strings.TrimPrefix(data, "blacklist_toggle:")
			h.toggleBlacklistPlayer(chatID, messageID, playerID)
		} else if data == "back" {
			h.sendMainMenu(chatID, messageID)
		}
	}()
}

func (h *TelegramHandlers) sendMainMenu(chatID int64, messageID int) {
	text := "📊 Статистика сервера Minecraft\n\nВыберите раздел:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Онлайн игроки", "online"),
			tgbotapi.NewInlineKeyboardButtonData("📜 Все игроки", "all_players"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗺 Карта мира", "world_map"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📖 Инструкция по подключению", "connection_guide"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔔 Уведомления", "notifications"),
		),
	)

	var sentMsg tgbotapi.Chattable
	if messageID > 0 {
		// Редактируем существующее сообщение
		edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
		edit.ReplyMarkup = &keyboard
		sentMsg = edit
	} else {
		// Отправляем новое (для /start или первого вызова)
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = keyboard
		sentMsg = msg
	}
	if _, err := h.bot.Send(sentMsg); err != nil {
		log.Printf("Ошибка при отправке сообщения: %v", err)
	}
}

func (h *TelegramHandlers) showOnlinePlayers(chatID int64, messageID int) {
	players, err := h.playerSvc.ListOnlinePlayers()
	if err != nil {
		h.sendError(chatID, "Ошибка при получении списка игроков")
		return
	}

	if len(players) == 0 {
		text := "Нет игроков онлайн"
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
			),
		)
		edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
		edit.ReplyMarkup = &keyboard
		h.sendEditMessage(edit)
		return
	}

	h.showPlayerList(chatID, messageID, players, "Игроки онлайн")
}

func (h *TelegramHandlers) showAllPlayers(chatID int64, messageID int) {
	players, err := h.playerSvc.ListAllPlayers()
	if err != nil {
		h.sendError(chatID, "Ошибка при получении списка игроков")
		return
	}

	if len(players) == 0 {
		text := "Нет игроков в базе"
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
			),
		)
		edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
		edit.ReplyMarkup = &keyboard
		h.sendEditMessage(edit)
		return
	}

	h.showPlayerList(chatID, messageID, players, "Все игроки")
}

func (h *TelegramHandlers) showPlayerList(chatID int64, messageID int, players []models.Player, title string) {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, player := range players {
		button := tgbotapi.NewInlineKeyboardButtonData(player.Username, fmt.Sprintf("player:%s", player.ID))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	text := fmt.Sprintf("%s:\nВыберите игрока", title)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &keyboard
	h.sendEditMessage(edit)
}

func (h *TelegramHandlers) showPlayerInfo(chatID int64, messageID int, playerID string) {
	player, err := h.playerSvc.GetPlayerStats(playerID)
	if err != nil {
		h.sendError(chatID, "Ошибка при получении информации об игроке")
		return
	}

	isOnline, err := h.playerSvc.IsPlayerOnline(playerID)
	if err != nil {
		h.sendError(chatID, "Ошибка при проверке статуса игрока")
		return
	}

	var statusText string
	var lastSessionText string

	if isOnline {
		statusText = "🟢 Онлайн"
		lastSession, err := h.playerSvc.GetLastSession(playerID)
		if err == nil && lastSession != nil {
			lastSessionText = fmt.Sprintf("Время входа: %s", lastSession.JoinTime.Format("02.01.2006 15:04"))
		}
	} else {
		statusText = "🔴 Офлайн"
		lastSession, err := h.playerSvc.GetLastSession(playerID)
		if err == nil && lastSession != nil {
			if lastSession.LeaveTime != nil {
				lastSessionText = fmt.Sprintf("Последний вход: %s\nВремя выхода: %s",
					lastSession.JoinTime.Format("02.01.2006 15:04"),
					lastSession.LeaveTime.Format("02.01.2006 15:04"))
			} else {
				lastSessionText = fmt.Sprintf("Последний вход: %s",
					lastSession.JoinTime.Format("02.01.2006 15:04"))
			}
		}
	}

	// Форматируем общее время игры в часах
	totalHours := formatPlayTime(player.TotalPlayTime)

	text := fmt.Sprintf("👤 Игрок: %s\n%s\n%s\n⏱ Время на сервере: %s",
		player.Player.Username,
		statusText,
		lastSessionText,
		totalHours)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Достижения", fmt.Sprintf("advancements:%s", playerID)),
			tgbotapi.NewInlineKeyboardButtonData("Команды", fmt.Sprintf("commands:%s", playerID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
		),
	)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &keyboard
	h.sendEditMessage(edit)
}

func (h *TelegramHandlers) showAdvancements(chatID int64, messageID int, playerID string) {
	advancements, err := h.advanceSvc.GetPlayerAdvancements(playerID)
	if err != nil {
		h.sendError(chatID, "Ошибка при получении достижений")
		return
	}

	player, err := h.playerSvc.GetPlayerStats(playerID)
	if err != nil {
		h.sendError(chatID, "Ошибка при получении информации об игроке")
		return
	}

	if len(advancements) == 0 {
		text := fmt.Sprintf("🎯 Достижения игрока %s:\n\nНет достижений", player.Player.Username)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Назад к игроку", fmt.Sprintf("player:%s", playerID)),
			),
		)
		edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
		edit.ReplyMarkup = &keyboard
		h.sendEditMessage(edit)
		return
	}

	var advText strings.Builder
	advText.WriteString(fmt.Sprintf("🎯 Достижения игрока %s:\n\n", player.Player.Username))
	for i, adv := range advancements {
		advText.WriteString(fmt.Sprintf("%d. %s\n   Получено: %s\n\n",
			i+1,
			adv.AdvancementName,
			adv.Timestamp.Format("02.01.2006 15:04")))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад к игроку", fmt.Sprintf("player:%s", playerID)),
		),
	)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, advText.String())
	edit.ReplyMarkup = &keyboard
	h.sendEditMessage(edit)
}

func (h *TelegramHandlers) showCommands(chatID int64, messageID int, playerID string) {
	commands, err := h.commandSvc.GetCommandHistory(playerID, 50)
	if err != nil {
		h.sendError(chatID, "Ошибка при получении команд")
		return
	}

	player, err := h.playerSvc.GetPlayerStats(playerID)
	if err != nil {
		h.sendError(chatID, "Ошибка при получении информации об игроке")
		return
	}

	if len(commands) == 0 {
		text := fmt.Sprintf("⌨️ Команды игрока %s:\n\nНет команд", player.Player.Username)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Назад к игроку", fmt.Sprintf("player:%s", playerID)),
			),
		)
		edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
		edit.ReplyMarkup = &keyboard
		h.sendEditMessage(edit)
		return
	}

	var cmdText strings.Builder
	cmdText.WriteString(fmt.Sprintf("⌨️ Команды игрока %s:\n\n", player.Player.Username))
	for i, cmd := range commands {
		if i >= 20 { // Ограничиваем вывод 20 командами
			cmdText.WriteString(fmt.Sprintf("\n... и еще %d команд", len(commands)-20))
			break
		}
		cmdText.WriteString(fmt.Sprintf("%d. %s\n   Время: %s\n\n",
			i+1,
			cmd.Command,
			cmd.Timestamp.Format("02.01.2006 15:04")))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад к игроку", fmt.Sprintf("player:%s", playerID)),
		),
	)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, cmdText.String())
	edit.ReplyMarkup = &keyboard
	h.sendEditMessage(edit)
}

func (h *TelegramHandlers) showConnectionGuide(chatID int64, messageID int) {
	text := `📖 Инструкция по подключению к серверу

Добро пожаловать на наш сервер Minecraft!

Для подключения к серверу выполните следующие шаги:

1️⃣ Скачайте и установите Minecraft версии 1.21.4

2️⃣ Запустите игру и выберите "Сетевая игра" (Multiplayer)

3️⃣ Нажмите "Добавить сервер" (Add Server)

4️⃣ Введите IP-адрес сервера:
   🌐 89.169.161.207

5️⃣ Нажмите "Готово" и подключитесь к серверу

Удачной игры! 🎮`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
		),
	)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &keyboard
	h.sendEditMessage(edit)
}

func (h *TelegramHandlers) showWorldMap(chatID int64, messageID int) {
	text := `🗺 Карта мира

В этом разделе вы можете посмотреть карту мира, которая обновляется в реальном времени и показывает положение всех игроков на сервере.`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🌐 Открыть карту", "https://minecraft.shapedby.ru"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
		),
	)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &keyboard
	h.sendEditMessage(edit)
}

func (h *TelegramHandlers) sendError(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, "❌ "+text)
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Ошибка при отправке сообщения: %v", err)
	}
}

// formatPlayTime форматирует время игры в формат "X.Xч"
func formatPlayTime(duration time.Duration) string {
	hours := duration.Hours()
	// Округляем до одного знака после запятой
	return fmt.Sprintf("%.1fч", hours)
}

func (h *TelegramHandlers) showBlacklist(chatID int64, messageID int) {
	// Получаем всех игроков
	players, err := h.playerSvc.ListAllPlayers()
	if err != nil {
		h.sendError(chatID, "Ошибка при получении списка игроков")
		return
	}

	// Получаем черный список для этого чата
	blacklist, err := h.notificationSvc.GetBlacklist(chatID)
	if err != nil {
		h.sendError(chatID, "Ошибка при получении черного списка")
		return
	}

	// Создаем map для быстрой проверки
	blacklistMap := make(map[string]bool)
	for _, item := range blacklist {
		blacklistMap[item.PlayerID] = true
	}

	text := "🚫 Черный список\n\nЧерный список нужен для того, чтобы не получать уведомления, когда конкретный игрок заходит на сервер.\n\nВыберите игрока:"

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, player := range players {
		// Определяем иконку: 🔔 если не в черном списке, 🔕 если в черном списке
		icon := "🔔"
		if blacklistMap[player.ID] {
			icon = "🔕"
		}
		buttonText := fmt.Sprintf("%s %s", icon, player.Username)
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("blacklist_toggle:%s", player.ID))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Назад", "notifications"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &keyboard
	h.sendEditMessage(edit)
}

func (h *TelegramHandlers) toggleBlacklistPlayer(chatID int64, messageID int, playerID string) {
	// Переключаем статус в черном списке
	_, err := h.notificationSvc.ToggleBlacklist(chatID, playerID)
	if err != nil {
		h.sendError(chatID, "Ошибка при изменении черного списка")
		return
	}

	// Показываем обновленный черный список
	h.showBlacklist(chatID, messageID)
}

func (h *TelegramHandlers) showNotificationsMenu(chatID int64, messageID int) {
	status, err := h.notificationSvc.GetSubscriptionStatus(chatID)
	if err != nil {
		h.sendError(chatID, "Ошибка при получении статуса уведомлений")
		return
	}

	var statusText string
	var buttonText string
	var buttonData string

	if status {
		statusText = "🔔 Уведомления включены"
		buttonText = "Выключить уведомления"
		buttonData = "disable_notifications"
	} else {
		statusText = "🔕 Уведомления выключены"
		buttonText = "Включить уведомления"
		buttonData = "enable_notifications"
	}

	text := fmt.Sprintf("🔔 Уведомления\n\nУведомления приходят, когда кто-то заходит на сервер.\n\n%s", statusText)

	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(buttonText, buttonData),
	))

	// Кнопка черного списка показывается только когда уведомления включены
	if status {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Черный список", "blacklist"),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &keyboard
	h.sendEditMessage(edit)
}

func (h *TelegramHandlers) enableNotifications(chatID int64, messageID int) {
	enabled, err := h.notificationSvc.ToggleSubscription(chatID)
	if err != nil {
		log.Printf("Ошибка при включении уведомлений для чата %d: %v", chatID, err)
		h.sendError(chatID, "Ошибка при включении уведомлений")
		return
	}

	log.Printf("Уведомления для чата %d переключены, новый статус: %v", chatID, enabled)

	// Показываем обновленное меню уведомлений
	h.showNotificationsMenu(chatID, messageID)
}

func (h *TelegramHandlers) disableNotifications(chatID int64, messageID int) {
	enabled, err := h.notificationSvc.ToggleSubscription(chatID)
	if err != nil {
		log.Printf("Ошибка при выключении уведомлений для чата %d: %v", chatID, err)
		h.sendError(chatID, "Ошибка при выключении уведомлений")
		return
	}

	log.Printf("Уведомления для чата %d переключены, новый статус: %v", chatID, enabled)

	// Показываем обновленное меню уведомлений
	h.showNotificationsMenu(chatID, messageID)
}
