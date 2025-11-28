package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/VitorCesarinoMarchese/DesafioGDASH/internal/models"
	"github.com/VitorCesarinoMarchese/DesafioGDASH/internal/services"
	"github.com/VitorCesarinoMarchese/DesafioGDASH/pkg/utils"
	amqp "github.com/rabbitmq/amqp091-go"
)

// MessageHandler processes incoming messages
type MessageHandler struct {
	apiClient *services.APIClient
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(apiClient *services.APIClient) *MessageHandler {
	return &MessageHandler{
		apiClient: apiClient,
	}
}

// ProcessDelivery processes a single message delivery
func (h *MessageHandler) ProcessDelivery(ctx context.Context, d *amqp.Delivery) {
	start := time.Now()
	log.Printf("Message received: deliveryTag=%d", d.DeliveryTag)

	var raw map[string]interface{}
	if err := json.Unmarshal(d.Body, &raw); err != nil {
		log.Printf("Invalid JSON. Nack (no requeue). Err: %v. Body: %s", err, string(utils.Truncate(d.Body, 1000)))
		_ = d.Nack(false, false)
		return
	}

	msg, err := models.NormalizeAndValidate(raw)
	if err != nil {
		log.Printf("Validation failed: %v. Nack (no requeue). Raw: %v", err, raw)
		_ = d.Nack(false, false)
		return
	}

	if err := h.apiClient.SendWeatherData(ctx, msg); err != nil {
		log.Printf("Failed to send weather data: %v. Nack (no requeue)", err)
		if err := d.Nack(false, false); err != nil {
			log.Printf("Nack failed: %v", err)
		}
		return
	}

	log.Printf("Successfully processed message deliveryTag=%d elapsed=%s",
		d.DeliveryTag, time.Since(start))
	if err := d.Ack(false); err != nil {
		log.Printf("Ack failed: %v", err)
	}
}