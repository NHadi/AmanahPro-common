package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/streadway/amqp"
)

type ConsumerService struct {
	esClient  *elasticsearch.Client
	index     string
	queueName string
	handlers  map[string]func(map[string]interface{}, map[string]interface{}) error // Event-specific handlers
}

// NewConsumerService initializes a consumer with handlers
func NewConsumerService(esClient *elasticsearch.Client, index, queueName string) *ConsumerService {
	service := &ConsumerService{
		esClient:  esClient,
		index:     index,
		queueName: queueName,
		handlers:  make(map[string]func(map[string]interface{}, map[string]interface{}) error),
	}

	// Register standard event handlers
	service.handlers["Created"] = service.handleCreatedOrUpdated
	service.handlers["Updated"] = service.handleCreatedOrUpdated
	service.handlers["Deleted"] = service.handleDeleted

	// Register custom event handlers
	service.handlers["Reindexed"] = service.handleCreatedOrUpdated

	return service
}

func (c *ConsumerService) StartConsumer(channel *amqp.Channel, concurrency int) error {
	msgs, err := channel.Consume(
		c.queueName,
		"",    // Consumer tag
		false, // Manual acknowledgment
		false, // Exclusive
		false, // No-local
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	workerChan := make(chan bool, concurrency)

	go func() {
		for msg := range msgs {
			workerChan <- true
			go func(m amqp.Delivery) {
				defer func() { <-workerChan }()
				if err := c.processMessage(m.Body); err != nil {
					log.Printf("Error processing message: %v", err)
					m.Nack(false, true) // Requeue message on failure
				} else {
					m.Ack(false) // Acknowledge successful processing
				}
			}(msg)
		}
	}()

	log.Printf("Consumer is now actively listening to queue: %s", c.queueName)
	select {} // Keep the consumer running
}

// processMessage routes messages to appropriate handlers
func (c *ConsumerService) processMessage(msg []byte) error {
	log.Printf("Processing message from queue %s: %s", c.queueName, string(msg))

	var event struct {
		Event   string                 `json:"event"`
		Payload map[string]interface{} `json:"payload"`
		Meta    map[string]interface{} `json:"meta"` // Optional metadata
	}

	// Parse the message
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// Route to the appropriate handler
	handler, exists := c.handlers[event.Event]
	if !exists {
		log.Printf("Unhandled event type: %s", event.Event)
		return nil // Acknowledge unknown event types to prevent re-delivery
	}

	// Pass both payload and meta to the handler
	return handler(event.Payload, event.Meta)
}

// handleCreatedOrUpdated handles "Created" or "Updated" events
func (c *ConsumerService) handleCreatedOrUpdated(payload map[string]interface{}, meta map[string]interface{}) error {
	idField := "id" // Default primary key field

	// Check for custom primary key field in metadata (optional)
	if field, exists := meta["idField"].(string); exists {
		idField = field
	}

	// Retrieve document ID
	docID, ok := payload[idField].(float64) // JSON numbers are float64
	if !ok {
		return fmt.Errorf("missing or invalid document ID (field: %s) in payload", idField)
	}

	// Convert float64 ID to string for Elasticsearch
	docIDStr := fmt.Sprintf("%.0f", docID)

	log.Printf("Indexing document with ID %s", docIDStr)
	return c.indexDocument(docIDStr, payload)
}

// handleDeleted handles "Deleted" events
func (c *ConsumerService) handleDeleted(payload map[string]interface{}, meta map[string]interface{}) error {
	idField := "id"

	if field, exists := meta["idField"].(string); exists {
		idField = field
	}

	docID, ok := payload[idField].(float64)
	if !ok {
		return fmt.Errorf("missing or invalid document ID (field: %s) in payload", idField)
	}

	docIDStr := fmt.Sprintf("%.0f", docID)

	log.Printf("Deleting document with ID %s", docIDStr)
	return c.deleteDocument(docIDStr)
}

// indexDocument indexes or updates a document in Elasticsearch
func (c *ConsumerService) indexDocument(docID string, document map[string]interface{}) error {
	data, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	res, err := c.esClient.Index(
		c.index,
		bytes.NewReader(data),
		c.esClient.Index.WithDocumentID(docID),
		c.esClient.Index.WithContext(context.Background()),
	)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	log.Printf("Document indexed in %s: %s", c.index, docID)
	return nil
}

// deleteDocument removes a document from Elasticsearch
func (c *ConsumerService) deleteDocument(docID string) error {
	res, err := c.esClient.Delete(
		c.index,
		docID,
		c.esClient.Delete.WithContext(context.Background()),
	)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer res.Body.Close()

	log.Printf("Document deleted from %s: %s", c.index, docID)
	return nil
}
