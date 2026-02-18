package migrations

import (
	"log"
	"mine-parser/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDB инициализирует соединение с БД и выполняет авто-миграцию
func InitDB(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Не удалось подключиться к БД: %v", err)
	}

	// Автоматическая миграция моделей
	err = db.AutoMigrate(&models.Player{}, &models.Session{}, &models.Command{}, &models.Advancement{}, &models.NotificationSubscription{}, &models.NotificationBlacklist{})
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	log.Println("Таблицы успешно созданы/обновлены.")
	return db
}

func ConnectDB(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Не удалось подключиться к БД: %v", err)
	}
	return db
}
