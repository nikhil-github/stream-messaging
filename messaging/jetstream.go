package messaging

import (
    "errors"
    "fmt"

    "github.com/nats-io/nats.go"
)

type jetStream struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
}

func NewStream(url string) (Stream, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}

    return &jetStream{nc: nc, js: js}, nil
}

func (s *jetStream) EnsureStream(streamName string, subjects []string) error {
	_, err := s.js.StreamInfo(streamName)
	if err == nil {
		return nil
	}
	if err == nats.ErrStreamNotFound {
		_, err = s.js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: subjects,
			Storage:  nats.FileStorage,
			Replicas: 1,
		})
	}
	return err
}

func (s *jetStream) NewPublisher(streamName string) (Publisher, error) {
    return &jsPublisher{js: s.js, stream: streamName}, nil
}

func (s *jetStream) NewConsumer(streamName string, cfg ConsumerConfig) (Consumer, error) {
    // Choose filter subject
    subject := cfg.FilterSubject
    if subject == "" {
        subject = cfg.Subject
    }
    if subject == "" {
        return nil, errors.New("subject or filter subject required")
    }

    durable := cfg.Durable
    if durable == "" && cfg.ConsumerGroup != "" {
        durable = cfg.ConsumerGroup
    }

    // If durable specified and advanced options provided, proactively create/update consumer
    if durable != "" && (cfg.MaxDeliver > 0 || cfg.MaxAckPending > 0 || cfg.AckWait > 0 || subject != cfg.Subject) {
        // Explicit-ack policy for pull consumers
        cconf := &nats.ConsumerConfig{
            Durable:        durable,
            AckPolicy:      nats.AckExplicitPolicy,
            AckWait:        cfg.AckWait,
            MaxDeliver:     cfg.MaxDeliver,
            MaxAckPending:  cfg.MaxAckPending,
            FilterSubject:  subject,
            DeliverPolicy:  nats.DeliverAllPolicy,
            ReplayPolicy:   nats.ReplayInstantPolicy,
        }
        if _, err := s.js.AddConsumer(streamName, cconf); err != nil {
            // If it already exists, attempt to update
            if !errors.Is(err, nats.ErrConsumerNameAlreadyInUse) {
                // Fallback attempt to update consumer; ignore returned info
                if _, uerr := s.js.UpdateConsumer(streamName, cconf); uerr != nil {
                    return nil, fmt.Errorf("ensure consumer: %w", err)
                }
            }
        }
    } else if durable == "" && (cfg.MaxDeliver > 0 || cfg.MaxAckPending > 0) {
        return nil, errors.New("durable required when setting MaxDeliver or MaxAckPending")
    }

    // Create or bind pull subscription
    sub, err := s.js.PullSubscribe(subject, durable,
        nats.BindStream(streamName),
        nats.AckWait(cfg.AckWait),
    )
    if err != nil {
        return nil, err
    }
    return &jsConsumer{sub: sub, batchSize: cfg.BatchSize, batchTimeout: cfg.BatchTimeout}, nil
}

func (s *jetStream) Close() error {
	s.nc.Close()
	return nil
}
