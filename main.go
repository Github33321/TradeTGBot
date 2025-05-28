package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
)

var (
	jar *cookiejar.Jar

	alerts      []Alert
	alertsMutex sync.Mutex
)

type StockInfo struct {
	Ticker string
	URL    string
	Name   string
}

type StockData struct {
	Name  string
	Price float64
}

type Alert struct {
	Ticker    string
	Target    float64
	ChatID    int64
	Direction string
}

var stocks = map[string]StockInfo{
	"LKOH":     {"LKOH", "https://ru.investing.com/equities/lukoil_rts", "Лукойл"},
	"AEROFLOT": {"AEROFLOT", "https://ru.investing.com/equities/aeroflot", "Аэрофлот"},
	"AFKS":     {"AFKS", "https://ru.investing.com/equities/afk-sistema_rts", "АФК Система"},
	"T":        {"T", "https://ru.investing.com/equities/tcs-group-holding-plc", "TCS Group Holding Plc"},
	"MAGN":     {"MAGN", "https://ru.investing.com/equities/mmk_rts", "ММК"},
	"SBER":     {"SBER", "https://ru.investing.com/equities/sberbank_rts", "Сбербанк"},
	"YDEX":     {"YDEX", "https://ru.investing.com/equities/yandex", "Яндекс"},
	"MSTT":     {"MSTT", "https://ru.investing.com/equities/mostotrest_rts", "Мостотрест"},
	"APTK":     {"APTK", "https://ru.investing.com/equities/apteka-36-6_rts", "Аптека-36.6"},
	"WUSH":     {"WUSH", "https://ru.investing.com/equities/whoosh-holding-pao", "Whoosh Holding"},
	"HEAD":     {"HEAD", "https://ru.investing.com/equities/headhunter-ipjsc", "Хэдхантер"},
	"FLOT":     {"FLOT", "https://ru.investing.com/equities/sovcomflot-pao", "Совкомфлот"},
	"CHMF":     {"CHMF", "https://ru.investing.com/equities/severstal_rts", "Северсталь"},
	"GAZP":     {"GAZP", "https://ru.investing.com/equities/gazprom_rts", "Газпром"},
	"SIBN":     {"SIBN", "https://ru.investing.com/equities/gazprom-neft_rts", "Газпром нефть"},
	"BLNG":     {"BLNG", "https://ru.investing.com/equities/belon_rts", "Белон"},
}

func fetchStockData(url string, baseCollector *colly.Collector) (StockData, error) {
	var data StockData
	collector := baseCollector.Clone()
	collector.SetCookieJar(jar)
	done := make(chan struct{})

	collector.OnHTML("h1", func(e *colly.HTMLElement) {
		if data.Name == "" {
			data.Name = strings.TrimSpace(e.Text)
		}
	})
	collector.OnHTML(`div[data-test="instrument-price-last"]`, func(e *colly.HTMLElement) {
		priceStr := strings.TrimSpace(e.Text)
		if priceStr != "" {
			priceStr = strings.ReplaceAll(priceStr, " ", "")
			priceStr = strings.ReplaceAll(priceStr, ".", "")
			priceStr = strings.ReplaceAll(priceStr, ",", ".")
			priceStr = strings.TrimSpace(priceStr)
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				log.Printf("Ошибка преобразования цены (%s): %v", priceStr, err)
			} else {
				data.Price = price
			}
		}
		close(done)
	})

	err := collector.Visit(url)
	if err != nil {
		return data, err
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		return data, fmt.Errorf("таймаут ожидания данных")
	}
	return data, nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	_ = godotenv.Load()
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN is not set in .env file")
	}

	var err error
	jar, err = cookiejar.New(nil)
	if err != nil {
		log.Fatal("Ошибка создания cookie jar:", err)
	}

	baseCollector := colly.NewCollector(
		colly.AllowURLRevisit(),
	)
	baseCollector.WithTransport(&http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
	})

	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:94.0) Gecko/20100101 Firefox/94.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Safari/605.1.15",
	}
	referers := []string{
		"https://ru.investing.com/",
		"https://www.google.com/",
		"https://yandex.ru/",
	}

	baseCollector.SetCookieJar(jar)
	baseCollector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
		r.Headers.Set("Referer", referers[rand.Intn(len(referers))])
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
		r.Headers.Set("Origin", "https://ru.investing.com")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Connection", "keep-alive")
		log.Printf("Visiting %s with User-Agent: %s", r.URL.String(), r.Headers.Get("User-Agent"))
	})
	baseCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("Ошибка запроса для %s: %v", r.Request.URL, err)
	})

	err = baseCollector.Visit("https://ru.investing.com")
	if err != nil {
		log.Printf("Предварительный запрос не удался: %v", err)
	}
	time.Sleep(2 * time.Second)

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Авторизован бот %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	go func() {
		for {
			alertsMutex.Lock()
			var remaining []Alert
			for _, alert := range alerts {
				stockInfo, ok := stocks[alert.Ticker]
				if !ok {
					continue
				}
				stock, err := fetchStockData(stockInfo.URL, baseCollector)
				if err != nil {
					log.Printf("Ошибка проверки оповещения для %s: %v", alert.Ticker, err)
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
					msgText := fmt.Sprintf("Оповещение! %s достигла цены %.2f (текущее значение: %.2f)", stock.Name, alert.Target, stock.Price)
					msg := tgbotapi.NewMessage(alert.ChatID, msgText)
					bot.Send(msg)
				} else {
					remaining = append(remaining, alert)
				}
			}
			alerts = remaining
			alertsMutex.Unlock()
			time.Sleep(30 * time.Second)
		}
	}()

	for update := range updates {
		if update.Message == nil {
			continue
		}
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID,
					"Привет! Введите тикер акции (например, LKOH или AEROFLOT) для запроса цены.\n"+
						"Чтобы установить оповещение, отправьте сообщение в формате: ТИКЕР ЦЕНА\n"+
						"Например: LKOH 7100.0\n"+
						"Чтобы получить список доступных тикеров, нажмите кнопку /list")
				bot.Send(msg)
			case "list":
				var sb strings.Builder
				sb.WriteString("Доступные тикеры:\n")
				for _, info := range stocks {
					sb.WriteString(fmt.Sprintf("<b>%s</b> – %s\n", info.Ticker, info.Name))
				}
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, sb.String())
				msg.ParseMode = "HTML"
				bot.Send(msg)
			}
			continue
		}

		tokens := strings.Fields(update.Message.Text)
		if len(tokens) == 2 {
			ticker := strings.ToUpper(tokens[0])
			target, err := strconv.ParseFloat(tokens[1], 64)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат цены. Попробуйте еще раз.")
				bot.Send(msg)
				continue
			}
			info, ok := stocks[ticker]
			if !ok {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Тикер %s не найден в базе.", ticker))
				bot.Send(msg)
				continue
			}
			stock, err := fetchStockData(info.URL, baseCollector)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка получения данных для %s: %v", ticker, err))
				bot.Send(msg)
				continue
			}
			var direction string
			if stock.Price < target {
				direction = "up"
			} else if stock.Price > target {
				direction = "down"
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("%s уже имеет цену %.2f", stock.Name, stock.Price))
				bot.Send(msg)
				continue
			}

			newAlert := Alert{
				Ticker:    ticker,
				Target:    target,
				ChatID:    update.Message.Chat.ID,
				Direction: direction,
			}
			alertsMutex.Lock()
			alerts = append(alerts, newAlert)
			alertsMutex.Unlock()

			msgText := fmt.Sprintf("Оповещение установлено для %s: когда цена достигнет %.2f, вы получите уведомление.", stock.Name, target)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
			bot.Send(msg)
			continue
		}

		if len(tokens) == 1 {
			ticker := strings.ToUpper(strings.TrimSpace(update.Message.Text))
			info, ok := stocks[ticker]
			if !ok {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Тикер %s не найден в базе.", ticker))
				bot.Send(msg)
				continue
			}

			stock, err := fetchStockData(info.URL, baseCollector)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка получения данных для %s: %v", ticker, err))
				bot.Send(msg)
				continue
			}

			response := fmt.Sprintf("Название: %s\nАктуальная цена: %.2f", stock.Name, stock.Price)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
			bot.Send(msg)
		}
	}
}
