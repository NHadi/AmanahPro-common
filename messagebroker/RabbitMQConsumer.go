package messagebroker

import (
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

type RabbitMQConsumer struct {
	service *RabbitMQService
}

// NewRabbitMQConsumer creates a new consumer
func NewRabbitMQConsumer(service *RabbitMQService) *RabbitMQConsumer {
	return &RabbitMQConsumer{service: service}
}

// Consume starts listening to messages and processes them with a handler
func (c *RabbitMQConsumer) Consume(queueName string, handler func(msg amqp.Delivery) error) error {
	msgs, err := c.service.Channel.Consume(
		queueName,
		"",    // Consumer tag
		true,  // Auto-ack
		false, // Exclusive
		false, // No-local
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %v", err)
	}

	go func() {
		for msg := range msgs {
			if err := handler(msg); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}()

	log.Printf("Started consuming messages from queue: %s", queueName)
	return nil
}
