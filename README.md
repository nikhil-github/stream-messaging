# stream-messaging

Minimal Go wrapper around NATS JetStream with simple interfaces for `Stream`, `Publisher`, `Consumer`, and `Message`.

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

This will connect to `nats://localhost:4222`, ensure example streams, and construct a consumer and publisher.

## Usage

Create a stream and ensure streams exist:

```go
s, _ := messaging.NewStream("nats://localhost:4222")
defer s.Close()
s.EnsureStream("PAYMENT_STREAM", []string{"payments.*"})
```

Create a consumer:

```go
consumer, _ := s.NewConsumer("PAYMENT_STREAM", messaging.ConsumerConfig{
    Subject:   "payments.received",
    BatchSize: 10,
    Durable:   "payment-worker",
})
msgs, _ := consumer.PullBatch(context.Background())
for _, m := range msgs {
    _ = m.Ack()
}
```

Publish a message:

```go
pub, _ := s.NewPublisher("ORDER_STREAM")
_ = pub.Publish(context.Background(), []byte("hello"), &messaging.PublishOptions{Subject: "orders.created"})
```

## Notes

- `Message` now surfaces headers, ID and timestamp (derived from JetStream metadata when available).
- `Publish` and `Fetch` respect `context.Context` for cancellation.
- Logging uses Go `slog`. You can inject your own logger via `WithLogger`.

## Roadmap

- Configurable stream and consumer options (replicas, storage, ack policy, etc.)
- Test suite and CI