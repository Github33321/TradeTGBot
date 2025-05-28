// TradeTGBot/internal/repository/repository.go
package repository

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"TradeTGBot/internal/db" // Используем GlobalDB из пакета db
)

// StockPrice represents a record in the stock_prices table
type StockPrice struct {
	ID        int
	Ticker    string
	Price     float64
	Timestamp time.Time
}

// Alert represents a user-defined alert
type Alert struct {
	ID        int
	Ticker    string
	Target    float64
	ChatID    int64
	Direction string // "up" or "down"
}

// SaveStockPrice сохраняет цену акции в базе данных.
func SaveStockPrice(ticker string, price float64) error {
	query := `INSERT INTO stock_prices (ticker, price) VALUES ($1, $2)`
	_, err := db.GlobalDB.Exec(query, ticker, price)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении цены акции %s (%.2f): %w", ticker, price, err)
	}
	log.Printf("Цена %s (%.2f) успешно сохранена в БД.", ticker, price)
	return nil
}

// GetAveragePrice получает среднюю цену акции за указанный период.
func GetAveragePrice(ticker string, duration time.Duration) (float64, error) {
	query := `
		SELECT AVG(price)
		FROM stock_prices
		WHERE ticker = $1 AND timestamp >= $2
	`
	startTime := time.Now().Add(-duration)

	var avgPrice sql.NullFloat64
	err := db.GlobalDB.QueryRow(query, ticker, startTime).Scan(&avgPrice)
	if err != nil {
		return 0, fmt.Errorf("ошибка при получении средней цены для %s: %w", ticker, err)
	}

	if !avgPrice.Valid {
		return 0, fmt.Errorf("нет данных для расчета средней цены %s за последние %s", ticker, duration)
	}

	return avgPrice.Float64, nil
}

// SaveAlert сохраняет новое оповещение пользователя в базе данных.
// (В текущей реализации алерты еще в памяти, но эта функция для будущего расширения)
func SaveAlert(alert Alert) error {
	// Пока алерты хранятся в памяти в pkg/bot, эта функция просто заглушка или для будущего использования с БД
	// Если бы алерты были в БД, тут был бы INSERT запрос
	log.Printf("DEBUG: Алерт для %s@%d (%.2f) был бы сохранен в БД", alert.Ticker, alert.ChatID, alert.Target)
	return nil
}

// GetActiveAlerts получает все активные оповещения из базы данных.
// (В текущей реализации алерты еще в памяти, но эта функция для будущего расширения)
func GetActiveAlerts() ([]Alert, error) {
	// Пока алерты хранятся в памяти в pkg/bot, эта функция просто заглушка
	// и должна быть заменена реальным SELECT запросом из БД
	log.Println("DEBUG: Получение активных алертов. Пока возвращаем пустой список, так как они в памяти.")
	return []Alert{}, nil // Здесь должен быть SELECT * FROM alerts
}

// DeleteAlert удаляет сработавший алерт из базы данных.
// (В текущей реализации алерты еще в памяти, но эта функция для будущего расширения)
func DeleteAlert(alert Alert) error {
	// Если бы алерты были в БД, тут был бы DELETE запрос по ID или другим полям
	log.Printf("DEBUG: Алерт для %s@%d (%.2f) был бы удален из БД", alert.Ticker, alert.ChatID, alert.Target)
	return nil
}
