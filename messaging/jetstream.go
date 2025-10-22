package messaging

import (
    "fmt"
    "log/slog"
    "os"

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
	_, err := s.js.StreamInfo(streamName)
	if err == nil {
		s.logger.Info("Stream exists", "stream", streamName)
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

func (s *jetStream) Close() error {
	s.nc.Close()
	return nil
}
