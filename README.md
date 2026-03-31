# Message Contracts on Top of a Pub/Sub Message Broker

A demonstration of how to build **type-safe, self-documenting message contracts** on top of a pub/sub message broker using Go and NATS.

---

## The Problem with Raw Pub/Sub

Pub/sub systems like NATS, Kafka, and RabbitMQ are powerful, but they're inherently untyped. At their core, they move raw bytes around. Publishers and subscribers must agree on:

- Which topics exist
- What message format to use on each topic
- How to serialize and deserialize messages

Without any structure, this implicit agreement lives only in developer heads and documentation, which quickly goes stale. Teams end up with:

- **Silent failures**: wrong message shape, no compile-time error
- **Topic sprawl**: no central catalog of what topics exist
- **Boilerplate**: every handler manually deserializes bytes
- **Tight coupling**: producers and consumers must coordinate out-of-band

The solution is a **message contract** layer that makes these agreements explicit and enforceable at compile time.

---

## What Is a Message Contract?

A message contract is a shared definition that specifies:

1. **What messages exist** — their fields and types
2. **Which topics carry them** — the topic naming scheme
3. **How to publish and subscribe** — with automatic serialization

In this project the contract is a plain Go struct that wires everything together:

```go
// pkg/broker/contract.go
type MessageContract struct {
    Hello HelloMessageTopic
}
```

The broker exposes this contract via its `Topics()` method, so callers never need to hard-code topic strings:

```go
broker.Topics().Hello.Single.Publish(msg)
broker.Topics().Hello.Batch.Publish(messages)
```

Everything flows through typed wrappers — the compiler catches mismatches before the code ever runs.

---

## Project Structure

```
message-contract-pub-sub/
├── main.go                     # Demo: publisher + subscriber
├── go.mod
└── pkg/
    ├── broker/
    │   ├── message_broker.go   # Core MessageBroker interface
    │   ├── contract.go         # MessageContract definition
    │   ├── types.go            # Generic TypedTopic / BatchedTopic
    │   ├── hello_message.go    # Example message and topic definitions
    │   └── nats/
    │       └── nats.go         # NATS implementation of MessageBroker
    └── encoding/
        └── encoder.go          # Pluggable serialization (JSON)
```

---

## Core Concepts

### 1. The MessageBroker Interface

The entire system depends on this abstraction, not on NATS directly:

```go
type MessageBroker interface {
    Publish(topic string, msg any) error
    Subscribe(topic, queue string, handler MessageHandler) error
    Close() error
    Topics() MessageContract
    Encoder() encoding.MessageEncoder
}

type MessageHandler func(subject string, msg any)
```

Swapping NATS for Kafka or RabbitMQ means writing a new struct that satisfies this interface — no other code changes.

---

### 2. Keyed Messages

Every message type must implement a single method:

```go
type KeyedMessage interface {
    Key() string
}
```

The key drives the topic name. For a `HelloMessage` the key is the `UserID`, which produces a per-user topic:

```go
type HelloMessage struct {
    UserID  string `json:"user_id"`
    Message string `json:"message"`
}

func (h HelloMessage) Key() string {
    return h.UserID
}
```

A message from `user123` is routed to `hello.user.user123.message`, while `user456` gets its own topic. This keeps messages isolated per entity and makes wildcard subscriptions simple.

---

### 3. TypedTopic — Single Message Pub/Sub

`TypedTopic[T]` wraps a raw broker and adds compile-time type safety and automatic JSON serialization:

```go
type TypedTopic[T KeyedMessage] struct {
    Broker  MessageBroker
    Format  string  // e.g. "hello.user.%s.message"
}
```

**Publishing:**

```go
topics.Hello.Single.Publish(HelloMessage{UserID: "user123", Message: "hi"})
// publishes JSON to "hello.user.user123.message"
```

**Subscribing to a specific key:**

```go
ch := make(chan HelloMessage)
topics.Hello.Single.Subscribe("user123", "my-service-group", ch)
// receives messages from "hello.user.user123.message"
```

**Subscribing to all keys:**

```go
topics.Hello.Single.SubscribeAll("my-service-group", ch)
// receives messages from "hello.user.*.message"
```

The channel receives already-decoded `HelloMessage` structs — no manual deserialization.

---

### 4. BatchedTopic — Automatic Chunking

`BatchedTopic[T]` solves the case where a producer wants to send many messages at once without hammering the broker with individual publishes.

```go
type BatchedTopic[T KeyedMessage] struct {
    Broker     MessageBroker
    BatchSize  int
    BatchTopic string
}
```

Publishing 250 messages with a `BatchSize` of 100 automatically creates three broker publishes: two of 100 and one of 50:

```go
topics.Hello.Batch.Publish(twoHundredFiftyMessages)
// publishes 3 BatchMessage payloads to "hello.user.message.batch"
```

Subscribers receive `BatchMessage[HelloMessage]` and iterate the slice, keeping per-message logic simple while reducing network round-trips.

---

### 5. Consumer Groups (Queue Groups)

Both topic types accept a `groupName` when subscribing. This maps directly to NATS queue groups (and equivalents in Kafka consumer groups, RabbitMQ competing consumers):

- All subscribers **within the same group** share delivery; Each message goes to exactly one of them. This is the horizontal scaling model.
- Subscribers **in different groups** each receive every message independently. This allows multiple processing pipelines on the same event stream.

```go
// Two instances of the same service share load
topics.Hello.Single.SubscribeAll("payment-service-group", ch)
topics.Hello.Single.SubscribeAll("payment-service-group", ch)

// An audit service also receives every message
topics.Hello.Single.SubscribeAll("audit-service-group", auditCh)
```

---

### 6. Pluggable Encoding

Serialization lives behind an interface:

```go
type MessageEncoder interface {
    Encode(v any) ([]byte, error)
    Decode(data []byte, v any) error
}
```

The default implementation uses `encoding/json`. Switching to Protobuf, Avro, or MessagePack is a one-line change when constructing the broker — none of the contract or topic code changes.

---

## Putting It All Together

Here is the full flow from `main.go`, condensed:

```go
func main() {
    broker, _ := nats.NewNATSBroker("nats://localhost:4222")
    defer broker.Close()

    go subscribe(broker)
    publish(broker)
}

func subscribe(broker broker.MessageBroker) {
    singleCh := make(chan broker.HelloMessage)
    batchCh  := make(chan broker.BatchMessage[broker.HelloMessage])

    broker.Topics().Hello.Single.SubscribeAll("demo-group", singleCh)
    broker.Topics().Hello.Batch.Subscribe("demo-group", batchCh)

    for {
        select {
        case msg := <-singleCh:
            fmt.Printf("single: %+v\n", msg)
        case batch := <-batchCh:
            fmt.Printf("batch of %d messages\n", len(batch.Messages))
        }
    }
}

func publish(broker broker.MessageBroker) {
    for {
        broker.Topics().Hello.Single.Publish(
            HelloMessage{UserID: "user123", Message: "hello"},
        )

        broker.Topics().Hello.Batch.Publish(make250Messages())

        time.Sleep(2 * time.Second)
    }
}
```

No raw topic strings. No manual JSON marshalling. No type assertions. The contract enforces correctness at compile time.

---

## Running the Demo

**Prerequisites:**
- Go 1.21+
- A running NATS server

Start NATS with Docker:

```bash
docker run -p 4222:4222 nats:latest
```

Run the demo:

```bash
go run main.go
```

You will see single messages and batches printed as they flow through the broker.

---

## Extending to Other Brokers

Adding a new message type is a three-step process:

1. **Define the message** — implement `Key() string`
2. **Define the topic** — create a `HelloMessageTopic`-style struct with `TypedTopic` / `BatchedTopic` fields
3. **Add it to the contract** — add a field to `MessageContract`

Consumers and producers immediately gain access through `broker.Topics()` with full type safety.

Adding a new broker (e.g. Kafka) means implementing the `MessageBroker` interface. The contracts, topic wrappers, and application code remain unchanged.

---

## Key Takeaways

| Without Contracts | With Contracts |
|---|---|
| Topic names as magic strings | Typed accessors via `broker.Topics()` |
| Manual JSON marshal/unmarshal | Automatic serialization in topic wrapper |
| No central catalog of topics | `MessageContract` documents all topics |
| Runtime type errors | Compile-time type errors |
| Scattered consumer group names | Encapsulated in topic definitions |
| Broker-specific code everywhere | Broker hidden behind `MessageBroker` interface |

Message contracts do not add overhead — they are a zero-cost abstraction in Go. The generated code is identical to what you would write by hand, but now the compiler verifies it for you.
