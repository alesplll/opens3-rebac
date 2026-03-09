package producer

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/kafka"
	"go.uber.org/zap"
)

type Logger interface {
	Info(ctx context.Context, msg string, fields ...zap.Field)
	Error(ctx context.Context, msg string, fields ...zap.Field)
}

type producer struct {
	syncProducer sarama.SyncProducer
	topic        string
	logger       Logger
}

func NewProducer(syncProducer sarama.SyncProducer, topic string, logger Logger) kafka.Producer {
	return &producer{
		syncProducer: syncProducer,
		topic:        topic,
		logger:       logger,
	}
}

func (p *producer) Send(ctx context.Context, key, value []byte, pretty kafka.PrettyDecoder) error {
	partition, offset, err := p.syncProducer.SendMessage(&sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	})
	if err != nil {
		p.logger.Error(ctx, "Failed to send message", zap.Error(err))
		return err
	}

	fields := []zap.Field{
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
		zap.ByteString("key", key),
		zap.Int("value_size", len(value)),
	}
	if pretty != nil {
		if js, ok := pretty(value); ok {
			fields = append(fields, zap.String("value_json", js))
		} else {
			fields = append(fields, zap.Binary("value_raw", value))
		}
	} else {
		fields = append(fields, zap.Binary("value_raw", value))
	}
	p.logger.Info(ctx, "Message sent", fields...)
	return nil
}
