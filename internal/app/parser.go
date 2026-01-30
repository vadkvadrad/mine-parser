package app

import (
	"bufio"
	"context"
	"log"
	"mine-parser/internal/config"
	"mine-parser/internal/migrations"
	"mine-parser/internal/repo"
	"mine-parser/internal/service"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Parser() {
	// 1. Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalln("Failed to load config:", err)
	}
	if cfg.App.ParsePath == "" {
		log.Fatal("parse_path не задан в конфигурации")
	}

	// 2. Инициализация БД
	dbConn := migrations.InitDB(cfg.Db.Dsn)

	// 3. Репозитории
	playerRepo := repo.NewPlayerRepository(dbConn)
	sessionRepo := repo.NewSessionRepository(dbConn)
	commandRepo := repo.NewCommandRepository(dbConn)
	advancementRepo := repo.NewAdvancementRepository(dbConn)

	// 4. Сервисы
	playerSvc := service.NewPlayerService(playerRepo, sessionRepo, commandRepo, advancementRepo)
	commandSvc := service.NewCommandService(commandRepo, sessionRepo)
	advancementSvc := service.NewAdvancementService(advancementRepo)

	parser := service.NewLogParserService(cfg, playerSvc, commandSvc, advancementSvc)

	// 5. Настройка graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 6. Запуск парсера
	errCh := make(chan error, 1)
	go func() {
		errCh <- startTailing(ctx, cfg.App.ParsePath, parser)
	}()

	// 7. Ожидание завершения
	select {
	case <-ctx.Done():
		log.Println("Получен сигнал завершения, останавливаем парсинг...")
	case err := <-errCh:
		if err != nil {
			log.Fatalf("Ошибка при чтении лога: %v", err)
		}
	}

	// Дополнительная задержка для завершения обработки
	time.Sleep(500 * time.Millisecond)
	log.Println("Приложение завершено.")
}

func startTailing(ctx context.Context, filePath string, parser service.LogParserService) error {
	var file *os.File
	var lastPos int64
	var currentInode uint64

	openFile := func() error {
		if file != nil {
			file.Close()
		}
		var err error
		file, err = os.Open(filePath)
		if err != nil {
			return err
		}

		stat, err := file.Stat()
		if err != nil {
			file.Close()
			return err
		}

		// При первом открытии читаем с конца
		if currentInode == 0 {
			lastPos = stat.Size()
		} else {
			lastPos = 0 // Новый файл после ротации
		}

		currentInode = stat.Sys().(*syscall.Stat_t).Ino
		file.Seek(lastPos, 0)
		return nil
	}

	if err := openFile(); err != nil {
		return err
	}
	defer file.Close()

	log.Printf("Начинаю отслеживание лога: %s", filePath)

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Остановка чтения лога...")
			return nil
		case <-ticker.C:
			// Проверяем ротацию по inode
			newStat, err := os.Stat(filePath)
			if err != nil {
				log.Printf("Ошибка проверки файла: %v", err)
				continue
			}

			newInode := newStat.Sys().(*syscall.Stat_t).Ino
			if newInode != currentInode {
				log.Println("Обнаружена ротация файла, переоткрываю...")
				if err := openFile(); err != nil {
					log.Printf("Ошибка переоткрытия файла: %v", err)
					continue
				}
				continue
			}

			// Читаем новые данные
			if newStat.Size() > lastPos {
				file.Seek(lastPos, 0)
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := scanner.Text()
					if err := parser.ProcessLogLine(line); err != nil {
						log.Printf("Ошибка обработки строки: %v\n  Строка: %s", err, line)
					}
				}
				if err := scanner.Err(); err != nil {
					log.Printf("Ошибка чтения файла: %v", err)
				}
				lastPos, _ = file.Seek(0, 1)
			}
		}
	}
}
