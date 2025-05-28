package stocks

import (
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

var jar *cookiejar.Jar

func InitCollector() *colly.Collector {
	jar, _ = cookiejar.New(nil)
	c := colly.NewCollector(colly.AllowURLRevisit())
	c.WithTransport(&http.Transport{TLSHandshakeTimeout: 10 * time.Second})
	c.SetCookieJar(jar)

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

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", userAgents[time.Now().UnixNano()%int64(len(userAgents))])
		r.Headers.Set("Referer", referers[time.Now().UnixNano()%int64(len(referers))])
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
		r.Headers.Set("Origin", "https://ru.investing.com")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Connection", "keep-alive")
		log.Printf("Visiting %s with User-Agent: %s", r.URL.String(), r.Headers.Get("User-Agent"))
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Ошибка запроса для %s: %v", r.Request.URL, err)
	})

	return c
}

func FetchStockData(url string, baseCollector *colly.Collector) (StockData, error) {
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
