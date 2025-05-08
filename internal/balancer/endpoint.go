package balancer

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Endpoint struct {
	id          uuid.UUID
	proxy       *httputil.ReverseProxy
	active      atomic.Bool
	url         *url.URL
	tick        <-chan time.Time
	mu          sync.RWMutex
	cancel      chan struct{}
	client      *http.Client
	connections atomic.Int64
	logger      *slog.Logger
}

func NewEndpoint(URL string, healthInterval time.Duration, logLevel slog.Level) (*Endpoint, error) {
	url, err := url.Parse(URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	proxy := httputil.NewSingleHostReverseProxy(url)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusServiceUnavailable)
		logger.Error("proxy error", "err", err)
	}

	endpoint := &Endpoint{
		id:     uuid.New(),
		proxy:  proxy,
		url:    url,
		tick:   time.Tick(healthInterval),
		cancel: make(chan struct{}),
		client: &http.Client{
			Timeout: healthInterval,
		},
		logger: logger,
	}

	endpoint.Enable()

	endpoint.SetHealthCheck(healthInterval)

	return endpoint, nil
}

func (e *Endpoint) pingEndpoint(cancel <-chan struct{}, attempts int, timeout time.Duration) bool {
	var current int

	for {
		select {
		case <-time.Tick(timeout):
			resp, err := e.client.Get(e.url.String())
			if err != nil {
				if current < attempts {
					current++
					continue
				}

				e.logger.Info("ping failed", "id", e.id, "current", current+1, "attempts", attempts)

				return false
			}

			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				e.logger.Info("ping succeeded", "id", e.id, "current", current+1, "attempts", attempts)
				return true
			}
		case <-cancel:
			e.logger.Info("stop ping process", "id", e.id)
		}
	}
}

func (e *Endpoint) SetHealthCheck(interval time.Duration) {
	attempts := 5
	timeout := interval / time.Duration(attempts)

	go func() {
		for {
			select {
			case <-e.tick:
				healthy := e.pingEndpoint(e.cancel, attempts, timeout)
				active := e.IsActive()

				if healthy && !active {
					e.logger.Info("endpoint is now active", "id", e.id)
					e.Enable()
				} else if !healthy && active {
					e.logger.Info("endpoint is not active now", "id", e.id)
					e.Disable()
				}
			case <-e.cancel:
				e.logger.Info("stop endpoint health check", "id", e.id)
				return
			}
		}
	}()
}

func (e *Endpoint) IsActive() bool {
	return e.active.Load()
}

func (e *Endpoint) Enable() {
	e.active.Store(true)
}

func (e *Endpoint) Disable() {
	e.active.Store(false)
}

func (e *Endpoint) Connections() int64 {
	return e.connections.Load()
}

func (e *Endpoint) NewConnection(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	e.connections.Add(1)

	go func() {
		<-ctx.Done()
		e.connections.Add(-1)
	}()
}
