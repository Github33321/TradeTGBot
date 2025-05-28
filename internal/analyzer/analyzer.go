// TradeTGBot/internal/analyzer/analyzer.go
package analyzer

import (
	"TradeTGBot/internal/repository"
	"TradeTGBot/pkg/stocks"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gocolly/colly" // <-- ДОБАВИТЬ ЭТОТ ИМПОРТ!
)

// PriceAnalyzer отвечает за анализ цен и отправку уведомлений о резких изменениях.
type PriceAnalyzer struct {
	Bot            *tgbotapi.BotAPI
	Collector      *colly.Collector // <-- ИЗМЕНЕНО НА *colly.Collector
	TargetChatID   int64
	Interval       time.Duration
	Threshold      float64 // Процент изменения для уведомления
	AveragePeriod  time.Duration
	lastAlertPrice float64 // Для предотвращения спама уведомлениями
	alertThreshold float64 // Процент для отправки нового уведомления после предыдущего
}

// NewPriceAnalyzer создает новый экземпляр PriceAnalyzer.
func NewPriceAnalyzer(bot *tgbotapi.BotAPI, collector *colly.Collector, targetChatID int64, interval, averagePeriod time.Duration, threshold float64) *PriceAnalyzer { // <-- ИЗМЕНЕНО НА *colly.Collector
	return &PriceAnalyzer{
		Bot:            bot,
		Collector:      collector,
		TargetChatID:   targetChatID,
		Interval:       interval,
		Threshold:      threshold,
		AveragePeriod:  averagePeriod,
		lastAlertPrice: 0.0,
		alertThreshold: 0.1, // 0.1% для нового уведомления
	}
}

// StartAnalysis запускает горутину для периодического анализа цен.
func (pa *PriceAnalyzer) StartAnalysis() {
	go pa.analyzeLoop()
}

func (pa *PriceAnalyzer) analyzeLoop() {
	ticker := "LKOH" // Отслеживаем только LKOH
	info, ok := stocks.Stocks[ticker]
	if !ok {
		log.Printf("Ошибка: Тикер %s не найден в списке отслеживаемых акций. Анализ не будет выполнен.", ticker)
		return
	}

	for {
		// 1. Получаем текущую цену LKOH
		stock, err := stocks.FetchStockData(info.URL, pa.Collector)
		if err != nil {
			log.Printf("Ошибка при получении данных для LKOH: %v", err)
			time.Sleep(pa.Interval)
			continue
		}
		currentPrice := stock.Price
		log.Printf("LKOH: Текущая цена: %.2f", currentPrice)

		// 2. Сохраняем текущую цену в БД через репозиторий
		err = repository.SaveStockPrice(ticker, currentPrice)
		if err != nil {
			log.Printf("Ошибка при сохранении цены LKOH в БД: %v", err)
		}

		// 3. Получаем среднюю цену за указанный период через репозиторий
		avgPrice, err := repository.GetAveragePrice(ticker, pa.AveragePeriod)
		if err != nil {
			log.Printf("Ошибка при получении средней цены LKOH за %s: %v", pa.AveragePeriod, err)
			time.Sleep(pa.Interval)
			continue
		}
		log.Printf("LKOH: Средняя цена за %s: %.2f", pa.AveragePeriod, avgPrice)

		// 4. Вычисляем процентное изменение
		if avgPrice > 0 { // Избегаем деления на ноль
			percentageChange := ((currentPrice - avgPrice) / avgPrice) * 100
			log.Printf("LKOH: Отклонение от средней за %s: %.2f%%", pa.AveragePeriod, percentageChange)

			// 5. Проверяем, достигнут ли порог для уведомления
			if percentageChange >= pa.Threshold || percentageChange <= -pa.Threshold {
				// Проверяем, чтобы не спамить уведомлениями
				if pa.lastAlertPrice == 0.0 || (currentPrice/pa.lastAlertPrice < (1 - pa.alertThreshold)) || (currentPrice/pa.lastAlertPrice > (1 + pa.alertThreshold)) {
					msgText := fmt.Sprintf("🚨 **Резкое изменение цены LKOH!**\nТекущая цена: %.2f\nСредняя цена за %s: %.2f\nОтклонение: %.2f%%",
						currentPrice, pa.AveragePeriod, avgPrice, percentageChange)

					if pa.TargetChatID != 0 {
						_, err := pa.Bot.Send(tgbotapi.NewMessage(pa.TargetChatID, msgText))
						if err != nil {
							log.Printf("Ошибка при отправке уведомления о резком изменении LKOH: %v", err)
						} else {
							log.Println("Уведомление о резком изменении LKOH отправлено.")
							pa.lastAlertPrice = currentPrice // Обновляем цену последнего алерта
						}
					}
				}
			} else {
				// Сбрасываем lastAlertPrice, если цена вернулась в норму
				pa.lastAlertPrice = 0.0
			}
		}

		time.Sleep(pa.Interval) // Ждем до следующей проверки
	}
}
