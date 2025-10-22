package messaging

import (
    "fmt"
    "log/slog"
    "os"
    "time"

    "github.com/nats-io/nats.go"
)

type jetStream struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	logger *slog.Logger
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

    defaultLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return &jetStream{nc: nc, js: js, logger: defaultLogger}, nil
}

func (s *jetStream) WithLogger(logger *slog.Logger) Stream {
	s.logger = logger
	return s
}

func (s *jetStream) EnsureStream(streamName string, subjects []string) error {
    info, err := s.js.StreamInfo(streamName)
    if err == nil {
        // Merge subjects if new ones are provided
        existing := map[string]struct{}{}
        for _, sub := range info.Config.Subjects {
            existing[sub] = struct{}{}
        }
        changed := false
        for _, sub := range subjects {
            if _, ok := existing[sub]; !ok {
                info.Config.Subjects = append(info.Config.Subjects, sub)
                changed = true
            }
        }
        if changed {
            _, err = s.js.UpdateStream(&info.Config)
            if err != nil {
                return err
            }
            s.logger.Info("Stream subjects updated", "stream", streamName, "subjects", info.Config.Subjects)
        } else {
            s.logger.Info("Stream exists", "stream", streamName)
        }
        return nil
    }
    if err == nats.ErrStreamNotFound {
        _, err = s.js.AddStream(&nats.StreamConfig{
            Name:     streamName,
            Subjects: subjects,
            Storage:  nats.FileStorage,
            Replicas: 1,
        })
        if err == nil {
            s.logger.Info("Stream created", "stream", streamName, "subjects", subjects)
        }
    }
    return err
}

func (s *jetStream) NewPublisher(streamName string) (Publisher, error) {
	return &jsPublisher{js: s.js, stream: streamName, logger: s.logger}, nil
}

func (s *jetStream) NewConsumer(streamName string, cfg ConsumerConfig) (Consumer, error) {
	sub, err := s.js.PullSubscribe(cfg.Subject, cfg.Durable,
		nats.BindStream(streamName),
		nats.AckWait(cfg.AckWait),
	)
	if err != nil {
		return nil, err
	}
	return &jsConsumer{sub: sub, batchSize: cfg.BatchSize, logger: s.logger}, nil
}

// EnsureConsumer creates or updates a durable pull consumer with provided options.
func (s *jetStream) EnsureConsumer(streamName string, durable string, opts ConsumerOptions) error {
    // Try to get existing consumer
    if _, err := s.js.ConsumerInfo(streamName, durable); err == nil {
        // Update limited fields if needed (nats.go requires full config). Fetch current, modify, update.
        info, err := s.js.ConsumerInfo(streamName, durable)
        if err != nil { return err }
        cfg := info.Config
        changed := false
        if opts.FilterSubject != "" && cfg.FilterSubject != opts.FilterSubject {
            cfg.FilterSubject = opts.FilterSubject
            changed = true
        }
        if opts.AckWait > 0 && cfg.AckWait != opts.AckWait {
            cfg.AckWait = opts.AckWait
            changed = true
        }
        if opts.MaxDeliver > 0 && cfg.MaxDeliver != opts.MaxDeliver {
            cfg.MaxDeliver = opts.MaxDeliver
            changed = true
        }
        if len(opts.Backoff) > 0 {
            // Convert []time.Duration to []nats.Backoff (alias is time.Duration slice)
            var bo []time.Duration
            for _, d := range opts.Backoff { bo = append(bo, d) }
            // Only set if different length or any value differs
            if len(cfg.BackOff) != len(bo) {
                cfg.BackOff = bo
                changed = true
            } else {
                diff := false
                for i := range bo { if cfg.BackOff[i] != bo[i] { diff = true; break } }
                if diff { cfg.BackOff = bo; changed = true }
            }
        }
        if changed {
            _, err = s.js.UpdateConsumer(streamName, &cfg)
            if err != nil { return err }
            s.logger.Info("Consumer updated", "stream", streamName, "durable", durable)
        } else {
            s.logger.Info("Consumer exists", "stream", streamName, "durable", durable)
        }
        return nil
    } else if err != nats.ErrConsumerNotFound {
        return err
    }

    // Create new consumer
    cfg := &nats.ConsumerConfig{
        Durable:       durable,
        AckPolicy:     nats.AckExplicitPolicy,
        FilterSubject: opts.FilterSubject,
        AckWait:       opts.AckWait,
    }
    if opts.MaxDeliver > 0 { cfg.MaxDeliver = opts.MaxDeliver }
    if len(opts.Backoff) > 0 { cfg.BackOff = opts.Backoff }
    if _, err := s.js.AddConsumer(streamName, cfg); err != nil {
        return err
    }
    s.logger.Info("Consumer created", "stream", streamName, "durable", durable, "filter", opts.FilterSubject)
    return nil
}

func (s *jetStream) Close() error {
	s.nc.Close()
	return nil
}
