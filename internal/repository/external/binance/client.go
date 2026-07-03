package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type BinanceClient struct {
	url        string
	name       string
	httpClient *http.Client
}

func New(url, name string) *BinanceClient {
	return &BinanceClient{
		url:  url,
		name: name,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *BinanceClient) GetName() string {
	return c.name
}

func (c *BinanceClient) GetCurrency(ctx context.Context, currency, quote string) (float64, error) {
	if quote == "USD" {
		quote = "USDT"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"?symbol="+currency+quote, nil)
	if err != nil {
		return -1, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("binance api returned status %d (%s)", resp.StatusCode, resp.Status)
	}

	type jsonObj struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}

	var obj jsonObj
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return -1, fmt.Errorf("failed to decode json: %w", err)
	}

	curPrice, err := strconv.ParseFloat(obj.Price, 64)
	if err != nil {
		return -1, fmt.Errorf("failed to parse price '%s': %w", obj.Price, err)
	}
	return curPrice, nil
}
