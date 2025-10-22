package broker

import (
	"fmt"
	"strings"
)

// NewClient creates a new broker client based on the provided URL.
// The URL scheme determines which broker implementation to use:
//   - nats:// - NATS JetStream
//   - amqp:// - RabbitMQ (future)
//   - kafka:// - Kafka (future)
//
// Example:
//
//	client, err := broker.NewClient("nats://localhost:4222")
func NewClient(url string) (Client, error) {
	if url == "" {
		return nil, fmt.Errorf("broker URL is required")
	}

	// Determine broker type from URL scheme
	scheme := parseScheme(url)

	switch scheme {
	case "nats":
		return newNATSClient(url)
	case "amqp", "amqps":
		return nil, fmt.Errorf("RabbitMQ support not yet implemented")
	case "kafka":
		return nil, fmt.Errorf("Kafka support not yet implemented")
	default:
		// Default to NATS if no scheme or unknown scheme
		return newNATSClient(url)
	}
}

// parseScheme extracts the scheme from a URL
// Examples: "nats://localhost:4222" -> "nats"
func parseScheme(url string) string {
	if idx := strings.Index(url, "://"); idx != -1 {
		return strings.ToLower(url[:idx])
	}
	return ""
}

// newNATSClient creates a NATS JetStream client
func newNATSClient(url string) (Client, error) {
	return newJetStreamClient(url)
}
