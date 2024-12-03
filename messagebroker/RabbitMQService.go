package messagebroker

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

type RabbitMQService struct {
	Conn          *amqp.Connection
	Channel       *amqp.Channel
	connURL       string
	notifyClose   chan *amqp.Error
	reconnectWait time.Duration
	queueNames    []string
	onReconnect   func()
	mutex         sync.Mutex // Protects reconnection and channel reinitialization
}

// NewRabbitMQService initializes RabbitMQ with auto-reconnection and queue declaration.
func NewRabbitMQService(connURL string, queueNames []string) (*RabbitMQService, error) {
	service := &RabbitMQService{
		connURL:       connURL,
		reconnectWait: 5 * time.Second,
		queueNames:    queueNames,
	}
	if err := service.connect(); err != nil {
		return nil, err
	}
	go service.handleReconnect()
	return service, nil
}

// connect establishes a new RabbitMQ connection and channel.
func (s *RabbitMQService) connect() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	conn, err := amqp.Dial(s.connURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to create RabbitMQ channel: %v", err)
	}

	s.Conn = conn
	s.Channel = channel
	s.notifyClose = make(chan *amqp.Error, 1)
	s.Conn.NotifyClose(s.notifyClose)

	log.Println("RabbitMQ connected successfully")

	// Declare queues upon connection
	if err := s.DeclareQueues(); err != nil {
		return fmt.Errorf("failed to declare queues: %v", err)
	}
	return nil
}

// DeclareQueues declares all required queues.
func (s *RabbitMQService) DeclareQueues() error {
	for _, queueName := range s.queueNames {
		if err := s.DeclareQueue(queueName); err != nil {
			return fmt.Errorf("failed to declare queue '%s': %w", queueName, err)
		}
	}
	log.Println("RabbitMQ queues declared successfully")
	return nil
}

// DeclareQueue declares a single queue.
func (s *RabbitMQService) DeclareQueue(queueName string) error {
	_, err := s.Channel.QueueDeclare(
		queueName,
		true,  // Durable
		false, // Auto-delete
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}
	return nil
}

// NewChannel creates and returns a new channel for publishers or consumers.
func (s *RabbitMQService) NewChannel() (*amqp.Channel, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Conn == nil || s.Conn.IsClosed() {
		return nil, amqp.ErrClosed
	}
	return s.Conn.Channel()
}

// Close cleans up the RabbitMQ connection and channel.
func (s *RabbitMQService) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Channel != nil {
		_ = s.Channel.Close()
	}
	if s.Conn != nil {
		_ = s.Conn.Close()
	}
	log.Println("RabbitMQ connection and channel closed")
}

// handleReconnect listens for connection closure and attempts to reconnect.
func (s *RabbitMQService) handleReconnect() {
	for {
		err := <-s.notifyClose
		if err != nil {
			log.Printf("RabbitMQ connection lost: %v. Attempting to reconnect...", err)
		}

		for {
			log.Println("Attempting to reconnect to RabbitMQ...")
			if err := s.connect(); err != nil {
				log.Printf("RabbitMQ reconnection failed: %v. Retrying in %s...", err, s.reconnectWait)
				time.Sleep(s.reconnectWait)
			} else {
				log.Println("RabbitMQ reconnected successfully. Reinitializing queues and consumers.")
				if s.onReconnect != nil {
					s.onReconnect()
				}
				break
			}
		}
	}
}

// SetOnReconnect sets or updates the onReconnect callback
func (s *RabbitMQService) SetOnReconnect(callback func()) {
	s.onReconnect = func() {
		log.Println("RabbitMQ reconnected. Reinitializing queues...")
		if err := s.DeclareQueues(); err != nil {
			log.Printf("Failed to redeclare RabbitMQ queues after reconnect: %v", err)
		}
		if callback != nil {
			callback()
		}
	}
}
