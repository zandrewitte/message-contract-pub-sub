package broker

type HelloMessage struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

func (h HelloMessage) Key() string {
	return h.UserID
}

type HelloMessageTopic struct {
	Single TypedTopic[HelloMessage]
	Batch  BatchedTopic[HelloMessage]
}

func NewHelloMessageTopic(broker MessageBroker) HelloMessageTopic {
	return HelloMessageTopic{
		Single: TypedTopic[HelloMessage]{
			Broker: broker,
			Format: "hello.user.%s.message",
		},
		Batch: BatchedTopic[HelloMessage]{
			Broker:     broker,
			BatchSize:  100,
			BatchTopic: "hello.user.message.batch",
		},
	}
}
