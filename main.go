package main

import (
    "log/slog"
    "os"
    "time"

    "stream-messaging/messaging"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    natsURL := os.Getenv("NATS_URL")
    if natsURL == "" {
        natsURL = "nats://localhost:4222"
    }

    stream, err := messaging.NewStream(natsURL)
    if err != nil {
        logger.Error("Failed to connect to NATS", "url", natsURL, "error", err)
        os.Exit(1)
    }
    defer stream.Close()
    stream = stream.WithLogger(logger)

    // Ensure streams
    if err := stream.EnsureStream("PAYMENT_STREAM", []string{"payments.*"}); err != nil {
        logger.Error("Failed to ensure stream", "stream", "PAYMENT_STREAM", "error", err)
        os.Exit(1)
    }
    if err := stream.EnsureStream("ORDER_STREAM", []string{"orders.*"}); err != nil {
        logger.Error("Failed to ensure stream", "stream", "ORDER_STREAM", "error", err)
        os.Exit(1)
    }

    // Consumer
    consumer, err := stream.NewConsumer("PAYMENT_STREAM", messaging.ConsumerConfig{
        Subject:   "payments.received",
        BatchSize: 10,
        AckWait:   30 * time.Second,
        Durable:   "payment-worker",
    })
    if err != nil {
        logger.Error("Failed to create consumer", "error", err)
        os.Exit(1)
    }
    _ = consumer

    // Publisher
    publisher, err := stream.NewPublisher("ORDER_STREAM")
    if err != nil {
        logger.Error("Failed to create publisher", "error", err)
        os.Exit(1)
    }
    _ = publisher

    // Your PaymentConsumer / OrderPublisher service logic here
}
