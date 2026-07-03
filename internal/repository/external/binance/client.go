package binance

import (
	"context"
	"cryptotracker/internal/model"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

type BinanceClient struct {
	url        string
	name       string
	httpClient *http.Client
	limiter    *rate.Limiter
}

func New(url, name string, rps rate.Limit, burst int) *BinanceClient {
	return &BinanceClient{
		url:  url,
		name: name,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		limiter: rate.NewLimiter(rps, burst),
	}
}

func (c *BinanceClient) GetName() string {
	return c.name
}

func (c *BinanceClient) GetCurrency(ctx context.Context, currency, quote string) (*model.Rate, error) {

	if !c.limiter.Allow() {
		return nil, fmt.Errorf("%s: rate limit exceeded", c.name)
	}

	if quote == "USD" {
		quote = "USDT"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"?symbol="+currency+quote, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance api returned status %d (%s)", resp.StatusCode, resp.Status)
	}

	type jsonObj struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}

	var obj jsonObj
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	curPrice, err := strconv.ParseFloat(obj.Price, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price '%s': %w", obj.Price, err)
	}
	return &model.Rate{Price: curPrice, Timestamp: time.Now(), Source: c.name}, nil
}
