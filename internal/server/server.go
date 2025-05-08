package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	*http.Server
}

func New(addr string, handler http.Handler) *Server {
	return &Server{
		Server: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  time.Minute,
		},
	}
}

// Метод для запуска прослушиваний HTTP-запросов сервером, который
// блокируется и ожидает завершения работы с помощью сигналов SIGINT, SIGTERM
func (s *Server) Listen(ctx context.Context) {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	wait := make(chan struct{})

	go func() {
		log.Println("Got an interrupt signal:", <-exit)

		err := s.Shutdown(ctx)
		if err != nil {
			log.Println("Failed to shutdown HTTP server:", err)
		}

		log.Println("Server HTTP was closed")

		close(wait)
	}()

	go func() {
		log.Printf("Started HTTP server at http://localhost%s\n", s.Addr)

		err := s.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start HTTP server: %v\n", err)
		}
	}()

	<-wait
}
