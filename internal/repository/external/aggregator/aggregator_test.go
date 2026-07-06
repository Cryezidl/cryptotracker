package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"

	"cryptotracker/internal/model"
	"cryptotracker/internal/repository/external"
	"cryptotracker/pkg/myerrors"
)

type fakeProvider struct {
	name  string
	rate  *model.Rate
	err   error
	delay time.Duration
}

func (f *fakeProvider) GetName() string { return f.name }

func (f *fakeProvider) GetCurrency(ctx context.Context, currency, quote string) (*model.Rate, error) {
	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return f.rate, f.err
}

func TestFailoverProvider_ReturnsFastestSuccess(t *testing.T) {
	slow := &fakeProvider{name: "slow", rate: &model.Rate{Price: 1, Source: "slow"}, delay: 50 * time.Millisecond}
	fast := &fakeProvider{name: "fast", rate: &model.Rate{Price: 2, Source: "fast"}, delay: 5 * time.Millisecond}

	agg := New([]external.Provider{slow, fast})
	rate, err := agg.GetCurrency(context.Background(), "BTC", "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate.Source != "fast" {
		t.Errorf("GetCurrency() source = %q, want %q (fastest provider should win)", rate.Source, "fast")
	}
}

func TestFailoverProvider_FallsBackOnError(t *testing.T) {
	failing := &fakeProvider{name: "failing", err: errors.New("boom")}
	working := &fakeProvider{name: "working", rate: &model.Rate{Price: 3, Source: "working"}, delay: 10 * time.Millisecond}

	agg := New([]external.Provider{failing, working})
	rate, err := agg.GetCurrency(context.Background(), "BTC", "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate.Source != "working" {
		t.Errorf("GetCurrency() source = %q, want %q (should fall back to working provider)", rate.Source, "working")
	}
}

func TestFailoverProvider_AllProvidersFail(t *testing.T) {
	p1 := &fakeProvider{name: "p1", err: errors.New("boom1")}
	p2 := &fakeProvider{name: "p2", err: errors.New("boom2")}

	agg := New([]external.Provider{p1, p2})
	_, err := agg.GetCurrency(context.Background(), "BTC", "USD")
	if !errors.Is(err, myerrors.FetchingCurrencyError) {
		t.Fatalf("GetCurrency() error = %v, want wrapped %v", err, myerrors.FetchingCurrencyError)
	}
}

func TestFailoverProvider_GetName(t *testing.T) {
	agg := New([]external.Provider{&fakeProvider{name: "A"}, &fakeProvider{name: "B"}})
	if got, want := agg.GetName(), "A, B"; got != want {
		t.Errorf("GetName() = %q, want %q", got, want)
	}
}

func TestFailoverProvider_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	slow := &fakeProvider{name: "slow", rate: &model.Rate{Price: 1}, delay: 100 * time.Millisecond}
	agg := New([]external.Provider{slow})

	_, err := agg.GetCurrency(ctx, "BTC", "USD")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetCurrency() error = %v, want %v", err, context.Canceled)
	}
}
