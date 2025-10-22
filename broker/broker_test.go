package broker

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestBrokerIntegration(t *testing.T) {
	// Skip if no NATS server available
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS server not available, skipping integration test")
	}
	defer client.Close()

	// Test stream creation
	streamName := "TEST_STREAM"
	subjects := []string{"test.*"}

	err = client.EnsureStream(streamName, subjects)
	if err != nil {
		t.Fatalf("Failed to ensure stream: %v", err)
	}

	// Test publisher
	publisher, err := client.NewPublisher(streamName)
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	// Test consumer
	consumer, err := client.NewConsumer(streamName, ConsumerConfig{
		Subject:       "test.messages",
		BatchSize:     5,
		BatchTimeout:  2 * time.Second,
		AckWait:       30 * time.Second,
		Durable:       "test-consumer",
		MaxDeliver:    3,
		MaxAckPending: 50,
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Test publishing messages
	ctx := context.Background()
	testMessages := []struct {
		subject string
		data    string
		headers map[string]string
	}{
		{"test.messages", "message-1", map[string]string{"type": "test"}},
		{"test.messages", "message-2", map[string]string{"type": "test"}},
		{"test.messages", "message-3", map[string]string{"type": "test"}},
	}

	for i, msg := range testMessages {
		err = publisher.Publish(ctx, msg.subject, []byte(msg.data), &PublishOptions{
			MessageID: fmt.Sprintf("msg-%d", i+1),
			Headers:   msg.headers,
		})
		if err != nil {
			t.Fatalf("Failed to publish message %d: %v", i+1, err)
		}
	}

	// Test batch consumption
	messages, err := consumer.PullBatch(ctx)
	if err != nil {
		t.Fatalf("Failed to pull batch: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	// Verify message content
	for i, msg := range messages {
		expectedData := fmt.Sprintf("message-%d", i+1)
		if string(msg.Data) != expectedData {
			t.Errorf("Message %d data mismatch: expected %s, got %s", i+1, expectedData, string(msg.Data))
		}
		if msg.Subject != "test.messages" {
			t.Errorf("Message %d subject mismatch: expected test.messages, got %s", i+1, msg.Subject)
		}
		if msg.Headers["type"] != "test" {
			t.Errorf("Message %d header mismatch: expected type=test, got %s", i+1, msg.Headers["type"])
		}
		if msg.ID == "" {
			t.Errorf("Message %d missing ID", i+1)
		}
	}

	// Test message acknowledgment
	for _, msg := range messages {
		err = msg.Ack()
		if err != nil {
			t.Errorf("Failed to ack message: %v", err)
		}
	}

	// Test empty batch (no messages available)
	messages, err = consumer.PullBatch(ctx)
	if err != nil {
		t.Fatalf("Failed to pull empty batch: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Expected empty batch, got %d messages", len(messages))
	}
}

func TestConsumerTimeout(t *testing.T) {
	// Skip if no NATS server available
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS server not available, skipping timeout test")
	}
	defer client.Close()

	// Create stream
	err = client.EnsureStream("TIMEOUT_STREAM", []string{"timeout.*"})
	if err != nil {
		t.Fatalf("Failed to ensure stream: %v", err)
	}

	// Create consumer with short timeout
	consumer, err := client.NewConsumer("TIMEOUT_STREAM", ConsumerConfig{
		Subject:      "timeout.test",
		BatchSize:    5,
		BatchTimeout: 100 * time.Millisecond, // Very short timeout
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
	if duration < 100*time.Millisecond || duration > 200*time.Millisecond {
		t.Errorf("Expected timeout around 100ms, got %v", duration)
	}
}

func TestConsumerContextCancellation(t *testing.T) {
	// Skip if no NATS server available
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS server not available, skipping cancellation test")
	}
	defer client.Close()

	// Create stream
	err = client.EnsureStream("CANCEL_STREAM", []string{"cancel.*"})
	if err != nil {
		t.Fatalf("Failed to ensure stream: %v", err)
	}

	// Create consumer
	consumer, err := client.NewConsumer("CANCEL_STREAM", ConsumerConfig{
		Subject:      "cancel.test",
		BatchSize:    5,
		BatchTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Test context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	messages, err := consumer.PullBatch(ctx)
	duration := time.Since(start)

	// The consumer should respect the context timeout
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded or nil, got: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Expected empty batch on cancellation, got %d messages", len(messages))
	}
	if duration < 50*time.Millisecond || duration > 150*time.Millisecond {
		t.Errorf("Expected cancellation around 50ms, got %v", duration)
	}
}

func TestMessageAckOperations(t *testing.T) {
	// Skip if no NATS server available
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS server not available, skipping ack test")
	}
	defer client.Close()

	// Create stream
	err = client.EnsureStream("ACK_STREAM", []string{"ack.*"})
	if err != nil {
		t.Fatalf("Failed to ensure stream: %v", err)
	}

	// Create publisher
	publisher, err := client.NewPublisher("ACK_STREAM")
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	// Create consumer
	consumer, err := client.NewConsumer("ACK_STREAM", ConsumerConfig{
		Subject:      "ack.test",
		BatchSize:    1,
		BatchTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Publish a message
	ctx := context.Background()
	err = publisher.Publish(ctx, "ack.test", []byte("test-message"), nil)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Pull the message
	messages, err := consumer.PullBatch(ctx)
	if err != nil {
		t.Fatalf("Failed to pull message: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0]

	// Test Ack
	err = msg.Ack()
	if err != nil {
		t.Errorf("Failed to ack message: %v", err)
	}

	// Test AckSync
	err = publisher.Publish(ctx, "ack.test", []byte("test-message-2"), nil)
	if err != nil {
		t.Fatalf("Failed to publish second message: %v", err)
	}

	messages, err = consumer.PullBatch(ctx)
	if err != nil {
		t.Fatalf("Failed to pull second message: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg = messages[0]
	err = msg.AckSync()
	if err != nil {
		t.Errorf("Failed to ack sync message: %v", err)
	}

	// Test InProgress
	err = publisher.Publish(ctx, "ack.test", []byte("test-message-3"), nil)
	if err != nil {
		t.Fatalf("Failed to publish third message: %v", err)
	}

	messages, err = consumer.PullBatch(ctx)
	if err != nil {
		t.Fatalf("Failed to pull third message: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg = messages[0]
	err = msg.InProgress()
	if err != nil {
		t.Errorf("Failed to mark message in progress: %v", err)
	}

	// Test Nak
	err = msg.Nak()
	if err != nil {
		t.Errorf("Failed to nak message: %v", err)
	}

	// Test Term
	err = publisher.Publish(ctx, "ack.test", []byte("test-message-4"), nil)
	if err != nil {
		t.Fatalf("Failed to publish fourth message: %v", err)
	}

	messages, err = consumer.PullBatch(ctx)
	if err != nil {
		t.Fatalf("Failed to pull fourth message: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg = messages[0]
	err = msg.Term()
	if err != nil {
		t.Errorf("Failed to term message: %v", err)
	}
}

func TestConsumerSubscribe(t *testing.T) {
	// Skip if no NATS server available
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS server not available, skipping subscribe test")
	}
	defer client.Close()

	// Create stream
	err = client.EnsureStream("SUBSCRIBE_STREAM", []string{"subscribe.*"})
	if err != nil {
		t.Fatalf("Failed to ensure stream: %v", err)
	}

	// Create publisher
	publisher, err := client.NewPublisher("SUBSCRIBE_STREAM")
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	// Create consumer with unique durable name to avoid message duplication
	consumer, err := client.NewConsumer("SUBSCRIBE_STREAM", ConsumerConfig{
		Subject:      "subscribe.test",
		BatchSize:    2,
		BatchTimeout: 1 * time.Second,
		Durable:      "subscribe-test-consumer",
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Start subscription
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	msgChan, err := consumer.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Publish messages
	go func() {
		time.Sleep(100 * time.Millisecond) // Let subscription start
		for i := 0; i < 3; i++ {
			err := publisher.Publish(ctx, "subscribe.test", []byte(fmt.Sprintf("subscribe-message-%d", i+1)), nil)
			if err != nil {
				t.Errorf("Failed to publish message %d: %v", i+1, err)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Collect messages
	var receivedMessages []*Message
	timeout := time.After(2 * time.Second)

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				// Channel closed
				goto done
			}
			receivedMessages = append(receivedMessages, msg)
			msg.Ack() // Acknowledge message
		case <-timeout:
			goto done
		}
	}
done:

	if len(receivedMessages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(receivedMessages))
	}

	// Verify message content
	for i, msg := range receivedMessages {
		expectedData := fmt.Sprintf("subscribe-message-%d", i+1)
		if string(msg.Data) != expectedData {
			t.Errorf("Message %d data mismatch: expected %s, got %s", i+1, expectedData, string(msg.Data))
		}
	}
}

func TestErrorHandling(t *testing.T) {
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
}

func TestConsumerConfigDefaults(t *testing.T) {
	// Skip if no NATS server available
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS server not available, skipping defaults test")
	}
	defer client.Close()

	// Create stream
	err = client.EnsureStream("DEFAULTS_STREAM", []string{"defaults.*"})
	if err != nil {
		t.Fatalf("Failed to ensure stream: %v", err)
	}

	// Test with minimal config (should use defaults)
	consumer, err := client.NewConsumer("DEFAULTS_STREAM", ConsumerConfig{
		Subject: "defaults.test",
		// No BatchSize or BatchTimeout specified - should use defaults
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// The consumer should be created successfully with defaults
	// (BatchSize=10, BatchTimeout=5s)
}
