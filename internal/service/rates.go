package service

import (
	"context"
	"cryptotracker/internal/repository/cache"
	"cryptotracker/internal/repository/external"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	cache    cache.CacheRepository
	provider external.Provider
}

func New(cache cache.CacheRepository, provider external.Provider) *Service {
	return &Service{cache: cache, provider: provider}
}

func (s *Service) GetRate(ctx context.Context, currency, quote string) (float64, error) {
	//normalize input
	clean, found := normalize(currency)
	if !found {
		return -1, fmt.Errorf("unknown currency")
	}

	quoteClean, found := normalize(quote)
	if !found {
		return -1, fmt.Errorf("unknown quote currency")
	}

	//check cache first
	price, err := s.cache.GetCrypto(ctx, clean, quoteClean)
	if err == nil {
		return price, nil
	}

	//if not found in cache, fetch from provider
	price, err = s.provider.GetCurrency(ctx, clean, quoteClean)
	if err != nil {
		return -1, fmt.Errorf("failed to fetch rate for %s: %w", clean, err)
	}

	//store in cache asynchronously
	go func() {
		s.cache.SetCrypto(context.Background(), clean, quoteClean, price, 30*time.Minute)
	}()

	return price, nil
}

func normalize(input string) (string, bool) {
	names := map[string]string{
		//crypto currencies
		"bitcoin": "BTC",
		"биткоин": "BTC",
		"btc":     "BTC",

		"eth":      "ETH",
		"ethereum": "ETH",
		"эфириум":  "ETH",

		"bnb":         "BNB",
		"binancecoin": "BNB",

		"sol":    "SOL",
		"solana": "SOL",
		"солана": "SOL",

		"xrp":    "XRP",
		"ripple": "XRP",
		"рипл":   "XRP",

		"ton":     "TON",
		"toncoin": "TON",
		"тонкоин": "TON",

		"ada":     "ADA",
		"cardano": "ADA",
		"кардано": "ADA",

		"doge":     "DOGE",
		"dogecoin": "DOGE",
		"догикоин": "DOGE",

		//quote currencies
		"usdt":   "USDT",
		"tether": "USDT",
		"тезер":  "USDT",
		"usdc":   "USDC",

		"usd":    "USD",
		"dollar": "USD",
		"доллар": "USD",

		"eur":  "EUR",
		"euro": "EUR",
		"евро": "EUR",

		"rub":   "RUB",
		"rur":   "RUB",
		"рубль": "RUB",
		"рубли": "RUB",

		"uah":    "UAH",
		"гривна": "UAH",
		"гривны": "UAH",

		"kzt":   "KZT",
		"тенге": "KZT",
	}

	clean := strings.ToLower(input)
	clean = strings.TrimSpace(clean)
	clean = strings.TrimPrefix(clean, "$")

	val, ok := names[clean]
	if ok {
		return val, true
	}

	if len(clean) >= 3 && len(clean) <= 5 {
		return strings.ToUpper(clean), true
	}

	return "", false
}
