package pubsub

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sergioferg/gochat/internal/routing"
	"github.com/sirupsen/logrus"
)

type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

// New connects to RabbitMQ and sets up the required exchanges.
func New(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(
		routing.ChatPrefix,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	logrus.Info("Successfully connected to RabbitMQ and declared chat_events exchange")

	return &RabbitMQ{
		Conn:    conn,
		Channel: ch,
	}, nil
}

func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}
