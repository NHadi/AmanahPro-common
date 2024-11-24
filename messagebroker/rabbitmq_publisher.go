package messagebroker

import (
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
func (p *RabbitMQPublisher) Publish(queueName string, message []byte) error {
	err := p.service.channel.Publish(
		"",        // Default exchange
		queueName, // Queue name
		false,     // Mandatory
		false,     // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	return nil
}
