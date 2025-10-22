package main

import (
	"log/slog"
	"os"
	"time"

	"yourapp/internal/messaging"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout))
	stream, _ := messaging.NewStream("nats://localhost:4222")
	stream = stream.WithLogger(logger)

	// Ensure streams
	stream.EnsureStream("PAYMENT_STREAM", []string{"payments.*"})
	stream.EnsureStream("ORDER_STREAM", []string{"orders.*"})

	// Consumer
	consumer, _ := stream.NewConsumer("PAYMENT_STREAM", messaging.ConsumerConfig{
		Subject:   "payments.received",
		BatchSize: 10,
		AckWait:   30 * time.Second,
		Durable:   "payment-worker",
	})

	// Publisher
	publisher, _ := stream.NewPublisher("ORDER_STREAM")

	// Inject into services
	_ = consumer
	_ = publisher
	// Your PaymentConsumer / OrderPublisher service logic here
}
