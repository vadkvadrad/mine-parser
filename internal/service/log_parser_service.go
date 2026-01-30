// log_parser_service.go
package service

import (
	"bufio"
	"fmt"
	"log"
	"mine-parser/internal/config"
	"os"
	"regexp"
	"strings"
	"time"
)

// LogParserService описывает сервис парсинга логов
type LogParserService interface {
	ProcessLogFile() error
	ProcessLogLine(line string) error
}

type logParserService struct {
	cfg              *config.Config
	playerSvc        PlayerService
	commandSvc       CommandService
	advancementSvc   AdvancementService
	usernameToUUID   map[string]string // кэш username → UUID
	currentSessionIP map[string]string // кэш username → IP (для момента входа)
}

// NewLogParserService создаёт новый парсер
func NewLogParserService(
	cfg *config.Config,
	playerSvc PlayerService,
	commandSvc CommandService,
	advancementSvc AdvancementService,
) LogParserService {
	s := &logParserService{
		cfg:              cfg,
		playerSvc:        playerSvc,
		commandSvc:       commandSvc,
		advancementSvc:   advancementSvc,
		usernameToUUID:   make(map[string]string),
		currentSessionIP: make(map[string]string),
	}

	players, err := s.playerSvc.ListAllPlayers()
	if err == nil {
		for _, p := range players {
			if p.Username != "" && p.ID != "" {
				s.usernameToUUID[p.Username] = p.ID
			}
		}
		log.Printf("Загружено %d известных username → UUID в кэш", len(players))
	} else {
		log.Printf("Не удалось загрузить игроков в кэш: %v", err)
	}
	return s
}

// ProcessLogFile читает файл по пути из конфига и парсит его
func (s *logParserService) ProcessLogFile() error {
	path := s.cfg.App.ParsePath
	if path == "" {
		return fmt.Errorf("parse_path не задан в конфигурации")
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("не удалось открыть лог-файл %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if err := s.ProcessLogLine(line); err != nil {
			log.Printf("Ошибка на строке %d: %v\n  Строка: %s", lineNum, err, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка при чтении файла: %w", err)
	}

	log.Printf("Файл %s успешно обработан (%d строк)", path, lineNum)
	return nil
}

// ProcessLogLine парсит одну строку лога
func (s *logParserService) ProcessLogLine(line string) error {
	// Регулярка для извлечения времени и сообщения
	re := regexp.MustCompile(`^\[(\d{2}:\d{2}:\d{2})\] \[([^\]]+)\]: (.+)$`)
	matches := re.FindStringSubmatch(line)
	if len(matches) != 4 {
		return nil // игнорируем нераспознанные строки
	}

	//timestampStr := matches[1]
	component := matches[2]
	message := matches[3]

	// Парсим время (предполагаем текущую дату; для продакшена — лучше использовать ротацию логов с датой)
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return fmt.Errorf("не удалось загрузить локацию: %w", err)
	}
	now := time.Now().In(location)

	// // Парсим только время из лога
	// timeOnly, err := time.Parse("15:04:05", timestampStr)
	// if err != nil {
	// 	return fmt.Errorf("неверный формат времени: %w", err)
	// }

	// Создаём полное время на основе текущей даты + время из лога
	timestamp := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour()+3,
		now.Minute(),
		0,
		0,
		now.Location(),
	)

	// 1. Обработка UUID
	if strings.Contains(component, "User Authenticator") && strings.Contains(message, "UUID of player") {
		return s.handleUUIDLine(message, timestamp)
	}

	// 2. Обработка входа
	if strings.HasSuffix(message, " joined the game") {
		username := strings.TrimSuffix(message, " joined the game")
		return s.playerSvc.RegisterLogin(s.usernameToUUID[username], username, s.currentSessionIP[username], 0, timestamp)
	}

	// 3. Обработка выхода
	if strings.HasSuffix(message, " left the game") {
		username := strings.TrimSuffix(message, " left the game")
		playerID := s.usernameToUUID[username]
		if playerID == "" {
			// Пытаемся восстановить из БД
			player, err := s.playerSvc.GetPlayerByUsername(username) // новый метод
			if err == nil && player != nil && player.ID != "" {
				playerID = player.ID
				s.usernameToUUID[username] = playerID // кэшируем на будущее
				log.Printf("Восстановили UUID из БД для %s → %s", username, playerID)
			} else {
				log.Printf("WARN: Нет UUID для %s даже в БД → используем username временно", username)
				playerID = username
			}
		}
		return s.playerSvc.RegisterLogout(playerID, timestamp)
	}

	// 4. Обработка команды
	if strings.Contains(message, " issued server command: ") {
		parts := strings.SplitN(message, " issued server command: ", 2)
		if len(parts) == 2 {
			username := parts[0]
			fullCmd := parts[1]
			playerID := s.usernameToUUID[username]
			if playerID == "" {
				playerID = username
			}
			return s.commandSvc.LogCommand(playerID, fullCmd, timestamp)
		}
	}

	// 5. Обработка достижения
	if strings.Contains(message, " has made the advancement [") {
		parts := strings.SplitN(message, " has made the advancement [", 2)
		if len(parts) == 2 {
			username := parts[0]
			advName := strings.TrimSuffix(parts[1], "]")
			playerID := s.usernameToUUID[username]
			if playerID == "" {
				playerID = username
			}
			return s.advancementSvc.GrantAdvancement(playerID, advName, timestamp)
		}
	}

	// 6. Извлечение IP при входе (из строки вида "vadkvad[/109.173.122.70:34284] logged in...")
	if strings.Contains(message, " logged in with entity id ") {
		// Пример: "vadkvad[/109.173.122.70:34284] logged in with entity id 46 at ..."
		ipRe := regexp.MustCompile(`^([^\[]+)\[/([0-9.:]+)\]`)
		ipMatch := ipRe.FindStringSubmatch(message)
		if len(ipMatch) == 3 {
			username := ipMatch[1]
			ipPort := ipMatch[2]
			// Обрезаем порт
			ip := strings.Split(ipPort, ":")[0]
			s.currentSessionIP[username] = ip
		}
	}

	return nil
}

// handleUUIDLine обрабатывает строку с UUID
func (s *logParserService) handleUUIDLine(message string, timestamp time.Time) error {
	// Пример: "UUID of player vadkvad is d731c558-db08-3fae-bbc3-c3e2f8051bf9"
	parts := strings.Split(message, " ")
	if len(parts) >= 6 && parts[0] == "UUID" && parts[1] == "of" && parts[2] == "player" {
		username := parts[3]
		uuid := parts[5]
		s.usernameToUUID[username] = uuid
		// Просто сохраняем UUID в кэш, игрок создастся при входе через RegisterLogin
		return nil
	}
	return nil
}
