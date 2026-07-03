package aggregator

import (
	"context"
	"cryptotracker/internal/model"
	"cryptotracker/internal/repository/external"
	"cryptotracker/pkg/myerrors"
	"fmt"
	"strings"
)

type FailoverProvider struct {
	providers []external.Provider
}

func New(providers []external.Provider) *FailoverProvider {
	return &FailoverProvider{providers: providers}
}

func (p *FailoverProvider) GetName() string {
	names := make([]string, len(p.providers))
	for i, provider := range p.providers {
		names[i] = provider.GetName()
	}
	return strings.Join(names, ", ")
}

func (p *FailoverProvider) GetCurrency(ctx context.Context, currency, quote string) (*model.Rate, error) {
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	type res struct {
		val *model.Rate
		err error
	}
	results := make(chan res, len(p.providers))

	for _, client := range p.providers {
		go func(c external.Provider) {
			val, err := c.GetCurrency(ctxCancel, currency, quote)
			results <- res{val: val, err: err}
		}(client)
	}
	var lastErr error
	for i := 0; i < len(p.providers); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r := <-results:
			if r.err == nil {
				cancel()
				return r.val, nil
			}

			lastErr = r.err
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("%w: last error from client: %v", myerrors.FetchingCurrencyError, lastErr)
	}
	return nil, myerrors.FetchingCurrencyError
}
