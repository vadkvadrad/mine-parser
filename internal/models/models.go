package models

import (
	"time"
)

// Player представляет игрока по UUID
type Player struct {
	ID        string    `gorm:"primaryKey;type:uuid;not null" json:"id"`
	Username  string    `gorm:"type:varchar(32);not null" json:"username"`
	FirstSeen time.Time `gorm:"not null" json:"first_seen"`
	LastSeen  time.Time `gorm:"not null" json:"last_seen"`
}

// Session — сессия подключения игрока
type Session struct {
	ID        uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	PlayerID  string     `gorm:"type:uuid;not null;index" json:"player_id"`
	JoinTime  time.Time  `gorm:"not null" json:"join_time"`
	LeaveTime *time.Time `gorm:"null" json:"leave_time,omitempty"`
	IPAddress string     `gorm:"type:varchar(45);not null" json:"ip_address"` // IPv6 совместимо
	EntityID  int        `gorm:"not null" json:"entity_id"`

	Player Player `gorm:"foreignKey:PlayerID;references:ID"`
}

// Command — выполненная команда
type Command struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID   uint      `gorm:"not null;index" json:"session_id"`
	Timestamp   time.Time `gorm:"not null" json:"timestamp"`
	Command     string    `gorm:"type:text;not null" json:"command"`
	CommandName string    `gorm:"type:varchar(64);not null;index" json:"command_name"`
	Args        string    `gorm:"type:text;null" json:"args,omitempty"`

	Session Session `gorm:"foreignKey:SessionID"`
}

// Advancement — полученное достижение
type Advancement struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PlayerID        string    `gorm:"type:uuid;not null;index" json:"player_id"`
	Timestamp       time.Time `gorm:"not null" json:"timestamp"`
	AdvancementName string    `gorm:"type:varchar(128);not null" json:"advancement_name"`

	Player Player `gorm:"foreignKey:PlayerID;references:ID"`
}

// NotificationSubscription — подписка на уведомления в Telegram
type NotificationSubscription struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatID    int64     `gorm:"uniqueIndex;not null" json:"chat_id"`
	Enabled   bool      `gorm:"default:false;not null" json:"enabled"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

// NotificationBlacklist — черный список игроков для уведомлений
type NotificationBlacklist struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatID    int64     `gorm:"index;not null" json:"chat_id"`
	PlayerID  string    `gorm:"type:uuid;index;not null" json:"player_id"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`

	Player Player `gorm:"foreignKey:PlayerID;references:ID"`
}
