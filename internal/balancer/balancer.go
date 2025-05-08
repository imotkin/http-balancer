package balancer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/imotkin/http-balancer/internal/client"
	"github.com/imotkin/http-balancer/internal/config"
	"github.com/imotkin/http-balancer/internal/limiter"
	"github.com/imotkin/http-balancer/internal/migrations"
	"github.com/imotkin/http-balancer/internal/server"
)

type Balancer struct {
	// Структура-обёртка для http.Server с добавленным graceful shutdown
	server *server.Server

	// Выбранный алгоритм для работы балансировщика
	strategy Strategy

	// Список серверов, на которые выполняется переадресация исходных запросов
	// endpoints []*Endpoint

	// Атомарное значение для текущего индекса из списка серверов
	// current atomic.Uint64

	// Интервал для проверки (ping) текущего состояния всех серверов балансировщика
	healthInterval time.Duration

	// Структура для работы с ограничением запросов клиентов (Rate Limiting)
	limiter *limiter.Limiter

	// Мютекс для работы с общими для разных горутин данными
	mu sync.Mutex

	// Логгер для событий балансировщика
	logger *slog.Logger

	clients client.DatabaseStorage

	config *config.Config

	ctx    context.Context
	cancel context.CancelFunc
}

// Функция создания нового балансировщика на основе переданной конфигурации
func New(cfg *config.Config) (*Balancer, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	endpoints := make([]*Endpoint, 0, len(cfg.Endpoints))

	for _, u := range cfg.Endpoints {
		endpoint, err := NewEndpoint(u, cfg.HealthInterval.Duration, cfg.LogLevel())
		if err != nil {
			return nil, err
		}

		endpoints = append(endpoints, endpoint)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)

	var output io.Writer

	if cfg.LoggingLevel == "none" {
		output = io.Discard
	} else {
		output = os.Stdout
	}

	logger := slog.New(slog.NewJSONHandler(
		output, &slog.HandlerOptions{
			Level: cfg.LogLevel(),
		}),
	)

	var driver, path string

	if cfg.Mode == "local" {
		driver = "sqlite"
		path = cfg.FilePath
	} else {
		driver = "postgres"
		path = cfg.DatabaseURL()
	}

	storage, err := client.NewStorage(
		driver,
		path,
		cfg.Defaults.Capacity,
		cfg.Defaults.Rate,
	)
	if err != nil {
		return nil, err
	}

	err = migrations.Up(storage.Connection(), driver, cfg.MigrationsPath)
	if err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	limiter := limiter.New(*storage)

	var (
		strategy         Strategy
		trackConnections bool
	)

	switch cfg.Strategy {
	case RoundRobinStrategy:
		strategy = &RoundRobin{endpoints: endpoints}
	case RandomStrategy:
		strategy = &Random{endpoints: endpoints}
	case LeastConnectionsStrategy:
		strategy = &LeastConnections{endpoints: endpoints}
		trackConnections = true
	}

	balancer := &Balancer{
		limiter:  limiter,
		logger:   logger,
		clients:  *storage,
		config:   cfg,
		strategy: strategy,
	}

	r := http.NewServeMux()

	// Обработчики для клиентов
	r.Handle("POST /client", balancer.AddClient())
	r.Handle("GET /client/{key}", balancer.GetClient())
	r.Handle("GET /clients", balancer.GetList())
	r.Handle("DELETE /client/{key}", balancer.DeleteClient())

	// Обработчик для балансировки запросов
	r.Handle("/", balancer.Forward(trackConnections))

	balancer.server = server.New(addr, r)

	return balancer, nil
}

func (b *Balancer) Start(ctx context.Context) {
	// child, cancel := context.WithCancel(ctx)

	// b.ctx = child
	// b.cancel = cancel

	go b.limiter.StartRefill(
		ctx, b.config.RefillInterval.Duration,
	)

	b.server.Listen(ctx)
}
