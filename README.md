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
    BatchSize: 10, // consider 1-5 if polling periodically
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
- `PullBatch` applies a default per-fetch timeout (5s) when the caller does not supply a deadline, to avoid indefinite blocking. If the timeout/deadline is reached and some messages were received, it returns the partial batch without error; if no messages were received it returns an empty slice and no error.
- Configure `ConsumerConfig.BatchTimeout` to override the default; set it just under your polling cadence (e.g., 9s for a 10s poll) to return promptly with available messages.
- Ensure `AckWait` exceeds your processing time, or call `InProgress()` for long-running work; consider raising `MaxAckPending` if you defer acks.
- Logging uses Go `slog`. You can inject your own logger via `WithLogger`.

## Roadmap

- Configurable stream and consumer options (replicas, storage, ack policy, etc.)
- Test suite and CI