# stream-messaging

A simple, production-ready message broker abstraction library for Go microservices.

This library provides clean interfaces for `Client`, `Publisher`, `Consumer`, and `Message` that abstract away the underlying broker implementation. Currently supports **NATS JetStream** with plans for RabbitMQ and Kafka.

## Features

- ✅ **Fully abstracted** - Apps don't know which broker they're using
- ✅ **URL-based configuration** - Just pass a broker URL, library handles the rest
- ✅ **Environment-driven** - Perfect for 12-factor apps
- ✅ **Clean, simple API** with proper error types
- ✅ **Synchronous and asynchronous** consumption patterns
- ✅ **Full acknowledgement support** (Ack, Nak, Term, InProgress, AckSync)
- ✅ **Context-aware** operations for timeouts and cancellation
- ✅ **Comprehensive configuration** (MaxDeliver, MaxAckPending, AckWait, etc.)
- ✅ **Message headers and metadata** support

## Quick start

1) Start NATS with JetStream locally (Docker):

```bash
docker run --rm -p 4222:4222 -p 8222:8222 nats:2 -js -sd /data
```

2) Build:

```bash
go build ./...
```

3) Run example `main.go`:

```bash
go run .
```

This will connect to `nats://localhost:4222`, ensure example streams exist, and demonstrate publishing and consuming messages.

## Usage

### Create a broker client (Fully Abstracted!)

The app **doesn't need to know** which broker it's using. Just provide a URL:

```go
import "stream-messaging/broker"

// The broker implementation is determined by the URL scheme
// Currently supports: nats://
// Future: amqp:// (RabbitMQ), kafka:// (Kafka)
client, err := broker.NewClient("nats://localhost:4222")
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

**Better yet - use environment variables:**

```go
// Get broker URL from environment (12-factor app style)
brokerURL := os.Getenv("BROKER_URL")  // Set to "nats://localhost:4222"
client, err := broker.NewClient(brokerURL)
```

### Ensure streams exist (optional)

**Note**: Streams are typically created by your infrastructure team. This method is optional.

```go
err := client.EnsureStream("PAYMENT_STREAM", []string{"payments.*"})
if err != nil {
    log.Fatal(err)
}
```

### Publish messages

```go
publisher, err := client.NewPublisher("ORDER_STREAM")
if err != nil {
    log.Fatal(err)
}
defer publisher.Close()

// Publish with clean API - subject as parameter
err = publisher.Publish(ctx, "orders.created", []byte("order-123"), &broker.PublishOptions{
    MessageID: "msg-001",
    Headers: map[string]string{
        "user-id": "user-456",
    },
})
```

### Consume messages (Pull Pattern)

```go
consumer, err := client.NewConsumer("PAYMENT_STREAM", broker.ConsumerConfig{
    Subject:       "payments.received",
    BatchSize:     10,
    AckWait:       30 * time.Second,
    Durable:       "payment-worker",
    MaxDeliver:    5,
    MaxAckPending: 100,
})
if err != nil {
    log.Fatal(err)
}
defer consumer.Close()

// Pull a batch of messages
// Note: Timeout is NOT an error - returns empty slice if no messages available
messages, err := consumer.PullBatch(ctx)
if err != nil {
    // This is a real error (connection issue, etc.)
    log.Fatal("failed to pull messages:", err)
}

if len(messages) == 0 {
    // No messages available - this is normal, not an error
    log.Println("No messages available, will retry...")
} else {
    for _, msg := range messages {
        // Process message
        fmt.Printf("Received: %s\n", string(msg.Data))
        
        // Acknowledge successful processing
        if err := msg.Ack(); err != nil {
            log.Error("failed to ack", err)
        }
    }
}
```

### Subscribe for continuous consumption (optional)

```go
msgChan, err := consumer.Subscribe(ctx)
if err != nil {
    log.Fatal(err)
}

for msg := range msgChan {
    fmt.Printf("Received: %s\n", string(msg.Data))
    msg.Ack()
}
```

## Timeout Handling

The library handles timeouts gracefully to distinguish between "no messages available" and actual errors:

```go
messages, err := consumer.PullBatch(ctx)
if err != nil {
    // This is a REAL error (connection issue, consumer closed, etc.)
    log.Fatal(err)
}

if len(messages) == 0 {
    // No messages available - this is NORMAL, not an error
    // The consumer will return empty slice on timeout
    log.Println("No messages right now, will retry...")
} else {
    // Process messages
}
```

**Timeout behavior:**
- ✅ **No messages available** → Returns `([], nil)` - empty slice, no error
- ✅ **Context cancelled/expired** → Returns `(nil, context.Err())`
- ✅ **Connection issues** → Returns `(nil, error)`

This makes polling loops cleaner and distinguishes timeouts from real failures.

## Message Acknowledgement Options

The library supports all JetStream acknowledgement patterns:

- `msg.Ack()` - Acknowledge successful processing
- `msg.Nak()` - Negative acknowledge (message will be redelivered)
- `msg.Term()` - Terminate (message will never be redelivered)
- `msg.InProgress()` - Signal processing in progress (resets AckWait timer)
- `msg.AckSync()` - Synchronous acknowledge (waits for server confirmation)

## Error Handling

The library provides typed errors for better error handling:

```go
import "errors"

if err := publisher.Publish(ctx, "", data, nil); err != nil {
    if errors.Is(err, broker.ErrSubjectRequired) {
        // Handle missing subject
    }
    if errors.Is(err, broker.ErrConnectionClosed) {
        // Handle closed connection
    }
}
```

Available error types:
- `ErrStreamNotFound`
- `ErrInvalidConfig`
- `ErrSubjectRequired`
- `ErrNilMessage`
- `ErrConnectionClosed`
- `ErrPublishFailed`
- `ErrConsumerClosed`

## Why "broker" Package?

The package is named `broker` because:
- ✅ **Generic** - works with any message broker (NATS, RabbitMQ, Kafka)
- ✅ **Industry standard** - "message broker" is universally understood
- ✅ **Future-proof** - easy to add new broker implementations
- ✅ **Clear intent** - describes the abstraction, not the implementation

## Multi-Broker Support (Future)

This library uses URL-based auto-detection to support multiple brokers:

```go
// NATS JetStream (current)
client, _ := broker.NewClient("nats://localhost:4222")

// RabbitMQ (future)
client, _ := broker.NewClient("amqp://localhost:5672")

// Kafka (future)
client, _ := broker.NewClient("kafka://localhost:9092")
```

Your microservices don't need code changes - just update the `BROKER_URL` environment variable!

## Configuration-Driven Architecture

Perfect for microservices:

```bash
# Development environment
export BROKER_URL=nats://localhost:4222

# Staging environment
export BROKER_URL=nats://staging-nats:4222

# Production (can switch to RabbitMQ without code changes!)
export BROKER_URL=amqp://prod-rabbitmq:5672
```

Your app code stays the same:
```go
client, _ := broker.NewClient(os.Getenv("BROKER_URL"))
```

## Testing

Run tests:

```bash
go test ./broker/...
```

## Notes

- `Message` includes ID, subject, data, headers, and timestamp (derived from JetStream metadata)
- All operations respect `context.Context` for timeouts and cancellation
- Default batch size is 10 messages if not specified
- Apps are completely abstracted from the broker implementation
- Switch brokers by changing the `BROKER_URL` environment variable