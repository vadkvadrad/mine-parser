package service

import (
	"errors"
	"mine-parser/internal/models"
	"mine-parser/internal/repo"
	"strings"
	"time"
)

type CommandService interface {
	LogCommand(playerID string, fullCommand string, timestamp time.Time) error
	GetCommandHistory(playerID string, limit int) ([]models.Command, error)
	GetMostUsedCommands(limit int) ([]CommandUsage, error)
}

type CommandUsage struct {
	CommandName string
	Count       int64
}

type commandService struct {
	commandRepo repo.CommandRepository
	sessionRepo repo.SessionRepository
}

func NewCommandService(
	commandRepo repo.CommandRepository,
	sessionRepo repo.SessionRepository,
) CommandService {
	return &commandService{
		commandRepo: commandRepo,
		sessionRepo: sessionRepo,
	}
}

func (s *commandService) LogCommand(playerID string, fullCommand string, timestamp time.Time) error {
	// Находим активную сессию игрока
	activeSession, err := s.sessionRepo.GetActiveSessionByPlayer(playerID)
	if err != nil {
		// Ошибка БД - возвращаем её
		return err
	}
	if activeSession == nil {
		// Если активной сессии нет, это не критично - возможно игрок еще не залогинился
		// или сессия была закрыта. Просто пропускаем команду.
		return nil
	}

	// Парсим команду: извлекаем имя команды и аргументы
	parts := strings.Fields(fullCommand)
	if len(parts) == 0 {
		return errors.New("пустая команда")
	}

	commandName := parts[0]
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	// Создаем запись о команде
	cmd := &models.Command{
		SessionID:   activeSession.ID,
		Timestamp:   timestamp,
		Command:     fullCommand,
		CommandName: commandName,
		Args:        args,
	}

	return s.commandRepo.Create(cmd)
}

func (s *commandService) GetCommandHistory(playerID string, limit int) ([]models.Command, error) {
	return s.commandRepo.ListByPlayer(playerID, limit)
}

func (s *commandService) GetMostUsedCommands(limit int) ([]CommandUsage, error) {
	usages, err := s.commandRepo.GetMostUsedCommands(limit)
	if err != nil {
		return nil, err
	}

	// Конвертируем из репозитория в сервисный тип
	result := make([]CommandUsage, len(usages))
	for i, u := range usages {
		result[i] = CommandUsage{
			CommandName: u.CommandName,
			Count:       u.Count,
		}
	}

	return result, nil
}
