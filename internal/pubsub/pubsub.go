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