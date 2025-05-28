package stocks

type StockInfo struct {
	Ticker string
	URL    string
	Name   string
}

type StockData struct {
	Name  string
	Price float64
}

var Stocks = map[string]StockInfo{
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
