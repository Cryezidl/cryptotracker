package broadcast

import (
	"cryptotracker/internal/model"
	"sync"
)

type UpdateRate struct {
	Pair          model.PairKey
	OldPrice      float64
	NewPrice      float64
	ChangePercent float64
}

type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[string]chan UpdateRate
}

func New() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[string]chan UpdateRate),
	}
}

func (b *Broadcaster) Subscribe(id string) chan UpdateRate {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan UpdateRate, 10)
	b.subscribers[id] = ch
	return ch
}

func (b *Broadcaster) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch, ok := b.subscribers[id]
	if !ok {
		return
	}
	close(ch)
	delete(b.subscribers, id)
}

func (b *Broadcaster) Publish(update UpdateRate) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- update:
		default:
		}
	}

}
