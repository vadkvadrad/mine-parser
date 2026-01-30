package app

import (
	"log"
	"mine-parser/internal/config"
	"mine-parser/internal/handlers"
	"mine-parser/internal/migrations"
	"mine-parser/internal/repo"
	"mine-parser/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TelegramBot() {
	// 1. Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load config for Telegram bot: %v", err)
		return
	}

	if cfg.Tg.Token == "" {
		log.Println("TG_TOKEN не задан, Telegram бот не запущен")
		return
	}

	// 2. Инициализация БД
	dbConn := migrations.ConnectDB(cfg.Db.Dsn)

	// 3. Репозитории
	playerRepo := repo.NewPlayerRepository(dbConn)
	sessionRepo := repo.NewSessionRepository(dbConn)
	commandRepo := repo.NewCommandRepository(dbConn)
	advancementRepo := repo.NewAdvancementRepository(dbConn)

	// 4. Сервисы
	playerSvc := service.NewPlayerService(playerRepo, sessionRepo, commandRepo, advancementRepo)
	commandSvc := service.NewCommandService(commandRepo, sessionRepo)
	advancementSvc := service.NewAdvancementService(advancementRepo)

	// 5. Создание бота
	bot, err := tgbotapi.NewBotAPI(cfg.Tg.Token)
	if err != nil {
		log.Printf("Не удалось создать бота: %v", err)
		return
	}

	bot.Debug = false
	log.Printf("Авторизован как %s", bot.Self.UserName)

	// 6. Создание хендлеров
	telegramHandlers := handlers.NewTelegramHandlers(bot, playerSvc, commandSvc, advancementSvc)

	// 7. Настройка обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// 8. Обработка обновлений
	for update := range updates {
		if update.Message != nil {
			go telegramHandlers.HandleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			go telegramHandlers.HandleCallback(update.CallbackQuery)
		}
	}
}
