package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"stream-messaging/broker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Get broker URL from environment (defaults to NATS)
	brokerURL := os.Getenv("BROKER_URL")
	if brokerURL == "" {
		brokerURL = "nats://localhost:4222"
	}

	// Create a broker client - implementation is abstracted!
	client, err := broker.NewClient(brokerURL)
	if err != nil {
		logger.Error("Failed to connect to broker", "url", brokerURL, "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// Ensure streams exist (optional - streams can be created by infra team)
	if err := client.EnsureStream("PAYMENT_STREAM", []string{"payments.*"}); err != nil {
		logger.Error("Failed to ensure stream", "stream", "PAYMENT_STREAM", "error", err)
		os.Exit(1)
	}
	if err := client.EnsureStream("ORDER_STREAM", []string{"orders.*"}); err != nil {
		logger.Error("Failed to ensure stream", "stream", "ORDER_STREAM", "error", err)
		os.Exit(1)
	}

	// Create a consumer
	consumer, err := client.NewConsumer("PAYMENT_STREAM", broker.ConsumerConfig{
		Subject:       "payments.received",
		BatchSize:     10,
		AckWait:       30 * time.Second,
		Durable:       "payment-worker",
		MaxDeliver:    5,
		MaxAckPending: 100,
	})
	if err != nil {
		logger.Error("Failed to create consumer", "error", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// Create a publisher
	publisher, err := client.NewPublisher("ORDER_STREAM")
	if err != nil {
		logger.Error("Failed to create publisher", "error", err)
		os.Exit(1)
	}
	defer publisher.Close()

	// Example: Publish a message with new API
	ctx := context.Background()
	err = publisher.Publish(ctx, "orders.created", []byte("order-123"), &broker.PublishOptions{
		MessageID: "msg-001",
		Headers: map[string]string{
			"user-id": "user-456",
		},
	})
	if err != nil {
		logger.Error("Failed to publish message", "error", err)
	} else {
		logger.Info("Published message successfully")
	}

	// Example: Pull batch of messages
	// Note: Timeout is NOT an error - returns empty slice if no messages available
	messages, err := consumer.PullBatch(ctx)
	if err != nil {
		// This is a real error (connection issue, etc.)
		logger.Error("Failed to pull messages", "error", err)
	} else if len(messages) == 0 {
		// No messages available - this is normal, not an error
		logger.Info("No messages available at this time")
	} else {
		logger.Info("Pulled messages", "count", len(messages))
		for _, msg := range messages {
			logger.Info("Processing message",
				"id", msg.ID,
				"subject", msg.Subject,
				"data", string(msg.Data))
			// Process message...
			msg.Ack()
		}
	}

	// Example: Subscribe for continuous consumption (optional)
	// Uncomment to use channel-based consumption
	/*
		msgChan, err := consumer.Subscribe(ctx)
		if err != nil {
			logger.Error("Failed to subscribe", "error", err)
			os.Exit(1)
		}
		for msg := range msgChan {
			logger.Info("Received message",
				"id", msg.ID,
				"subject", msg.Subject,
				"data", string(msg.Data))
			msg.Ack()
		}
	*/

	logger.Info("Example completed successfully")
}
