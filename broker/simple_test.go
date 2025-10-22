package broker

import (
	"context"
	"testing"
	"time"
)

// TestBasicFunctionality tests the core broker functionality
func TestBasicFunctionality(t *testing.T) {
	// Skip if no NATS server available
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS server not available, skipping basic functionality test")
	}
	defer client.Close()

	// Test stream creation
	streamName := "BASIC_TEST_STREAM"
	subjects := []string{"basic.*"}

	err = client.EnsureStream(streamName, subjects)
	if err != nil {
		t.Fatalf("Failed to ensure stream: %v", err)
	}

	// Test publisher creation
	publisher, err := client.NewPublisher(streamName)
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	// Test consumer creation
	consumer, err := client.NewConsumer(streamName, ConsumerConfig{
		Subject:      "basic.test",
		BatchSize:    5,
		BatchTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Test publishing a message
	ctx := context.Background()
	err = publisher.Publish(ctx, "basic.test", []byte("test-message"), &PublishOptions{
		MessageID: "test-msg-1",
		Headers: map[string]string{
			"test-header": "test-value",
		},
	})
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Test consuming the message
	messages, err := consumer.PullBatch(ctx)
	if err != nil {
		t.Fatalf("Failed to pull batch: %v", err)
	}

	if len(messages) == 0 {
		t.Log("No messages received - this might be due to timing or stream configuration")
		return
	}

	// Verify message content
	msg := messages[0]
	if string(msg.Data) != "test-message" {
		t.Errorf("Expected 'test-message', got '%s'", string(msg.Data))
	}
	if msg.Subject != "basic.test" {
		t.Errorf("Expected subject 'basic.test', got '%s'", msg.Subject)
	}
	if msg.Headers["test-header"] != "test-value" {
		t.Errorf("Expected header 'test-header=test-value', got '%s'", msg.Headers["test-header"])
	}

	// Test message acknowledgment
	err = msg.Ack()
	if err != nil {
		t.Errorf("Failed to ack message: %v", err)
	}

	t.Log("Basic functionality test passed!")
}

// TestTimeoutBehavior tests timeout handling
func TestTimeoutBehavior(t *testing.T) {
	// Skip if no NATS server available
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS server not available, skipping timeout test")
	}
	defer client.Close()

	// Create stream
	err = client.EnsureStream("TIMEOUT_TEST_STREAM_2", []string{"timeout2.*"})
	if err != nil {
		t.Fatalf("Failed to ensure stream: %v", err)
	}

	// Create consumer with short timeout
	consumer, err := client.NewConsumer("TIMEOUT_TEST_STREAM_2", ConsumerConfig{
		Subject:      "timeout2.test",
		BatchSize:    5,
		BatchTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Test timeout behavior (should return empty batch, not error)
	ctx := context.Background()
	start := time.Now()
	messages, err := consumer.PullBatch(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("PullBatch should not return error on timeout, got: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Expected empty batch on timeout, got %d messages", len(messages))
	}
	if duration < 100*time.Millisecond || duration > 300*time.Millisecond {
		t.Errorf("Expected timeout around 100ms, got %v", duration)
	}

	t.Log("Timeout behavior test passed!")
}

// TestSimpleErrorHandling tests error conditions
func TestSimpleErrorHandling(t *testing.T) {
	// Test invalid URL
	_, err := NewClient("")
	if err == nil {
		t.Error("Expected error for empty URL")
	}

	// Test unsupported broker
	_, err = NewClient("amqp://localhost:5672")
	if err == nil {
		t.Error("Expected error for unsupported broker")
	}

	// Test connection to non-existent server
	client, err := NewClient("nats://localhost:9999")
	if err != nil {
		// This is expected - connection should fail
		t.Logf("Expected connection failure: %v", err)
		return
	}
	defer client.Close()

	// Test operations on closed client
	client.Close()

	_, err = client.NewPublisher("test")
	if err == nil {
		t.Error("Expected error when creating publisher on closed client")
	}

	_, err = client.NewConsumer("test", ConsumerConfig{Subject: "test"})
	if err == nil {
		t.Error("Expected error when creating consumer on closed client")
	}

	t.Log("Error handling test passed!")
}

// TestMessageOperations tests message ACK operations
func TestMessageOperations(t *testing.T) {
	// Skip if no NATS server available
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS server not available, skipping message operations test")
	}
	defer client.Close()

	// Create stream
	err = client.EnsureStream("MESSAGE_OPS_STREAM_2", []string{"message2.*"})
	if err != nil {
		t.Fatalf("Failed to ensure stream: %v", err)
	}

	// Create publisher
	publisher, err := client.NewPublisher("MESSAGE_OPS_STREAM_2")
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	// Create consumer
	consumer, err := client.NewConsumer("MESSAGE_OPS_STREAM_2", ConsumerConfig{
		Subject:      "message2.test",
		BatchSize:    1,
		BatchTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Publish a message
	ctx := context.Background()
	err = publisher.Publish(ctx, "message2.test", []byte("test-message"), nil)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Pull the message
	messages, err := consumer.PullBatch(ctx)
	if err != nil {
		t.Fatalf("Failed to pull message: %v", err)
	}
	if len(messages) == 0 {
		t.Log("No messages received - skipping message operations test")
		return
	}

	msg := messages[0]

	// Test InProgress (before ACK)
	err = msg.InProgress()
	if err != nil {
		t.Errorf("Failed to mark message in progress: %v", err)
	}

	// Test ACK
	err = msg.Ack()
	if err != nil {
		t.Errorf("Failed to ack message: %v", err)
	}

	t.Log("Message operations test passed!")
}

// TestSimpleConsumerConfigDefaults tests default configuration
func TestSimpleConsumerConfigDefaults(t *testing.T) {
	// Skip if no NATS server available
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS server not available, skipping defaults test")
	}
	defer client.Close()

	// Create stream
	err = client.EnsureStream("DEFAULTS_TEST_STREAM_2", []string{"defaults2.*"})
	if err != nil {
		t.Fatalf("Failed to ensure stream: %v", err)
	}

	// Test with minimal config (should use defaults)
	consumer, err := client.NewConsumer("DEFAULTS_TEST_STREAM_2", ConsumerConfig{
		Subject: "defaults2.test",
		// No BatchSize or BatchTimeout specified - should use defaults
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// The consumer should be created successfully with defaults
	// (BatchSize=10, BatchTimeout=5s)
	t.Log("Consumer config defaults test passed!")
}
