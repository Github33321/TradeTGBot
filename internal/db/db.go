// TradeTGBot/internal/db/db.go
package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" // Драйвер PostgreSQL

	"TradeTGBot/internal/config"
)

// GlobalDB - глобальная переменная для экземпляра подключения к базе данных.
// Экспортируем её, чтобы пакет repository мог её использовать.
var GlobalDB *sql.DB

// InitDB инициализирует подключение к базе данных.
func InitDB(cfg config.DBConfig) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	log.Printf("Попытка подключения к БД: user=%s host=%s port=%s db=%s",
		cfg.User, cfg.Host, cfg.Port, cfg.Name)

	var err error
	GlobalDB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("ошибка при открытии соединения с БД: %w", err)
	}

	GlobalDB.SetMaxOpenConns(25)
	GlobalDB.SetMaxIdleConns(10)
	GlobalDB.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Пингуем базу данных для проверки соединения...")
	err = GlobalDB.Ping()
	if err != nil {
		GlobalDB.Close() // Закрываем соединение, если пинг не удался
		return fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	log.Println("Успешное подключение к базе данных!")
	return nil
}

// CloseDB закрывает соединение с базой данных.
func CloseDB() {
	if GlobalDB != nil {
		err := GlobalDB.Close()
		if err != nil {
			log.Printf("Внимание: ошибка при закрытии соединения с БД: %v", err)
		} else {
			log.Println("Соединение с базой данных закрыто.")
		}
	}
}
