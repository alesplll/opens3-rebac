package consumer

import (
	"github.com/IBM/sarama"
	"go.uber.org/zap"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/kafka"
)

// Middleware — функция middleware для дополнительной обработки.
type Middleware func(next kafka.MessageHandler) kafka.MessageHandler

// groupHandler — обёртка для sarama.ConsumerGroupHandler
type groupHandler struct {
	handler kafka.MessageHandler
	logger  Logger
}

// NewGroupHandler создаёт новый groupHandler с middleware цепочкой.
func NewGroupHandler(handler kafka.MessageHandler, logger Logger, middlewares ...Middleware) *groupHandler {
	// Применяем middleware цепочку
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return &groupHandler{
		handler: handler,
		logger:  logger,
	}
}

func (g *groupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (g *groupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (g *groupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				g.logger.Info(session.Context(), "Kafka message channel closed")
				return nil
			}

			msg := kafka.Message{
				Key:            message.Key,
				Value:          message.Value,
				Topic:          message.Topic,
				Partition:      message.Partition,
				Offset:         message.Offset,
				Timestamp:      message.Timestamp,
				BlockTimestamp: message.BlockTimestamp,
				Headers:        extractHeaders(message.Headers),
			}

			if err := g.handler(session.Context(), msg); err != nil {
				g.logger.Error(session.Context(), "Kafka handler error", zap.Error(err))
				continue
			}

			session.MarkMessage(message, "")

		case <-session.Context().Done():
			g.logger.Info(session.Context(), "Kafka session context done")
			return nil
		}
	}
}

func extractHeaders(headers []*sarama.RecordHeader) map[string][]byte {
	result := make(map[string][]byte)
	for _, h := range headers {
		if h != nil && h.Key != nil {
			result[string(h.Key)] = h.Value
		}
	}

	return result
}
