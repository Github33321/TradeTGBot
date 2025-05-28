// TradeTGBot/internal/config/config.go
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config хранит всю конфигурацию приложения
type Config struct {
	BotToken string
	DB       DBConfig
}

// DBConfig хранит конфигурацию для подключения к базе данных
type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

// LoadConfig загружает конфигурацию из переменных окружения и .env файла
func LoadConfig() (*Config, error) {
	// Загружаем переменные из .env файла.
	// godotenv.Load() ищет .env в текущей директории или выше по дереву.
	// Если вы запускаете `go run cmd/bot/main.go` из корня проекта,
	// то он найдет `.env` в корне.
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) { // Не считаем ошибкой, если .env не найден, но есть другие ошибки
		return nil, fmt.Errorf("ошибка при загрузке .env файла: %w", err)
	}

	cfg := &Config{
		BotToken: os.Getenv("BOT_TOKEN"),
		DB: DBConfig{
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			Name:     os.Getenv("DB_NAME"),
		},
	}

	// Проверяем, что все критически важные переменные загружены
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN не установлен в переменных окружения")
	}
	if cfg.DB.User == "" || cfg.DB.Password == "" || cfg.DB.Host == "" || cfg.DB.Port == "" || cfg.DB.Name == "" {
		return nil, fmt.Errorf("одна или несколько переменных окружения БД не установлены (DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME)")
	}

	return cfg, nil
}
