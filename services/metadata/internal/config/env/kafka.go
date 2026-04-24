package env

import (
	"github.com/caarlos0/env/v11"
)

type kafkaEnvConfig struct {
	Bootstrap          string `env:"KAFKA_BOOTSTRAP,required"`
	ObjectDeletedTopic string `env:"KAFKA_OBJECT_DELETED_TOPIC,required"`
	BucketDeletedTopic string `env:"KAFKA_BUCKET_DELETED_TOPIC,required"`
}

type kafkaConfig struct {
	raw kafkaEnvConfig
}

func NewKafkaConfig() (*kafkaConfig, error) {
	var raw kafkaEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	return &kafkaConfig{raw: raw}, nil
}

func (cfg *kafkaConfig) BootstrapServers() string {
	return cfg.raw.Bootstrap
}

func (cfg *kafkaConfig) ObjectDeletedTopic() string {
	return cfg.raw.ObjectDeletedTopic
}

func (cfg *kafkaConfig) BucketDeletedTopic() string {
	return cfg.raw.BucketDeletedTopic
}
