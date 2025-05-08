package balancer

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/imotkin/http-balancer/internal/client"
)

// Основной метод для работы балансировщика. Обработчик получает данные ключа клиента из
// HTTP-заголовка 'X-API-Key' в формате UUID, проверяет наличие свободных запросов для
// данного клиента и при их наличии выполняет переадресацию исходного HTTP-запроса
func (b *Balancer) Forward(trackConnections bool) http.Handler {
	Error := func(w http.ResponseWriter, code int, message string, args ...any) {
		b.logger.Error(message, append(args, "code", code)...)
		ResponseError(w, message, code)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")

		if key == "" {
			Error(w, http.StatusUnauthorized, "client key is not found")
			return
		}

		err := uuid.Validate(key)
		if err != nil {
			Error(w, http.StatusUnauthorized, "invalid client key")
			return
		}

		if !b.limiter.Available(r.Context(), key) {
			Error(w, http.StatusTooManyRequests, "too many requests", "client", key)
			return
		}

		b.mu.Lock()
		endpoint := b.strategy.Next()
		b.mu.Unlock()

		if endpoint == nil {
			Error(w, http.StatusServiceUnavailable, "no available endpoint", "client", key)
			return
		}

		if trackConnections {
			endpoint.NewConnection(r.Context())
		}

		b.logger.Info("Forward request", "client", key, "endpoint", endpoint.id)

		endpoint.proxy.ServeHTTP(w, r)
	})
}

func (b *Balancer) AddClient() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var client client.Client

		err := json.NewDecoder(r.Body).Decode(&client)
		if err != nil {
			ResponseError(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		err = client.Valid()
		if err != nil {
			ResponseError(w, err.Error(), http.StatusBadRequest)
			return
		}

		key, err := b.clients.Add(r.Context(), client)
		if err != nil {
			b.logger.Error("add client", "err", err)
			ResponseError(w, "failed to add a client", http.StatusInternalServerError)
			return
		}

		b.logger.Info(
			"add client",
			"key", key,
			"name", client.Name,
			"capacity", client.Capacity,
			"rate", client.Rate,
		)

		Response(w, ResponseKey{Key: key})
	})
}

func (b *Balancer) GetClient() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		if uuid.Validate(key) != nil {
			ResponseError(w, "invalid client key", http.StatusBadRequest)
			return
		}

		client, err := b.clients.Get(r.Context(), key)
		if err != nil {
			b.logger.Error("get client", "key", key, "err", err)

			if errors.Is(err, sql.ErrNoRows) {
				ResponseError(w, "client is not found", http.StatusNotFound)
				return
			}

			ResponseError(w, "failed to get a client", http.StatusInternalServerError)
			return
		}

		b.logger.Info(
			"get client",
			"key", key,
			"name", client.Name,
			"capacity", client.Capacity,
			"rate", client.Rate,
		)

		Response(w, client)
	})
}

func (b *Balancer) DeleteClient() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		if uuid.Validate(key) != nil {
			ResponseError(w, "invalid client key", http.StatusBadRequest)
			return
		}

		err := b.clients.Delete(r.Context(), key)
		if err != nil {
			b.logger.Error("delete client", "key", key, "err", err)

			if errors.Is(err, sql.ErrNoRows) {
				ResponseError(w, "client is not found", http.StatusNotFound)
				return
			}

			ResponseError(w, "failed to get clients", http.StatusInternalServerError)
			return
		}

		b.logger.Info("delete client", "key", key)

		w.WriteHeader(http.StatusOK)
	})
}

func (b *Balancer) GetList() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clients, err := b.clients.List(r.Context())
		if err != nil {
			b.logger.Error("get clients list", "err", err)
			ResponseError(w, "failed to get clients", http.StatusInternalServerError)
			return
		}

		b.logger.Info("get clients list", "len", len(clients))

		Response(w, clients)
	})
}
