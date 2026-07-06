package worker

import (
	"context"
	"log"
	"time"

	"cryptotracker/internal/broadcast"
	"cryptotracker/internal/model"
	"cryptotracker/internal/service"
)

type Worker struct {
	rateService  *service.Service
	broadcaster  *broadcast.Broadcaster
	interval     time.Duration
	trackedPairs []model.PairKey
	oldRates     map[model.PairKey]model.Rate
}

func New(rateService *service.Service, broadcaster *broadcast.Broadcaster, interval time.Duration, pairs []model.PairKey) *Worker {
	return &Worker{
		rateService:  rateService,
		broadcaster:  broadcaster,
		interval:     interval,
		trackedPairs: pairs,
		oldRates:     make(map[model.PairKey]model.Rate),
	}
}

func (w *Worker) Start(ctx context.Context) error {
	t := time.NewTicker(w.interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-t.C:
			{
				for _, pair := range w.trackedPairs {
					rate, err := w.rateService.GetRate(ctx, pair.Base, pair.Quote)
					if err != nil {
						log.Printf("worker faced error while getting rate from the service: %v\n", err)
						continue
					}

					oldRate, seen := w.oldRates[pair]
					if !seen {
						w.oldRates[pair] = *rate
						continue
					}
					if oldRate.Price != rate.Price {
						w.broadcaster.Publish(broadcast.UpdateRate{
							Pair:          pair,
							OldPrice:      oldRate.Price,
							NewPrice:      rate.Price,
							ChangePercent: (rate.Price - oldRate.Price) / oldRate.Price * 100,
						})
						w.oldRates[pair] = *rate
					}
				}
			}
		}
	}

}
