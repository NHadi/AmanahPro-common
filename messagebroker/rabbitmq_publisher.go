package messagebroker

import (
	"encoding/json"
	"fmt"

	"github.com/streadway/amqp"
)

type RabbitMQPublisher struct {
	service *RabbitMQService
}

// NewRabbitMQPublisher creates a new publisher
func NewRabbitMQPublisher(service *RabbitMQService) *RabbitMQPublisher {
	return &RabbitMQPublisher{service: service}
}

// Publish sends a message to the specified queue
func (p *RabbitMQPublisher) Publish(queueName string, message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	err = p.service.channel.Publish(
		"",        // Default exchange
		queueName, // Queue name
		false,     // Mandatory
		false,     // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	return nil
}
