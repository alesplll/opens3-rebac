package kafka

import (
	"context"
)

type MessageHandler func(ctx context.Context, msg Message) error

type Consumer interface {
	Consume(ctx context.Context, handler MessageHandler) error
}

// PrettyDecoder is function for decoding raw bytes
// to human-read json (string)
type PrettyDecoder func([]byte) (josn string, ok bool)

type Producer interface {
	Send(ctx context.Context, key, value []byte, prettyDecoder PrettyDecoder) error
}
