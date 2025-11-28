FROM golang:1.25.4 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/worker .

CMD ["./worker"]
