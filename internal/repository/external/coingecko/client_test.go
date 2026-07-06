package coingecko

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestCoinGeckoClient_GetCurrency_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("ids"); got != "bitcoin" {
			t.Errorf("request ids = %q, want %q", got, "bitcoin")
		}
		if got := r.URL.Query().Get("vs_currencies"); got != "usd" {
			t.Errorf("request vs_currencies = %q, want %q (USDT should map to usd)", got, "usd")
		}
		_ = json.NewEncoder(w).Encode(map[string]map[string]float64{"bitcoin": {"usd": 65000.5}})
	}))
	defer srv.Close()

	client := New(srv.URL, "CoinGecko", rate.Inf, 1)
	got, err := client.GetCurrency(context.Background(), "BTC", "USDT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Price != 65000.5 {
		t.Errorf("Price = %v, want %v", got.Price, 65000.5)
	}
	if got.Source != "CoinGecko" {
		t.Errorf("Source = %q, want %q", got.Source, "CoinGecko")
	}
}

func TestCoinGeckoClient_GetCurrency_UnsupportedCurrency(t *testing.T) {
	// Unknown symbols are rejected before any HTTP call is made.
	client := New("http://unused.invalid", "CoinGecko", rate.Inf, 1)
	if _, err := client.GetCurrency(context.Background(), "NOTREAL", "USD"); err == nil {
		t.Fatal("expected error for unsupported currency, got nil")
	}
}

func TestCoinGeckoClient_GetCurrency_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := New(srv.URL, "CoinGecko", rate.Inf, 1)
	if _, err := client.GetCurrency(context.Background(), "BTC", "USD"); err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}

func TestCoinGeckoClient_GetCurrency_PriceMissingForVsCurrency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]map[string]float64{"bitcoin": {"eur": 1}})
	}))
	defer srv.Close()

	client := New(srv.URL, "CoinGecko", rate.Inf, 1)
	if _, err := client.GetCurrency(context.Background(), "BTC", "USD"); err == nil {
		t.Fatal("expected error when response has no price for requested vs_currency, got nil")
	}
}

func TestCoinGeckoClient_GetCurrency_RateLimited(t *testing.T) {
	client := New("http://unused.invalid", "CoinGecko", rate.Limit(1), 0)
	if _, err := client.GetCurrency(context.Background(), "BTC", "USD"); err == nil {
		t.Fatal("expected rate limit error, got nil")
	}
}
