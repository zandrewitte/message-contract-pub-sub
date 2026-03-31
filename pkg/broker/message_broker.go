package broker

import (
	"github.com/zandrewitte/message-contract-pub-sub/pkg/encoding"
)

type MessageBroker interface {
	Publish(topic string, msg any) error
	Subscribe(topic, queue string, handler MessageHandler) error
	Close() error
	Topics() MessageContract
	Encoder() encoding.MessageEncoder
}

type MessageHandler func(subject string, msg any)
