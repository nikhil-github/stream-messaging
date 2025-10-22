package messaging

import (
    "context"
    "errors"
    "log/slog"
    "time"

    "github.com/nats-io/nats.go"
)

type PublishOptions struct {
	MessageID string
	Headers   map[string]string
	Subject   string // required
}

type Publisher interface {
	Publish(ctx context.Context, data []byte, opts *PublishOptions) error
	Close() error
}

type jsPublisher struct {
	js     nats.JetStreamContext
	stream string
	logger *slog.Logger
}

// RunConsumer is a helper that loops fetching batches and invoking a handler.
// It handles basic backoff on fetch errors and acknowledges messages based on handler result.
type HandlerFunc func(context.Context, *Message) error

type RunConsumerOptions struct {
    // Deprecated: the batch size is specified when creating the Consumer; this is kept for API symmetry.
    FetchBatchSize int
    // Sleep on fetch error before retrying
    FetchErrorBackoff time.Duration
}

func RunConsumer(ctx context.Context, c Consumer, handler HandlerFunc, opts RunConsumerOptions) error {
    batchSize := opts.FetchBatchSize
    if batchSize <= 0 {
        batchSize = 32
    }
    backoff := opts.FetchErrorBackoff
    if backoff <= 0 {
        backoff = time.Second
    }

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        msgs, err := c.PullBatch(ctx)
        if err != nil {
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
            }
            time.Sleep(backoff)
            continue
        }
        for _, m := range msgs {
            if err := handler(ctx, m); err != nil {
                _ = m.Nak()
            } else {
                _ = m.Ack()
            }
        }
    }
}

func (p *jsPublisher) Publish(ctx context.Context, data []byte, opts *PublishOptions) error {
	if opts == nil || opts.Subject == "" {
		return errors.New("subject required")
	}

	pubOpts := []nats.PubOpt{}
	if opts.MessageID != "" {
		pubOpts = append(pubOpts, nats.MsgId(opts.MessageID))
	}
    // Ensure publish respects the provided context for timeout/cancel
    if ctx != nil {
        pubOpts = append(pubOpts, nats.Context(ctx))
    }

	msg := &nats.Msg{
		Subject: opts.Subject,
		Data:    data,
	}
	if opts.Headers != nil {
		msg.Header = nats.Header{}
		for k, v := range opts.Headers {
			msg.Header.Set(k, v)
		}
	}

    pa, err := p.js.PublishMsg(msg, pubOpts...)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("Failed to publish", "subject", opts.Subject, "error", err)
		}
		return err
	}

	if p.logger != nil {
		p.logger.Info("Published message", "subject", opts.Subject, "seq", pa.Sequence)
	}
	return nil
}

func (p *jsPublisher) Close() error {
	return nil
}
