package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cryptotracker/internal/model"
	"cryptotracker/internal/repository/cache"
	"cryptotracker/internal/repository/external"
)

type Service struct {
	cache    cache.CacheRepository
	provider external.Provider
	cacheTTL time.Duration
}

func New(cache cache.CacheRepository, provider external.Provider, cacheTTL time.Duration) *Service {
	return &Service{cache: cache, provider: provider, cacheTTL: cacheTTL}
}

func (s *Service) GetRate(ctx context.Context, currency, quote string) (*model.Rate, error) {
	//normalize input
	clean, found := normalize(currency)
	if !found {
		return nil, fmt.Errorf("unknown currency")
	}

	quoteClean, found := normalize(quote)
	if !found {
		return nil, fmt.Errorf("unknown quote currency")
	}

	//check cache first
	rate, err := s.cache.GetCrypto(ctx, clean, quoteClean)
	if err == nil {
		return rate, nil
	}

	//if not found in cache, fetch from provider
	rate, err = s.provider.GetCurrency(ctx, clean, quoteClean)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rate for %s: %w", clean, err)
	}

	//store in cache asynchronously
	go func() {
		s.cache.SetCrypto(context.Background(), clean, quoteClean, rate, s.cacheTTL)
	}()

	return rate, nil
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
