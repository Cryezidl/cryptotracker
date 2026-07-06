package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestBinanceClient_GetCurrency_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("symbol"); got != "BTCUSDT" {
			t.Errorf("request symbol = %q, want %q (USD should map to USDT)", got, "BTCUSDT")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"symbol": "BTCUSDT", "price": "65000.5"})
	}))
	defer srv.Close()

	client := New(srv.URL, "Binance", rate.Inf, 1)
	got, err := client.GetCurrency(context.Background(), "BTC", "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Price != 65000.5 {
		t.Errorf("Price = %v, want %v", got.Price, 65000.5)
	}
	if got.Source != "Binance" {
		t.Errorf("Source = %q, want %q", got.Source, "Binance")
	}
}

func TestBinanceClient_GetCurrency_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := New(srv.URL, "Binance", rate.Inf, 1)
	if _, err := client.GetCurrency(context.Background(), "BTC", "USD"); err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}

func TestBinanceClient_GetCurrency_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := New(srv.URL, "Binance", rate.Inf, 1)
	if _, err := client.GetCurrency(context.Background(), "BTC", "USD"); err == nil {
		t.Fatal("expected error for malformed json body, got nil")
	}
}

func TestBinanceClient_GetCurrency_UnparsablePrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"symbol": "BTCUSDT", "price": "not-a-number"})
	}))
	defer srv.Close()

	client := New(srv.URL, "Binance", rate.Inf, 1)
	if _, err := client.GetCurrency(context.Background(), "BTC", "USD"); err == nil {
		t.Fatal("expected error for unparsable price, got nil")
	}
}

func TestBinanceClient_GetCurrency_RateLimited(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_ = json.NewEncoder(w).Encode(map[string]string{"symbol": "BTCUSDT", "price": "1"})
	}))
	defer srv.Close()

	// burst 0 means Allow() never has a token to give out.
	client := New(srv.URL, "Binance", rate.Limit(1), 0)
	if _, err := client.GetCurrency(context.Background(), "BTC", "USD"); err == nil {
		t.Fatal("expected rate limit error, got nil")
	}
	if called {
		t.Error("server should not have been called when rate limit is exhausted")
	}
}
