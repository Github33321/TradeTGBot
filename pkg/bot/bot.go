// TradeTGBot/pkg/bot/bot.go
package bot

import (
	"TradeTGBot/pkg/stocks"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly" // Импортируем colly, так как InitCollector возвращает *colly.Collector
)

// Alert - структура для оповещения (пока в памяти).
// В будущем это может быть строка в БД, и мы будем использовать repository.Alert
type Alert struct {
	Ticker    string
	Target    float64
	ChatID    int64
	Direction string
}

// bot.go будет управлять локальным списком алертов,
// если они не хранятся в БД. Если будут в БД, то эту переменную убрать.
var userAlerts []Alert // Переименовал, чтобы не конфликтовать с repository.Alert

// BotService инкапсулирует логику бота и зависимости.
type BotService struct {
	bot       *tgbotapi.BotAPI
	collector *colly.Collector // Нужно передать, чтобы бот мог запрашивать данные
}

// NewBotService создает новый экземпляр BotService.
func NewBotService(token string, collector *colly.Collector) (*BotService, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации Telegram API: %w", err)
	}
	botAPI.Debug = false // Рекомендуется установить false для продакшена
	log.Printf("Авторизован бот %s", botAPI.Self.UserName)

	return &BotService{
		bot:       botAPI,
		collector: collector,
	}, nil
}

// StartPolling начинает опрос Telegram API на наличие новых обновлений.
func (bs *BotService) StartPolling() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bs.bot.GetUpdatesChan(u)

	// Горутина для проверки пользовательских оповещений
	go bs.checkUserAlerts()

	// Основной цикл обработки обновлений от Telegram API
	for update := range updates {
		if update.Message == nil {
			continue
		}
		bs.handleMessage(update.Message)
	}
}

// handleMessage обрабатывает входящие сообщения.
func (bs *BotService) handleMessage(message *tgbotapi.Message) {
	if message.IsCommand() {
		bs.handleCommand(message)
	} else {
		bs.handleText(message)
	}
}

// handleCommand обрабатывает команды бота.
func (bs *BotService) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"Привет! Введите тикер акции (например, LKOH или AEROFLOT) для запроса цены.\n"+
				"Чтобы установить оповещение, отправьте сообщение в формате: ТИКЕР ЦЕНА\n"+
				"Например: LKOH 7100.0\n"+
				"Чтобы получить список доступных тикеров, нажмите кнопку /list")
		bs.bot.Send(msg)
	case "list":
		var sb strings.Builder
		sb.WriteString("Доступные тикеры:\n")
		for _, info := range stocks.Stocks {
			sb.WriteString(fmt.Sprintf("<b>%s</b> – %s\n", info.Ticker, info.Name))
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
		msg.ParseMode = "HTML"
		bs.bot.Send(msg)
	default:
		bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда."))
	}
}

// handleText обрабатывает текстовые сообщения (запросы цен или установки алертов).
func (bs *BotService) handleText(message *tgbotapi.Message) {
	tokens := strings.Fields(message.Text)

	if len(tokens) == 2 { // Установка оповещения (ТИКЕР ЦЕНА)
		ticker := strings.ToUpper(tokens[0])
		target, err := strconv.ParseFloat(tokens[1], 64)
		if err != nil {
			bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Неверный формат цены. Попробуйте еще раз."))
			return
		}
		info, ok := stocks.Stocks[ticker]
		if !ok {
			bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Тикер %s не найден в базе.", ticker)))
			return
		}
		stock, err := stocks.FetchStockData(info.URL, bs.collector) // Используем collector из BotService
		if err != nil {
			bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Ошибка получения данных для %s: %v", ticker, err)))
			return
		}
		var direction string
		if stock.Price < target {
			direction = "up"
		} else if stock.Price > target {
			direction = "down"
		} else {
			bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("%s уже имеет цену %.2f", stock.Name, stock.Price)))
			return
		}

		// Добавляем алерт в локальный список (для текущей реализации)
		userAlerts = append(userAlerts, Alert{
			Ticker:    ticker,
			Target:    target,
			ChatID:    message.Chat.ID,
			Direction: direction,
		})
		// Если бы алерты были в БД, тут мы бы вызывали repository.SaveAlert
		// err = repository.SaveAlert(repository.Alert{Ticker: ticker, Target: target, ChatID: message.Chat.ID, Direction: direction})
		// if err != nil {
		//    log.Printf("Ошибка сохранения алерта в БД: %v", err)
		//    bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Ошибка при сохранении оповещения."))
		//    return
		// }

		msgText := fmt.Sprintf("Оповещение установлено для %s: когда цена достигнет %.2f, вы получите уведомление.", stock.Name, target)
		bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, msgText))
		return
	}

	if len(tokens) == 1 { // Запрос цены по тикеру
		ticker := strings.ToUpper(strings.TrimSpace(message.Text))
		info, ok := stocks.Stocks[ticker]
		if !ok {
			bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Тикер %s не найден в базе.", ticker)))
			return
		}

		stock, err := stocks.FetchStockData(info.URL, bs.collector) // Используем collector из BotService
		if err != nil {
			bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Ошибка получения данных для %s: %v", ticker, err)))
			return
		}

		response := fmt.Sprintf("Название: %s\nАктуальная цена: %.2f", stock.Name, stock.Price)
		bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, response))
		return
	}

	bs.bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Неизвестный формат сообщения. Попробуйте ввести тикер или 'ТИКЕР ЦЕНА'."))
}

// checkUserAlerts проверяет пользовательские оповещения.
func (bs *BotService) checkUserAlerts() {
	for {
		// Если алерты хранятся в БД, мы бы получили их здесь через repository.GetActiveAlerts()
		// alertsFromDB, err := repository.GetActiveAlerts()
		// if err != nil {
		//    log.Printf("Ошибка получения алертов из БД: %v", err)
		//    time.Sleep(30 * time.Second) // Задержка перед следующей попыткой
		//    continue
		// }
		// for _, alert := range alertsFromDB { ... }

		var remaining []Alert
		for _, alert := range userAlerts { // Пока используем локальный список
			stockInfo, ok := stocks.Stocks[alert.Ticker]
			if !ok {
				continue
			}
			stock, err := stocks.FetchStockData(stockInfo.URL, bs.collector) // Используем collector из BotService
			if err != nil {
				log.Printf("Ошибка проверки пользовательского оповещения для %s: %v", alert.Ticker, err)
				remaining = append(remaining, alert)
				continue
			}
			trigger := false
			if alert.Direction == "up" && stock.Price >= alert.Target {
				trigger = true
			} else if alert.Direction == "down" && stock.Price <= alert.Target {
				trigger = true
			}
			if trigger {
				msgText := fmt.Sprintf("🔔 Оповещение сработало для %s: цена достигла %.2f (текущее значение: %.2f)", stock.Name, alert.Target, stock.Price)
				bs.bot.Send(tgbotapi.NewMessage(alert.ChatID, msgText))
				// Если бы алерты хранились в БД, мы бы удаляли сработавший алерт из БД через repository.DeleteAlert
				// err = repository.DeleteAlert(repository.Alert{ID: alert.ID}) // Предполагая, что Alert имеет ID
				// if err != nil {
				//    log.Printf("Ошибка удаления алерта из БД: %v", err)
				// }
			} else {
				remaining = append(remaining, alert)
			}
		}
		userAlerts = remaining // Обновляем список алертов в памяти
		time.Sleep(30 * time.Second)
	}
}
