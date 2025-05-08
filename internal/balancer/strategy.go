package balancer

import (
	"math"
	"math/rand/v2"
	"sync/atomic"
)

const (
	LeastConnectionsStrategy = "least-connections"
	RandomStrategy           = "random"
	RoundRobinStrategy       = "round-robin"
)

type Strategy interface {
	Next() *Endpoint
}

type RoundRobin struct {
	endpoints []*Endpoint
	current   atomic.Uint64
}

func (r *RoundRobin) Next() *Endpoint {
	total := len(r.endpoints)

	if total == 0 {
		return nil
	}

	start := int(r.current.Load() % uint64(total))

	for i := range total {
		index := (start + i) % total
		endpoint := r.endpoints[index]

		if endpoint.IsActive() {
			r.current.Store(uint64((index + 1) % total))
			return endpoint
		}
	}

	return nil
}

type Random struct {
	endpoints []*Endpoint
}

func (r *Random) Next() *Endpoint {
	total := len(r.endpoints)

	if total == 0 {
		return nil
	}

	for {
		endpoint := r.endpoints[rand.IntN(total)]

		if endpoint.IsActive() {
			return endpoint
		}
	}
}

type LeastConnections struct {
	endpoints []*Endpoint
}

func (lc *LeastConnections) Next() *Endpoint {
	var endpoint *Endpoint
	minimal := int64(math.MaxInt64)

	for _, e := range lc.endpoints {
		if !e.IsActive() {
			continue
		}

		num := endpoint.Connections()

		if num < minimal {
			minimal = num
			endpoint = e
		}
	}

	return endpoint
}
