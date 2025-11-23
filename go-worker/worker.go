package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type WeatherMsg struct {
	Timestamp          string                 `json:"timestamp"`
	Temperatura        float64                `json:"temperatura"`
	Umidade            float64                `json:"umidade"`
	Vento              float64                `json:"vento"`
	Ceu                float64                `json:"ceu"`
	ProbabilidadeChuva float64                `json:"probabilidade_chuva"`
	_raw               map[string]interface{} `json:"-"`
}

var (
	rabbitURL     = getenvDefault("RABBITMQ_URL", "")
	rabbitHost    = getenvDefault("RABBITMQ_HOST", "rabbitmq")
	rabbitUser    = getenvDefault("RABBITMQ_DEFAULT_USER", "guest")
	rabbitPass    = getenvDefault("RABBITMQ_DEFAULT_PASS", "guest")
	queueName     = getenvDefault("RABBITMQ_QUEUE", "weather")
	backendURL    = getenvDefault("API_URL", "http://localhost:3000")
	nestEndpoint  = getenvDefault("NESTJS_ENDPOINT", "/api/weather/logs")
	retryAttempts = getenvIntDefault("RETRY_ATTEMPTS", 3)
	retryDelayMs  = getenvIntDefault("RETRY_DELAY_MS", 2000)
	httpTimeoutMs = getenvIntDefault("HTTP_TIMEOUT_MS", 10000)
	prefetchCount = getenvIntDefault("RABBITMQ_PREFETCH", 1)
)

func main() {
	log.Println("Starting Go Weather Worker...")

	if rabbitURL == "" {
		rabbitURL = "amqp://" + rabbitUser + ":" + rabbitPass + "@" + rabbitHost + ":5672/"
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Failed connecting to RabbitMQ: %v", err)
	}
	defer conn.Close()
	log.Println("Connected to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	if err := ch.Qos(prefetchCount, 0, false); err != nil {
		log.Fatalf("Failed to set QoS: %v", err)
	}

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	msgs, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	httpClient := &http.Client{
		Timeout: time.Duration(httpTimeoutMs) * time.Millisecond,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
			MaxIdleConns:        100,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("Received signal %s. Shutting down...", sig)
		cancel()
		time.Sleep(1 * time.Second)
		_ = conn.Close()
		os.Exit(0)
	}()

	log.Printf("Waiting for messages on queue '%s' ...", queueName)
	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				log.Println("Delivery channel closed, exiting")
				return
			}
			go func(delivery amqp.Delivery) {
				processDelivery(ctx, httpClient, &delivery, ch)
			}(d)
		case <-ctx.Done():
			log.Println("Context cancelled, stopping consumer loop")
			return
		}
	}
}

func processDelivery(ctx context.Context, client *http.Client, d *amqp.Delivery, ch *amqp.Channel) {
	start := time.Now()
	log.Printf("Message received: deliveryTag=%d", d.DeliveryTag)

	var raw map[string]interface{}
	if err := json.Unmarshal(d.Body, &raw); err != nil {
		log.Printf("Invalid JSON. Nack (no requeue). Err: %v. Body: %s", err, string(truncate(d.Body, 1000)))
		_ = d.Nack(false, false)
		return
	}

	msg, err := normalizeAndValidate(raw)
	if err != nil {
		log.Printf("Validation failed: %v. Nack (no requeue). Raw: %v", err, raw)
		_ = d.Nack(false, false)
		return
	}

	payloadBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal payload: %v. Nack (no requeue)", err)
		_ = d.Nack(false, false)
		return
	}

	nestURL := strings.TrimSuffix(backendURL, "/") + nestEndpoint
	var lastErr error
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, time.Duration(httpTimeoutMs)*time.Millisecond)
		req, _ := http.NewRequestWithContext(reqCtx, http.MethodPost, nestURL, bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		cancel()
		if err != nil {
			lastErr = err
			log.Printf("Attempt %d/%d: HTTP request error: %v", attempt, retryAttempts, err)
		} else {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				log.Printf("Successfully Posted to NestJS (status=%d) deliveryTag=%d elapsed=%s",
					resp.StatusCode, d.DeliveryTag, time.Since(start))
				if err := d.Ack(false); err != nil {
					log.Printf("Ack failed: %v", err)
				}
				return
			} else {
				lastErr = errors.New("unexpected status " + strconv.Itoa(resp.StatusCode) + " body: " + string(body))
				log.Printf("Attempt %d/%d: NestJS returned %d, body: %s", attempt, retryAttempts, resp.StatusCode, truncate(body, 2000))
			}
		}

		if attempt < retryAttempts {
			backoff := time.Duration(retryDelayMs) * time.Millisecond * time.Duration(attempt)
			log.Printf("Will retry in %s", backoff)
			time.Sleep(backoff)
		}
	}

	log.Printf("All %d attempts failed for deliveryTag=%d. LastErr: %v. Nacking (no requeue).", retryAttempts, d.DeliveryTag, lastErr)
	if err := d.Nack(false, false); err != nil {
		log.Printf("Nack failed: %v", err)
	}
}

func normalizeAndValidate(raw map[string]interface{}) (*WeatherMsg, error) {
	msg := &WeatherMsg{_raw: raw}

	getString := func(k string) (string, bool) {
		v, ok := raw[k]
		if !ok || v == nil {
			return "", false
		}
		switch t := v.(type) {
		case string:
			return t, true
		case float64:
			return strconv.FormatFloat(t, 'f', -1, 64), true
		case json.Number:
			return t.String(), true
		default:
			b, _ := json.Marshal(v)
			return string(b), true
		}
	}

	getFloat := func(k string) (float64, bool) {
		v, ok := raw[k]
		if !ok || v == nil {
			return 0, false
		}
		switch t := v.(type) {
		case float64:
			return t, true
		case string:
			f, err := strconv.ParseFloat(t, 64)
			if err != nil {
				return 0, false
			}
			return f, true
		case json.Number:
			f, err := t.Float64()
			if err != nil {
				return 0, false
			}
			return f, true
		default:
			return 0, false
		}
	}

	if ts, ok := getString("timestamp"); ok {
		msg.Timestamp = ts
	} else {
		return nil, errors.New("missing 'timestamp'")
	}

	if f, ok := getFloat("temperatura"); ok {
		msg.Temperatura = f
	} else {
		return nil, errors.New("missing or invalid 'temperatura'")
	}

	if f, ok := getFloat("umidade"); ok {
		msg.Umidade = f
	} else {
		return nil, errors.New("missing or invalid 'umidade'")
	}

	if f, ok := getFloat("vento"); ok {
		msg.Vento = f
	} else {
		return nil, errors.New("missing or invalid 'vento'")
	}

	if f, ok := getFloat("probabilidade_chuva"); ok {
		msg.ProbabilidadeChuva = f
	} else if f, ok := getFloat("precipitation_probability"); ok {
		msg.ProbabilidadeChuva = f
	} else {
		return nil, errors.New("missing or invalid 'probabilidade_chuva'")
	}

	return msg, nil
}

func getenvDefault(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func getenvIntDefault(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		if ival, err := strconv.Atoi(v); err == nil {
			return ival
		}
	}
	return def
}

func truncate(b []byte, max int) []byte {
	if len(b) <= max {
		return b
	}
	return append(b[:max], []byte("...[truncated]")...)
}
