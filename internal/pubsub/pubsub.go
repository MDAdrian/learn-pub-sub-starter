package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	dat, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %s", err)
	}

	err = ch.ExchangeDeclare(exchange, "direct", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error declaring exchange: %w", err)
	}

	msg := amqp.Publishing {
		ContentType: "application/json",
		Body: dat,
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, msg)
	if err != nil {
		return fmt.Errorf("error when publishing: %w", err)
	}

	return nil
}

type SimpleQueueType int

const (
	Durable   SimpleQueueType = 1
	Transient SimpleQueueType = 2
)


func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	// create channel
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("error opening the channel: %w", err)
	}

	durable := false
	autoDelete := false
	exclusive := false
	switch queueType {
		case Durable:
			durable = true
		case Transient:
			autoDelete = true
			exclusive = true
		default:
			return nil, amqp.Queue{}, fmt.Errorf("unknown queue type: %d", queueType)
	}

	qu, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("error when declaring queue: %w", err)
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("error when binding queue: %w", err)
	}

	return ch, qu, nil
}

func SubscribeJSON[T any](
    conn *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType SimpleQueueType, // an enum to represent "durable" or "transient"
    handler func(T),
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("error declaring queue: %w", err)
	}

	deliveryChan, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error consuming channel: %w", err)
	}

	go func() {
		for d := range deliveryChan {
			var msg T

			// 1) Unmarshal body into T
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				fmt.Errorf("consumer: unmarshal error: %w", err)
			}

			// 2) Call handler
			handler(msg)

			// 3) Acknowledge the message to remove it from the queue
			if err := d.Ack(false); err != nil {
				fmt.Errorf("consumer: ack error: %w", err)
			}
		}
	}()

	return nil
}