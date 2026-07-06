package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"cryptotracker/internal/model"
	"cryptotracker/pkg/myerrors"
)

type mockCache struct {
	getFunc func(ctx context.Context, currency, quote string) (*model.Rate, error)
	setFunc func(ctx context.Context, currency, quote string, rate *model.Rate, ttl time.Duration) error
}

func (m *mockCache) GetCrypto(ctx context.Context, currency, quote string) (*model.Rate, error) {
	return m.getFunc(ctx, currency, quote)
}

func (m *mockCache) SetCrypto(ctx context.Context, currency, quote string, rate *model.Rate, ttl time.Duration) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, currency, quote, rate, ttl)
	}
	return nil
}

type mockProvider struct {
	rate *model.Rate
	err  error
}

func (m *mockProvider) GetName() string { return "mock" }

func (m *mockProvider) GetCurrency(ctx context.Context, currency, quote string) (*model.Rate, error) {
	return m.rate, m.err
}

func TestService_GetRate_CacheHit(t *testing.T) {
	cachedRate := &model.Rate{Price: 100, Source: "cache", Timestamp: time.Now()}
	cache := &mockCache{
		getFunc: func(ctx context.Context, currency, quote string) (*model.Rate, error) {
			return cachedRate, nil
		},
	}
	provider := &mockProvider{err: errors.New("provider should not be called on cache hit")}
	svc := New(cache, provider, 30*time.Minute)

	rate, err := svc.GetRate(context.Background(), "btc", "usd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != cachedRate {
		t.Errorf("GetRate() = %v, want cached rate %v", rate, cachedRate)
	}
}

func TestService_GetRate_CacheMiss_ProviderSuccess(t *testing.T) {
	providerRate := &model.Rate{Price: 200, Source: "Binance", Timestamp: time.Now()}
	setCalled := make(chan struct{}, 1)

	cache := &mockCache{
		getFunc: func(ctx context.Context, currency, quote string) (*model.Rate, error) {
			return nil, myerrors.KeyNotFoundInCacheError
		},
		setFunc: func(ctx context.Context, currency, quote string, rate *model.Rate, ttl time.Duration) error {
			if currency != "BTC" || quote != "USD" {
				t.Errorf("SetCrypto key = %s/%s, want BTC/USD", currency, quote)
			}
			if rate != providerRate {
				t.Errorf("SetCrypto rate = %v, want %v", rate, providerRate)
			}
			if ttl != 30*time.Minute {
				t.Errorf("SetCrypto ttl = %v, want %v", ttl, 30*time.Minute)
			}
			setCalled <- struct{}{}
			return nil
		},
	}
	provider := &mockProvider{rate: providerRate}
	svc := New(cache, provider, 30*time.Minute)

	rate, err := svc.GetRate(context.Background(), "btc", "usd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != providerRate {
		t.Errorf("GetRate() = %v, want %v", rate, providerRate)
	}

	select {
	case <-setCalled:
	case <-time.After(time.Second):
		t.Fatal("expected cache.SetCrypto to be called asynchronously")
	}
}

func TestService_GetRate_ProviderError(t *testing.T) {
	cache := &mockCache{
		getFunc: func(ctx context.Context, currency, quote string) (*model.Rate, error) {
			return nil, myerrors.KeyNotFoundInCacheError
		},
	}
	provider := &mockProvider{err: errors.New("boom")}
	svc := New(cache, provider, 30*time.Minute)

	if _, err := svc.GetRate(context.Background(), "btc", "usd"); err == nil {
		t.Fatal("expected error when provider fails, got nil")
	}
}

func TestService_GetRate_UnknownCurrency(t *testing.T) {
	// cache/provider are never touched because normalization fails first.
	svc := New(&mockCache{}, &mockProvider{}, 30*time.Minute)

	if _, err := svc.GetRate(context.Background(), "notacurrency", "usd"); err == nil {
		t.Fatal("expected error for unrecognized currency, got nil")
	}
}

func TestService_GetRate_UnknownQuote(t *testing.T) {
	svc := New(&mockCache{}, &mockProvider{}, 30*time.Minute)

	if _, err := svc.GetRate(context.Background(), "btc", "x"); err == nil {
		t.Fatal("expected error for unrecognized quote currency, got nil")
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOk bool
	}{
		{"lowercase ticker", "btc", "BTC", true},
		{"english name", "bitcoin", "BTC", true},
		{"russian name", "биткоин", "BTC", true},
		{"dollar sign prefix", "$usd", "USD", true},
		{"whitespace padding", "  eth  ", "ETH", true},
		{"unknown but valid length falls back to uppercase", "xyz", "XYZ", true},
		{"too short", "a", "", false},
		{"too long", "abcdefgh", "", false},
		{"empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalize(tt.input)
			if ok != tt.wantOk {
				t.Fatalf("normalize(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
