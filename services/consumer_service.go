package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/streadway/amqp"
)

type ConsumerService struct {
	esClient        *elasticsearch.Client
	index           string
	auditTrailIndex string
	queueName       string
	handlers        map[string]func(map[string]interface{}, map[string]interface{}) error // Event-specific handlers
}

// NewConsumerService initializes a consumer with handlers and audit trail configuration
func NewConsumerService(esClient *elasticsearch.Client, index, auditTrailIndex, queueName string) *ConsumerService {
	service := &ConsumerService{
		esClient:        esClient,
		index:           index,
		auditTrailIndex: auditTrailIndex,
		queueName:       queueName,
		handlers:        make(map[string]func(map[string]interface{}, map[string]interface{}) error),
	}

	// Register standard event handlers
	service.handlers["Created"] = service.handleCreatedOrUpdated
	service.handlers["Updated"] = service.handleCreatedOrUpdated
	service.handlers["Deleted"] = service.handleDeleted
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

// handleCreatedOrUpdated handles "Created" or "Updated" events and logs an audit trail
func (c *ConsumerService) handleCreatedOrUpdated(payload map[string]interface{}, meta map[string]interface{}) error {
	// Extract idField from metadata
	idField, ok := meta["idField"].(string)
	if !ok || idField == "" {
		log.Printf("Invalid or missing idField in meta: %v", meta)
		return fmt.Errorf("invalid or missing idField")
	}

	// Extract action from metadata
	action, ok := meta["action"].(string)
	if !ok || action == "" {
		action = "UPDATE" // Default action if not provided
	}

	// Extract traceID from metadata
	traceID, _ := meta["traceId"].(string)

	// Extract userID from metadata
	userID, _ := meta["userId"].(int)

	// Extract newData and oldData from payload
	newData, ok := payload["newData"].(map[string]interface{})
	if !ok {
		log.Printf("Missing or invalid newData in payload: %v", payload)
		return fmt.Errorf("missing or invalid newData in payload")
	}

	oldData, _ := payload["oldData"].(map[string]interface{}) // oldData is optional

	// Retrieve document ID from newData
	docID, ok := newData[idField]
	if !ok || docID == nil {
		log.Printf("Missing document ID for field %s in newData: %v", idField, newData)
		return fmt.Errorf("missing document ID for field %s in newData", idField)
	}

	// Convert docID to string
	docIDStr, err := convertToString(docID)
	if err != nil {
		log.Printf("Invalid document ID for field %s: %v", idField, err)
		return fmt.Errorf("invalid document ID for field %s: %w", idField, err)
	}

	// Log audit trail
	err = c.logAuditTrail(traceID, action, c.index, docIDStr, int(userID), newData, oldData)
	if err != nil {
		log.Printf("TraceID: %s - Error logging audit trail: %v", traceID, err)
	}

	// Index the updated document
	return c.indexDocument(docIDStr, newData)
}

// convertToString safely converts an interface to a string
func convertToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil // Convert float64 to an integer-like string
	case int:
		return fmt.Sprintf("%d", v), nil
	case int32, int64:
		return fmt.Sprintf("%d", v), nil
	default:
		return "", fmt.Errorf("unsupported type: %T", value)
	}
}

// handleDeleted handles "Deleted" events
func (c *ConsumerService) handleDeleted(payload map[string]interface{}, meta map[string]interface{}) error {
	idField := "id" // Default primary key field

	// Check for custom primary key field in metadata (optional)
	if field, exists := meta["idField"].(string); exists {
		idField = field
	}

	// Retrieve document ID from payload
	docID, ok := payload[idField]
	if !ok || docID == nil {
		log.Printf("Missing document ID for field %s in payload: %v", idField, payload)
		return fmt.Errorf("missing document ID for field %s", idField)
	}

	// Convert docID to string
	docIDStr, err := convertToString(docID)
	if err != nil {
		log.Printf("Invalid document ID for field %s: %v", idField, err)
		return fmt.Errorf("invalid document ID for field %s: %w", idField, err)
	}

	// Extract metadata
	traceID, _ := meta["traceId"].(string)
	userID, _ := meta["userId"].(int)

	// Log the audit trail
	action := "DELETE"
	oldData := payload
	if traceID != "" && userID != 0 {
		if err := c.logAuditTrail(traceID, action, c.index, docIDStr, int(userID), nil, oldData); err != nil {
			log.Printf("Error logging audit trail for document ID %s: %v", docIDStr, err)
		}
	} else {
		log.Printf("Skipping audit trail logging due to missing traceID or userID for document ID %s", docIDStr)
	}

	// Proceed with deleting the document
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

// logAuditTrail logs actions into the audit trail index
func (c *ConsumerService) logAuditTrail(traceID, action, resource, resourceID string, userID int, newData, oldData interface{}) error {
	auditLog := map[string]interface{}{
		"traceId":    traceID,
		"action":     action,
		"resource":   resource,
		"resourceId": resourceID,
		"userId":     userID,
		"newData":    newData,
		"oldData":    oldData,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(auditLog)
	if err != nil {
		return fmt.Errorf("failed to marshal audit log: %w", err)
	}

	res, err := c.esClient.Index(
		c.auditTrailIndex,
		bytes.NewReader(data),
		c.esClient.Index.WithContext(context.Background()),
	)
	if err != nil {
		return fmt.Errorf("failed to index audit log: %w", err)
	}
	defer res.Body.Close()

	log.Printf("Audit trail logged: %+v", auditLog)
	return nil
}
