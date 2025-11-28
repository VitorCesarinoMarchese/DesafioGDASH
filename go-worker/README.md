# Weather Worker

A Go application that consumes weather data from RabbitMQ and forwards it to a NestJS API.

## Project Structure

This project follows the recommended Go project layout:

```
cmd/worker/          # Main application entry point
internal/            # Private application code
├── config/         # Configuration management
├── handlers/       # Message handlers
├── models/         # Data models
└── services/       # Business logic services
pkg/utils/          # Public utility functions
```

## Key Components

- **cmd/worker/main.go**: Application entry point with service initialization and message processing loop
- **internal/config**: Environment-based configuration loading
- **internal/models**: Weather message data structure and validation
- **internal/services**: RabbitMQ and API client services
- **internal/handlers**: Message processing logic
- **pkg/utils**: Reusable utility functions

## Configuration

The application is configured via environment variables:

### RabbitMQ Settings
- `RABBITMQ_URL`: Full RabbitMQ connection URL (overrides other RabbitMQ settings)
- `RABBITMQ_HOST`: RabbitMQ host (default: "rabbitmq")
- `RABBITMQ_DEFAULT_USER`: Username (default: "guest")
- `RABBITMQ_DEFAULT_PASS`: Password (default: "guest")
- `RABBITMQ_QUEUE`: Queue name (default: "weather")
- `RABBITMQ_PREFETCH`: Prefetch count (default: 1)

### API Settings
- `API_URL`: Base URL for the NestJS API (default: "http://localhost:3000")
- `NESTJS_ENDPOINT`: API endpoint path (default: "/api/weather/logs")
- `HTTP_TIMEOUT_MS`: HTTP request timeout (default: 10000)
- `RETRY_ATTEMPTS`: Number of retry attempts (default: 3)
- `RETRY_DELAY_MS`: Base retry delay (default: 2000)

## Building

```bash
go build ./cmd/worker
```

## Docker

```bash
docker build -t weather-worker .
```

## Architecture Benefits

The restructured code provides:

1. **Separation of Concerns**: Each package has a single responsibility
2. **Testability**: Services and handlers can be easily unit tested
3. **Maintainability**: Clear module boundaries make code easier to understand
4. **Extensibility**: New features can be added without modifying existing code
5. **Standard Layout**: Follows Go community conventions