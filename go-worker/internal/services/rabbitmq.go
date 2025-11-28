package services

import (
	"context"
	"log"

	"github.com/VitorCesarinoMarchese/DesafioGDASH/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQService handles RabbitMQ connection and queue management
type RabbitMQService struct {
	config *config.RabbitMQConfig
	conn   *amqp.Connection
	ch     *amqp.Channel
}

// NewRabbitMQService creates a new RabbitMQ service
func NewRabbitMQService(cfg *config.RabbitMQConfig) (*RabbitMQService, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	if err := ch.Qos(cfg.PrefetchCount, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	_, err = ch.QueueDeclare(
		cfg.Queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitMQService{
		config: cfg,
		conn:   conn,
		ch:     ch,
	}, nil
}

// Consume starts consuming messages from the queue
func (s *RabbitMQService) Consume(ctx context.Context) (<-chan amqp.Delivery, error) {
	msgs, err := s.ch.Consume(
		s.config.Queue,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	log.Printf("Waiting for messages on queue '%s'...", s.config.Queue)
	return msgs, nil
}

// Close closes the RabbitMQ connection
func (s *RabbitMQService) Close() error {
	if s.ch != nil {
		s.ch.Close()
	}
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}