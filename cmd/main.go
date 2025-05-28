// TradeTGBot/cmd/main.go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"TradeTGBot/internal/analyzer"
	"TradeTGBot/internal/config"
	"TradeTGBot/internal/db"
	"TradeTGBot/pkg/bot"
	"TradeTGBot/pkg/stocks"
)

func main() {
	// 1. Загрузка конфигурации приложения
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Критическая ошибка: не удалось загрузить конфигурацию: %v", err)
	}
	log.Println("Конфигурация успешно загружена.")

	// 2. Инициализация подключения к базе данных
	err = db.InitDB(cfg.DB)
	if err != nil {
		log.Fatalf("Критическая ошибка: не удалось подключиться к базе данных: %v", err)
	}
	defer db.CloseDB() // Гарантированное закрытие соединения с БД

	// 3. Инициализация парсера (colly.Collector)
	collector := stocks.InitCollector()
	log.Println("Инициализация парсера (colly.Collector)...")
	_ = collector.Visit("https://ru.investing.com") // Первый визит для инициализации куки
	time.Sleep(2 * time.Second)                     // Дать время для установки куки

	// Инициализация экземпляра бота для анализатора
	// ОБРАБАТЫВАЕМ ВОЗВРАЩАЕМОЕ ЗНАЧЕНИЕ tgbotapi.NewBotAPI
	analysisBot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("Ошибка инициализации BotAPI для анализатора: %v", err)
	}

	// 4. Инициализация и запуск сервиса анализа цен (для LKOH)
	// ВАЖНО: Замените YOUR_CHAT_ID на реальный ID чата, куда бот должен отправлять уведомления.
	// Это может быть ваш личный ChatID.
	lKohAnalyzer := analyzer.NewPriceAnalyzer(
		analysisBot, // <-- Теперь передаем уже созданный экземпляр
		collector,
		int64(964949247), // <--- ЗАМЕНИТЕ НА ВАШ АЙДИ ЧАТА!
		10*time.Second,
		5*time.Minute,
		0.42,
	)
	lKohAnalyzer.StartAnalysis() // Запускаем горутину анализа цен

	// 5. Инициализация и запуск Telegram-бота
	botService, err := bot.NewBotService(cfg.BotToken, collector) // Передаем collector боту
	if err != nil {
		log.Fatalf("Ошибка инициализации Telegram-бота: %v", err)
	}
	go botService.StartPolling() // Запускаем опрос Telegram API в отдельной горутине

	log.Println("Приложение запущено. Ожидание сигналов завершения (Ctrl+C)...")

	// Ожидание сигнала завершения (например, Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Блокируем main горутину до получения сигнала

	log.Println("Получен сигнал завершения. Завершение работы приложения...")
}
