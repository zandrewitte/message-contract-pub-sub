package broker

type MessageContract struct {
	Hello HelloMessageTopic
}

func NewMessageContract(broker MessageBroker) MessageContract {
	return MessageContract{
		Hello: NewHelloMessageTopic(broker),
	}
}
