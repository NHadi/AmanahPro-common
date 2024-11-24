package messagebroker

import (
	"fmt"

	"github.com/streadway/amqp"
)

type RabbitMQService struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQService initializes RabbitMQ connection and channel
func NewRabbitMQService(connURL string) (*RabbitMQService, error) {
	conn, err := amqp.Dial(connURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ channel: %v", err)
	}

	return &RabbitMQService{conn: conn, channel: channel}, nil
}

// DeclareQueue declares a queue if it doesn't exist
func (s *RabbitMQService) DeclareQueue(queueName string) error {
	_, err := s.channel.QueueDeclare(
		queueName,
		true,  // Durable
		false, // Auto-delete
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	return err
}

// Close closes RabbitMQ connection and channel
func (s *RabbitMQService) Close() {
	s.channel.Close()
	s.conn.Close()
}
