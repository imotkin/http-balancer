package balancer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/imotkin/http-balancer/internal/config"
)

func BenchmarkBalancerTestSingle(b *testing.B) {
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	}))
	defer endpoint.Close()

	cfg := config.Default()

	cfg.LoggingLevel = "none"
	cfg.Endpoints = []string{endpoint.URL}
	cfg.MigrationsPath = "./../../migrations"
	cfg.FilePath = "clients.sqlite"

	balancer, err := New(cfg)
	if err != nil {
		b.Fatalf("Failed to create a balancer: %v", err)
	}

	// Добавляем клиента с ёмкостью 10000 и скоростью добавления 100 токенов в секунду
	body := `{"name":"test-client","capacity":10000000,"rate":100}`
	req := httptest.NewRequest("POST", "/clients", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	balancer.AddClient().ServeHTTP(rr, req)

	defer func() {
		os.Remove(cfg.FilePath)
	}()

	if rr.Code != http.StatusOK {
		b.Fatalf("failed to add client, got status code: %d", rr.Code)
	}

	// Получаем API-ключ для созданного клиента
	var key ResponseKey

	if err := json.NewDecoder(rr.Body).Decode(&key); err != nil {
		b.Fatalf("failed to parse response: %v", err)
	}

	// Добавляем в запросы заголовок X-API-Key
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", key.Key)

	b.ResetTimer()

	for b.Loop() {
		resp := httptest.NewRecorder()
		balancer.Forward(false).ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			b.Fatalf("unexpected code: %d", resp.Code)
		}
	}
}

func BenchmarkBalancerSingle(b *testing.B) {
	// Чтобы отследить порядок запросов в логах серверов
	// возможно передать 'os.Stdout' вместо 'io.Discard'

	NewListener(8001, io.Discard)

	cfg := config.Default()

	cfg.LoggingLevel = "none"
	cfg.Endpoints = []string{
		"http://localhost:8001",
	}
	cfg.MigrationsPath = "./../../migrations"
	cfg.FilePath = "clients.sqlite"

	balancer, err := New(cfg)
	if err != nil {
		b.Fatalf("Failed to create a balancer: %v", err)
	}

	// Добавляем клиента с ёмкостью 10000 и скоростью добавления 100 токенов в секунду
	body := `{"name":"test-client","capacity":10000000,"rate":100}`
	req := httptest.NewRequest("POST", "/clients", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	balancer.AddClient().ServeHTTP(rr, req)

	defer func() {
		os.Remove(cfg.FilePath)
	}()

	if rr.Code != http.StatusOK {
		b.Fatalf("failed to add client, got status code: %d", rr.Code)
	}

	// Получаем API-ключ для созданного клиента
	var key ResponseKey

	if err := json.NewDecoder(rr.Body).Decode(&key); err != nil {
		b.Fatalf("failed to parse response: %v", err)
	}

	// Добавляем в запросы заголовок X-API-Key
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", key.Key)

	b.ResetTimer()

	for b.Loop() {
		resp := httptest.NewRecorder()
		balancer.Forward(false).ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			b.Fatalf("unexpected code: %d", resp.Code)
		}
	}
}

func BenchmarkBalancerMultiple(b *testing.B) {
	// Чтобы отследить порядок запросов в логах серверов
	// возможно передать 'os.Stdout' вместо 'io.Discard'
	for _, port := range []int{8001, 8002, 8003} {
		NewListener(port, io.Discard)
	}

	cfg := config.Default()

	cfg.LoggingLevel = "none"
	cfg.Endpoints = []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}
	cfg.MigrationsPath = "./../../migrations"
	cfg.FilePath = "clients.sqlite"

	balancer, err := New(cfg)
	if err != nil {
		b.Fatalf("Failed to create a balancer: %v", err)
	}

	// Добавляем клиента с ёмкостью 10000 и скоростью добавления 100 токенов в секунду
	body := `{"name":"test-client","capacity":10000000,"rate":100}`
	req := httptest.NewRequest("POST", "/clients", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	balancer.AddClient().ServeHTTP(rr, req)

	defer func() {
		os.Remove(cfg.FilePath)
	}()

	if rr.Code != http.StatusOK {
		b.Fatalf("failed to add client, got status code: %d", rr.Code)
	}

	// Получаем API-ключ для созданного клиента
	var key ResponseKey

	if err := json.NewDecoder(rr.Body).Decode(&key); err != nil {
		b.Fatalf("failed to parse response: %v", err)
	}

	// Добавляем в запросы заголовок X-API-Key
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", key.Key)

	b.ResetTimer()

	for b.Loop() {
		resp := httptest.NewRecorder()
		balancer.Forward(false).ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			b.Fatalf("unexpected code: %d", resp.Code)
		}
	}
}
