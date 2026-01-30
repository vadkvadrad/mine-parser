package repo

import (
	"mine-parser/internal/models"

	"gorm.io/gorm"
)

type CommandRepository interface {
	Create(cmd *models.Command) error
	ListByPlayer(playerID string, limit int) ([]models.Command, error)
	ListByCommandName(name string) ([]models.Command, error)
	CountCommandsByPlayer(playerID string) (int64, error)
	GetMostUsedCommands(limit int) ([]CommandUsage, error)
}

type CommandUsage struct {
	CommandName string
	Count       int64
}

type commandRepository struct {
	db *gorm.DB
}

func NewCommandRepository(db *gorm.DB) CommandRepository {
	return &commandRepository{db: db}
}

func (r *commandRepository) Create(cmd *models.Command) error {
	return r.db.Create(cmd).Error
}

func (r *commandRepository) ListByPlayer(playerID string, limit int) ([]models.Command, error) {
	var commands []models.Command
	query := r.db.Joins("JOIN sessions ON commands.session_id = sessions.id").
		Where("sessions.player_id = ?", playerID).
		Order("commands.timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&commands).Error
	return commands, err
}

func (r *commandRepository) ListByCommandName(name string) ([]models.Command, error) {
	var commands []models.Command
	err := r.db.Where("command_name = ?", name).
		Order("timestamp DESC").
		Find(&commands).Error
	return commands, err
}

func (r *commandRepository) CountCommandsByPlayer(playerID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Command{}).
		Joins("JOIN sessions ON commands.session_id = sessions.id").
		Where("sessions.player_id = ?", playerID).
		Count(&count).Error
	return count, err
}

func (r *commandRepository) GetMostUsedCommands(limit int) ([]CommandUsage, error) {
	var results []struct {
		CommandName string `gorm:"column:command_name"`
		Count       int64  `gorm:"column:count"`
	}

	query := r.db.Model(&models.Command{}).
		Select("command_name, COUNT(*) as count").
		Group("command_name").
		Order("count DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Scan(&results).Error
	if err != nil {
		return nil, err
	}

	usages := make([]CommandUsage, len(results))
	for i, result := range results {
		usages[i] = CommandUsage{
			CommandName: result.CommandName,
			Count:       result.Count,
		}
	}

	return usages, nil
}
