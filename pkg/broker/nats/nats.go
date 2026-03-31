package nats

import (
	"github.com/nats-io/nats.go"
	"github.com/zandrewitte/message-contract-pub-sub/pkg/broker"
	"github.com/zandrewitte/message-contract-pub-sub/pkg/encoding"
)

type Broker struct {
	conn   *nats.Conn
	topics broker.MessageContract
}

func (n *Broker) Topics() broker.MessageContract {
	return n.topics
}

func NewNATSBroker(url string) (*Broker, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	nb := &Broker{conn: nc}
	nb.topics = broker.NewMessageContract(nb)
	return nb, nil
}

func (n *Broker) Publish(topic string, msg any) error {
	jsonPayload, err := n.Encoder().Encode(msg)
	if err != nil {
		return err
	}
	return n.conn.Publish(topic, jsonPayload)
}

func (n *Broker) Subscribe(topic, queue string, handler broker.MessageHandler) error {
	_, err := n.conn.QueueSubscribe(topic, queue, func(msg *nats.Msg) {
		handler(msg.Subject, msg.Data)
	})
	return err
}

func (n *Broker) Close() error {
	return n.conn.Drain()
}

func (n *Broker) Encoder() encoding.MessageEncoder {
	return encoding.NewJSONEncoder()
}
