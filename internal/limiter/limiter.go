package limiter

import (
	"context"
	"sync"
	"time"

	"github.com/imotkin/http-balancer/internal/client"
)

type TokenBucket struct {
	capacity   uint
	tokens     uint
	rate       uint
	lastRefill time.Time
	mu         sync.Mutex
}

func (b *TokenBucket) Available() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill(time.Now())

	if b.tokens != 0 {
		b.tokens--
		return true
	} else {
		return false
	}
}

func (b *TokenBucket) refill(now time.Time) {
	elapsed := now.Sub(b.lastRefill)
	amount := uint(float64(elapsed) / float64(time.Second) * float64(b.rate))
	if amount > 0 {
		b.tokens = min(b.capacity, (b.tokens + amount))
		b.lastRefill = now
	}
}

func NewBucket(capacity, rate uint) *TokenBucket {
	bucket := &TokenBucket{
		capacity: capacity,
		tokens:   capacity,
		rate:     rate,
	}

	return bucket
}

type Limiter struct {
	buckets map[string]*TokenBucket
	// clients Storage
	clients client.DatabaseStorage
	mu      sync.RWMutex
}

func New(storage client.DatabaseStorage) *Limiter {
	return &Limiter{
		buckets: make(map[string]*TokenBucket),
		clients: storage,
	}
}

func (l *Limiter) Available(ctx context.Context, key string) bool {
	l.mu.RLock()
	bucket, found := l.buckets[key]
	l.mu.RUnlock()

	if !found {
		l.mu.Lock()
		defer l.mu.Unlock()

		if bucket, found = l.buckets[key]; !found {
			client, err := l.clients.Has(ctx, key)
			if err != nil {
				return false
			}

			created := NewBucket(client.Capacity, client.Rate)
			l.buckets[key] = created
			return created.Available()
		} else {
			return bucket.Available()
		}
	}

	return bucket.Available()
}

func (l *Limiter) StartRefill(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.mu.RLock()

			for _, bucket := range l.buckets {
				bucket.mu.Lock()
				bucket.refill(time.Now())
				bucket.mu.Unlock()
			}

			l.mu.RUnlock()
		case <-ctx.Done():
			return
		}
	}
}
