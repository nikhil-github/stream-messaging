package mocks

//go:generate mockgen -destination=./mock_stream.go -package=mocks stream-messaging/broker Client
//go:generate mockgen -destination=./mock_publisher.go -package=mocks stream-messaging/broker Publisher
//go:generate mockgen -destination=./mock_consumer.go -package=mocks stream-messaging/broker Consumer
