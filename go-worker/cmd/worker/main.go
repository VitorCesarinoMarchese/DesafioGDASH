package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VitorCesarinoMarchese/DesafioGDASH/internal/config"
	"github.com/VitorCesarinoMarchese/DesafioGDASH/internal/handlers"
	"github.com/VitorCesarinoMarchese/DesafioGDASH/internal/services"
)

func main() {
	log.Println("Starting Go Weather Worker...")

	// Load configuration
	cfg := config.Load()

	// Initialize services
	rabbitmqSvc, err := services.NewRabbitMQService(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ service: %v", err)
	}
	defer rabbitmqSvc.Close()
	log.Println("Connected to RabbitMQ")

	apiClient := services.NewAPIClient(&cfg.API)
	messageHandler := handlers.NewMessageHandler(apiClient)

	// Setup context and signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("Received signal %s. Shutting down...", sig)
		cancel()
		time.Sleep(1 * time.Second)
		_ = rabbitmqSvc.Close()
		os.Exit(0)
	}()

	// Start consuming messages
	msgs, err := rabbitmqSvc.Consume(ctx)
	if err != nil {
		log.Fatalf("Failed to start consuming messages: %v", err)
	}

	// Message processing loop
	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				log.Println("Delivery channel closed, exiting")
				return
			}
			go messageHandler.ProcessDelivery(ctx, &d)
		case <-ctx.Done():
			log.Println("Context cancelled, stopping consumer loop")
			return
		}
	}
}