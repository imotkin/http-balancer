package main

import (
	"context"
	"log"

	"github.com/imotkin/http-balancer/internal/balancer"
	"github.com/imotkin/http-balancer/internal/config"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalln("Failed to load a configuration:", err)
	}

	log.Printf("Configuration: %#v\n", cfg)

	balancer, err := balancer.New(cfg)
	if err != nil {
		log.Fatalln("Failed to create a balancer:", err)
	}

	balancer.Start(ctx)
}
