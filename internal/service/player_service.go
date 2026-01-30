package service

import (
	"mine-parser/internal/models"
	"mine-parser/internal/repo"
	"time"
)

type PlayerService interface {
	RegisterLogin(playerID, username string, ip string, entityID int, timestamp time.Time) error
	RegisterLogout(playerID string, timestamp time.Time) error
	GetPlayerStats(playerID string) (*PlayerStats, error)
	ListOnlinePlayers() ([]models.Player, error)
	ListAllPlayers() ([]models.Player, error)
	IsPlayerOnline(playerID string) (bool, error)
	GetLastSession(playerID string) (*models.Session, error)
	GetPlayerByUsername(username string) (*models.Player, error)
}

// PlayerStats — DTO для агрегированных данных
type PlayerStats struct {
	Player        models.Player
	TotalPlayTime time.Duration
	SessionCount  int
	CommandsUsed  int64
	Advancements  []models.Advancement
}

type playerService struct {
	playerRepo  repo.PlayerRepository
	sessionRepo repo.SessionRepository
	commandRepo repo.CommandRepository
	advanceRepo repo.AdvancementRepository
}

func NewPlayerService(
	playerRepo repo.PlayerRepository,
	sessionRepo repo.SessionRepository,
	commandRepo repo.CommandRepository,
	advanceRepo repo.AdvancementRepository,
) PlayerService {
	return &playerService{
		playerRepo:  playerRepo,
		sessionRepo: sessionRepo,
		commandRepo: commandRepo,
		advanceRepo: advanceRepo,
	}
}

func (s *playerService) RegisterLogin(playerID, username string, ip string, entityID int, timestamp time.Time) error {
	// Создаем или обновляем игрока
	_, err := s.playerRepo.GetOrCreate(playerID, username, timestamp)
	if err != nil {
		return err
	}

	// Проверяем, нет ли уже активной сессии
	activeSession, err := s.sessionRepo.GetActiveSessionByPlayer(playerID)
	if err == nil && activeSession != nil {
		// Если есть активная сессия, закрываем её перед созданием новой
		_ = s.sessionRepo.CloseSession(activeSession.ID, timestamp)
	}

	// Создаем новую сессию
	session := &models.Session{
		PlayerID:  playerID,
		JoinTime:  timestamp,
		LeaveTime: nil,
		IPAddress: ip,
		EntityID:  entityID,
	}

	return s.sessionRepo.Create(session)
}

func (s *playerService) RegisterLogout(playerID string, timestamp time.Time) error {
	// Обновляем last_seen игрока
	if err := s.playerRepo.UpdateLastSeen(playerID, timestamp); err != nil {
		return err
	}

	// Находим активную сессию и закрываем её
	activeSession, err := s.sessionRepo.GetActiveSessionByPlayer(playerID)
	if err != nil {
		// Если активной сессии нет, это не критично (может быть уже закрыта)
		return nil
	}

	return s.sessionRepo.CloseSession(activeSession.ID, timestamp)
}

func (s *playerService) GetPlayerStats(playerID string) (*PlayerStats, error) {
	// Получаем игрока
	player, err := s.playerRepo.FindByID(playerID)
	if err != nil {
		return nil, err
	}

	// Получаем все сессии игрока
	sessions, err := s.sessionRepo.ListByPlayer(playerID)
	if err != nil {
		return nil, err
	}

	// Вычисляем общее время игры
	var totalPlayTime time.Duration
	// Коррекция часового пояса: сервер на -3 часа, нужно добавить 3 часа
	timeZoneCorrection := 3 * time.Hour
	for _, session := range sessions {
		if session.LeaveTime != nil {
			totalPlayTime += session.LeaveTime.Sub(session.JoinTime)
		} else {
			// Если сессия еще активна, считаем до текущего времени
			// Сначала добавляем 3 часа к текущему времени, потом вычитаем время входа
			currentTimeCorrected := time.Now().Add(timeZoneCorrection)
			totalPlayTime += currentTimeCorrected.Sub(session.JoinTime)
		}
	}

	// Получаем количество команд
	commandsCount, err := s.commandRepo.CountCommandsByPlayer(playerID)
	if err != nil {
		return nil, err
	}

	// Получаем достижения
	advancements, err := s.advanceRepo.ListByPlayer(playerID)
	if err != nil {
		return nil, err
	}

	return &PlayerStats{
		Player:        *player,
		TotalPlayTime: totalPlayTime,
		SessionCount:  len(sessions),
		CommandsUsed:  commandsCount,
		Advancements:  advancements,
	}, nil
}

func (s *playerService) ListOnlinePlayers() ([]models.Player, error) {
	// Получаем все активные сессии
	activeSessions, err := s.sessionRepo.ListActive()
	if err != nil {
		return nil, err
	}

	// Собираем уникальных игроков
	playerMap := make(map[string]bool)
	var playerIDs []string
	for _, session := range activeSessions {
		if !playerMap[session.PlayerID] {
			playerMap[session.PlayerID] = true
			playerIDs = append(playerIDs, session.PlayerID)
		}
	}

	// Получаем данные игроков
	var players []models.Player
	for _, playerID := range playerIDs {
		player, err := s.playerRepo.FindByID(playerID)
		if err != nil {
			// Пропускаем игроков, которых не удалось найти
			continue
		}
		players = append(players, *player)
	}

	return players, nil
}

func (s *playerService) GetPlayerByUsername(username string) (*models.Player, error) {
	return s.playerRepo.FindByUsername(username)
}

func (s *playerService) ListAllPlayers() ([]models.Player, error) {
	return s.playerRepo.ListAll()
}

func (s *playerService) IsPlayerOnline(playerID string) (bool, error) {
	activeSession, err := s.sessionRepo.GetActiveSessionByPlayer(playerID)
	if err != nil {
		return false, err
	}
	return activeSession != nil, nil
}

func (s *playerService) GetLastSession(playerID string) (*models.Session, error) {
	sessions, err := s.sessionRepo.ListByPlayer(playerID)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	return &sessions[0], nil
}
