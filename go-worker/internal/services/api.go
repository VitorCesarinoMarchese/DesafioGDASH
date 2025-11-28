package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/VitorCesarinoMarchese/DesafioGDASH/internal/config"
	"github.com/VitorCesarinoMarchese/DesafioGDASH/internal/models"
)

// APIClient handles communication with the NestJS API
type APIClient struct {
	client *http.Client
	config *config.APIConfig
}

// NewAPIClient creates a new API client
func NewAPIClient(cfg *config.APIConfig) *APIClient {
	client := &http.Client{
		Timeout: time.Duration(cfg.TimeoutMs) * time.Millisecond,
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

	return &APIClient{
		client: client,
		config: cfg,
	}
}

// SendWeatherData sends weather data to the NestJS API with retry logic
func (c *APIClient) SendWeatherData(ctx context.Context, msg *models.WeatherMsg) error {
	payloadBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	url := strings.TrimSuffix(c.config.BaseURL, "/") + c.config.Endpoint
	var lastErr error

	for attempt := 1; attempt <= c.config.RetryAttempts; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, time.Duration(c.config.TimeoutMs)*time.Millisecond)
		req, _ := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		cancel()

		if err != nil {
			lastErr = err
			log.Printf("Attempt %d/%d: HTTP request error: %v", attempt, c.config.RetryAttempts, err)
		} else {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				log.Printf("Successfully posted to API (status=%d)", resp.StatusCode)
				return nil
			}

			lastErr = errors.New("unexpected status " + strconv.Itoa(resp.StatusCode) + " body: " + string(body))
			log.Printf("Attempt %d/%d: API returned %d, body: %s", attempt, c.config.RetryAttempts, resp.StatusCode, truncate(body, 2000))
		}

		if attempt < c.config.RetryAttempts {
			backoff := time.Duration(c.config.RetryDelayMs) * time.Millisecond * time.Duration(attempt)
			log.Printf("Will retry in %s", backoff)
			time.Sleep(backoff)
		}
	}

	return lastErr
}

func truncate(b []byte, max int) []byte {
	if len(b) <= max {
		return b
	}
	return append(b[:max], []byte("...[truncated]")...)
}