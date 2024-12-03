package messagebroker

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

type RabbitMQPublisher struct {
	service       *RabbitMQService
	retryInterval time.Duration
	messageQueue  []queuedMessage
	queueMutex    sync.Mutex
	paused        bool
}

type queuedMessage struct {
	QueueName string
	Message   []byte
}

// NewRabbitMQPublisher creates a new publisher
func NewRabbitMQPublisher(service *RabbitMQService) *RabbitMQPublisher {
	publisher := &RabbitMQPublisher{
		service:       service,
		retryInterval: 5 * time.Second,
		messageQueue:  make([]queuedMessage, 0),
		paused:        false,
	}

	// Pause publishing during RabbitMQ reconnections
	service.SetOnReconnect(func() {
		publisher.pausePublishing()
		service.DeclareQueues()
		publisher.resumePublishing()
	})

	go publisher.retryMessages()
	return publisher
}

// Publish sends a message to the specified queue
func (p *RabbitMQPublisher) Publish(queueName string, message []byte) error {
	if p.paused {
		p.queueMessage(queueName, message)
		return fmt.Errorf("publishing paused due to RabbitMQ reconnection")
	}

	err := p.service.Channel.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)

	if err != nil {
		log.Printf("Failed to publish message: %v. Queuing for retry.", err)
		p.queueMessage(queueName, message)
		return fmt.Errorf("failed to publish message: %v", err)
	}

	return nil
}

// PublishEvent marshals an event and publishes it
func (p *RabbitMQPublisher) PublishEvent(queueName string, event interface{}) error {
	message, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return p.Publish(queueName, message)
}

// retryMessages retries publishing queued messages
func (p *RabbitMQPublisher) retryMessages() {
	backoff := p.retryInterval
	maxBackoff := 30 * time.Second

	for {
		time.Sleep(backoff)
		if p.paused {
			continue
		}

		p.queueMutex.Lock()
		queue := p.messageQueue
		p.messageQueue = nil // Clear the queue
		p.queueMutex.Unlock()

		if len(queue) == 0 {
			backoff = p.retryInterval // Reset backoff
			continue
		}

		log.Printf("Retrying %d queued messages...", len(queue))
		for _, msg := range queue {
			if err := p.Publish(msg.QueueName, msg.Message); err != nil {
				p.queueMessage(msg.QueueName, msg.Message)
			}
		}

		// Increase backoff for subsequent retries, up to maxBackoff
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// queueMessage adds a message to the retry queue
func (p *RabbitMQPublisher) queueMessage(queueName string, message []byte) {
	p.queueMutex.Lock()
	defer p.queueMutex.Unlock()

	for _, msg := range p.messageQueue {
		if string(msg.Message) == string(message) && msg.QueueName == queueName {
			log.Printf("Message already queued, skipping duplicate for queue: %s", queueName)
			return
		}
	}

	p.messageQueue = append(p.messageQueue, queuedMessage{
		QueueName: queueName,
		Message:   message,
	})
}

// pausePublishing pauses message publishing
func (p *RabbitMQPublisher) pausePublishing() {
	p.queueMutex.Lock()
	defer p.queueMutex.Unlock()
	p.paused = true
	log.Println("Publishing paused during RabbitMQ reconnection.")
}

func (p *RabbitMQPublisher) resumePublishing() {
	p.queueMutex.Lock()
	defer p.queueMutex.Unlock()
	p.paused = false
	log.Println("Publishing resumed after RabbitMQ reconnection. Processing queued messages immediately.")
	go p.retryMessages() // Trigger immediate retry for queued messages
}
