package config

import (
	"errors"
	// "fmt"
	"log"
	"os"
	// "path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	App AppConfig
	Db  DbConfig
	Tg  TelegramCongig
}

type DbConfig struct {
	Dsn string
}

type AppConfig struct {
	Port      string
	ParsePath string
}

type TelegramCongig struct {
	Token string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден — используем переменные окружения")
	}

	config := &Config{
		App: AppConfig{
			Port:      getEnv("PORT", "8081"),
			ParsePath: getEnv("LOG_PATH", ""),
		},
		Db: DbConfig{
			Dsn: getEnv("DATABASE_URL", ""),
		},
		Tg: TelegramCongig{
			Token: getEnv("TG_TOKEN", ""),
		},
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.Db.Dsn == "" {
		return errors.New("DSN is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// func dir(envFile string) string {
// 	currentDir, err := os.Getwd()
// 	if err != nil {
// 		panic(err)
// 	}

// 	for {
// 		goModPath := filepath.Join(currentDir, "go.mod")
// 		if _, err := os.Stat(goModPath); err == nil {
// 			break
// 		}

// 		parent := filepath.Dir(currentDir)
// 		if parent == currentDir {
// 			panic(fmt.Errorf("go.mod not found"))
// 		}
// 		currentDir = parent
// 	}

// 	return filepath.Join(currentDir, envFile)
// }
