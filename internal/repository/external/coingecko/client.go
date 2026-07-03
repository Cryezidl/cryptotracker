package coingecko

import (
	"context"
	"cryptotracker/internal/model"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type CoinGeckoClient struct {
	url        string
	name       string
	httpClient *http.Client
	limiter    *rate.Limiter
	coinIDs    map[string]string
}

func New(url, name string, rps rate.Limit, burst int) *CoinGeckoClient {
	return &CoinGeckoClient{
		url:  url,
		name: name,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		limiter: rate.NewLimiter(rps, burst),
		coinIDs: map[string]string{
			"BTC":  "bitcoin",
			"ETH":  "ethereum",
			"BNB":  "binancecoin",
			"SOL":  "solana",
			"XRP":  "ripple",
			"TON":  "the-open-network",
			"ADA":  "cardano",
			"DOGE": "dogecoin",
			"USDT": "tether",
		},
	}
}

func (c *CoinGeckoClient) GetName() string {
	return c.name
}

func (c *CoinGeckoClient) GetCurrency(ctx context.Context, currency, quote string) (*model.Rate, error) {

	if !c.limiter.Allow() {
		return nil, fmt.Errorf("%s: rate limit exceeded", c.name)
	}

	coinID, ok := c.coinIDs[strings.ToUpper(currency)]
	if !ok {
		return nil, fmt.Errorf("unsupported currency for coingecko: %s", currency)
	}

	vsCurrency := strings.ToLower(quote)
	if vsCurrency == "usdt" {
		vsCurrency = "usd"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"?ids="+coinID+"&vs_currencies="+vsCurrency, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoinGecko api returned status %d (%s)", resp.StatusCode, resp.Status)
	}

	var obj map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	currencyData, exist := obj[coinID]
	if !exist {
		return nil, fmt.Errorf("no data for coin id: %s", coinID)
	}

	price, exists := currencyData[vsCurrency]
	if !exists {
		return nil, fmt.Errorf("price in %s not found for: %s", quote, coinID)
	}
	return &model.Rate{Price: price, Timestamp: time.Now(), Source: c.name}, nil
}
