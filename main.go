package main

import (
	"fmt"
	"time"

	"github.com/zandrewitte/message-contract-pub-sub/pkg/broker"
	"github.com/zandrewitte/message-contract-pub-sub/pkg/broker/nats"
)

func main() {
	nb, err := nats.NewNATSBroker("nats://localhost:4222")
	if err != nil {
		panic(err)
	}
	defer nb.Close()

	go Subscribe(nb)

	for {
		select {
		case <-time.After(time.Second * 2):
			Publish(nb)
		}
	}
}

func Subscribe(mb broker.MessageBroker) {
	singleChan := make(chan broker.HelloMessage, 10)
	if err := mb.Topics().Hello.Single.SubscribeAll("hello-batch-service-group", singleChan); err != nil {
		panic(err)
	}

	batchChan := make(chan broker.BatchMessage[broker.HelloMessage], 10)
	if err := mb.Topics().Hello.Batch.Subscribe("hello-batch-service-group", batchChan); err != nil {
		panic(err)
	}

	for {
		select {
		case single := <-singleChan:
			fmt.Printf("Received single message from %s: %s\n", single.UserID, single.Message)
		case batch := <-batchChan:
			fmt.Printf("Received batch of %d messages\n", len(batch.Messages))
			for _, msg := range batch.Messages {
				fmt.Printf("  - User %s: %s\n", msg.UserID, msg.Message)
			}
		}
	}
}

func Publish(mb broker.MessageBroker) {
	// Single message publish
	if err := mb.Topics().Hello.Single.Publish(broker.HelloMessage{
		UserID:  "user123",
		Message: "Hello, World!",
	}); err != nil {
		panic(err)
	}

	// Batched publish — automatically chunks into batches of 100
	messages := make([]broker.HelloMessage, 0, 250)
	for i := 0; i < 250; i++ {
		messages = append(messages, broker.HelloMessage{
			UserID:  fmt.Sprintf("user%d", i+1),
			Message: fmt.Sprintf("Hello from user%d", i+1),
		})
	}

	if err := mb.Topics().Hello.Batch.Publish(messages); err != nil {
		panic(err)
	}
	fmt.Printf("Published %d messages in batches of 100\n", len(messages))
}
