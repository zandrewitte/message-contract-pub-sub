package broker

import (
	"fmt"
)

type KeyedMessage interface {
	Key() string
}

type TypedTopic[T KeyedMessage] struct {
	Broker MessageBroker
	Format string // e.g., "hello.user.%s.message"
}

func (t TypedTopic[T]) Topic(key string) string {
	return fmt.Sprintf(t.Format, key)
}

func (t TypedTopic[T]) Publish(msg T) error {
	return t.Broker.Publish(t.Topic(msg.Key()), msg)
}

func (t TypedTopic[T]) Subscribe(key string, groupName string, ch chan<- T) error {
	return t.Broker.Subscribe(t.Topic(key), groupName, func(subject string, msgAny any) {
		var msg T
		err := t.Broker.Encoder().Decode(msgAny.([]byte), &msg)
		if err != nil {
			panic(err)
		}
		ch <- msg
	})
}

func (t TypedTopic[T]) SubscribeAll(groupName string, ch chan<- T) error {
	return t.Subscribe("*", groupName, ch)
}

type BatchMessage[T any] struct {
	Messages []T
}

type BatchedTopic[T KeyedMessage] struct {
	Broker     MessageBroker
	BatchSize  int
	BatchTopic string
}

func (b BatchedTopic[T]) Publish(messages []T) error {
	for i := 0; i < len(messages); i += b.BatchSize {
		end := i + b.BatchSize
		if end > len(messages) {
			end = len(messages)
		}
		batch := BatchMessage[T]{Messages: messages[i:end]}
		if err := b.Broker.Publish(b.BatchTopic, batch); err != nil {
			return err
		}
	}
	return nil
}

func (b BatchedTopic[T]) Subscribe(groupName string, ch chan<- BatchMessage[T]) error {
	return b.Broker.Subscribe(b.BatchTopic, groupName, func(subject string, msgAny any) {
		var message BatchMessage[T]
		_ = b.Broker.Encoder().Decode(msgAny.([]byte), &message)
		ch <- message
	})
}
