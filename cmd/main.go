package main

import (
	"log"
	"mine-parser/internal/app"
	"sync"
)

func main() {
	log.Println("Запуск парсера логов Minecraft...")

	var wg sync.WaitGroup

	// Запускаем парсер логов
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.Parser()
	}()

	// Запускаем Telegram бота
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.TelegramBot()
	}()

	wg.Wait()
}
