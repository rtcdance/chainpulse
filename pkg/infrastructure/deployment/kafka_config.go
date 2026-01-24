package deployment

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers              []string
	Topic                string
	ConsumerGroup        string
	Partitions           int
	ReplicationFactor    int
	RetentionMs          int64
	CompressionType      string
	SecurityProtocol     string
	SASLMechanism        string
	SASLUsername         string
	SASLPassword         string
	SSLCALocation        string
	SSLCertLocation      string
	SSLKeyLocation       string
	SSLKeyPassword       string
	ConnectTimeoutMs     int
	RequestTimeoutMs     int
	SessionTimeoutMs     int
	HeartbeatIntervalMs  int
	MaxPollIntervalMs    int
	MaxPollRecords       int
	FetchMinBytes        int
	FetchMaxWaitMs       int
}

// NewKafkaConfig creates a new Kafka configuration with defaults
func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		Brokers:             []string{"localhost:9092"},
		Topic:               "events",
		ConsumerGroup:       "chainpulse-consumer",
		Partitions:          3,
		ReplicationFactor:   1,
		RetentionMs:         604800000, // 7 days
		CompressionType:     "snappy",
		SecurityProtocol:    "PLAINTEXT",
		ConnectTimeoutMs:    10000,
		RequestTimeoutMs:    30000,
		SessionTimeoutMs:    10000,
		HeartbeatIntervalMs: 3000,
		MaxPollIntervalMs:   300000,
		MaxPollRecords:      500,
		FetchMinBytes:       1,
		FetchMaxWaitMs:      500,
	}
}

// Validate validates the Kafka configuration
func (k *KafkaConfig) Validate() error {
	if len(k.Brokers) == 0 {
		k.Brokers = []string{"localhost:9092"}
	}
	if k.Topic == "" {
		k.Topic = "events"
	}
	if k.ConsumerGroup == "" {
		k.ConsumerGroup = "chainpulse-consumer"
	}
	if k.Partitions == 0 {
		k.Partitions = 3
	}
	if k.ReplicationFactor == 0 {
		k.ReplicationFactor = 1
	}
	if k.CompressionType == "" {
		k.CompressionType = "snappy"
	}
	if k.SecurityProtocol == "" {
		k.SecurityProtocol = "PLAINTEXT"
	}
	return nil
}
