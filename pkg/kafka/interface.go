package kafka

// IProducer defines the interface for Kafka producer.
// Implementations are safe for concurrent use.
type IProducer interface {
	Publish(key, value []byte) error
	Close() error
	HealthCheck() error
}

// NewProducer creates a new Kafka producer. Returns the interface.
func NewProducer(cfg Config) (IProducer, error) {
	if err := validateProducerConfig(cfg); err != nil {
		return nil, err
	}
	return newProducerImpl(cfg)
}
