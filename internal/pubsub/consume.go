package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	Durable SimpleQueueType = iota
	Transient
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	connChannel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("failed to open channel: %w", err)
	}

	// For Transient queues (like our WebSockets server), we want RabbitMQ to
	// auto-delete the queue when the server disconnects, and keep it exclusive to this server.
	isDurable := queueType == Durable
	isTransient := queueType == Transient

	connQueue, err := connChannel.QueueDeclare(
		queueName,   // name (if "", RabbitMQ generates a random one)
		isDurable,   // durable
		isTransient, // delete when unused
		isTransient, // exclusive
		false,       // no-wait
		nil,         // args (removed the hardcoded dead-letter exchange)
	)
	if err != nil {
		connChannel.Close()
		return nil, amqp.Queue{}, fmt.Errorf("queue declare failed: %w", err)
	}

	err = connChannel.QueueBind(
		connQueue.Name, // queue name (use the generated name)
		key,            // routing key
		exchange,       // exchange
		false,          // no-wait
		nil,            // args
	)
	if err != nil {
		connChannel.Close()
		return nil, amqp.Queue{}, fmt.Errorf("queue bind failed: %w", err)
	}

	return connChannel, connQueue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	ch, queue, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		queueType,
	)
	if err != nil {
		return fmt.Errorf("failed to declare and bind queue: %v", err)
	}

	fmt.Printf("Queue %v declared and bound to exchange %v!\n", queue.Name, exchange)

	err = ch.Qos(10, 0, false)
	if err != nil {
		return fmt.Errorf("failed to establish prefetch limit: %v", err)
	}

	msgs, err := ch.Consume(
		queue.Name, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %v", err)
	}

	// Spin up the background worker
	go func() {
		defer ch.Close()
		for msg := range msgs {
			var target T
			err := json.Unmarshal(msg.Body, &target)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				msg.Nack(false, false) // discard bad JSON
				continue
			}

			ack := handler(target)
			switch ack {
			case Ack:
				msg.Ack(false)
			case NackRequeue:
				msg.Nack(false, true)
			case NackDiscard:
				msg.Nack(false, false)
			default:
				msg.Ack(false)
			}
		}
	}()

	return nil
}
