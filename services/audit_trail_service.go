package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

type AuditTrailService struct {
	esClient *elasticsearch.Client
	index    string
}

// NewAuditTrailService initializes a new AuditTrailService
func NewAuditTrailService(esClient *elasticsearch.Client, index string) *AuditTrailService {
	return &AuditTrailService{
		esClient: esClient,
		index:    index,
	}
}

// LogAction logs an audit trail action
func (a *AuditTrailService) LogAction(traceID, action, resource string, resourceID interface{}, userID int, newData, oldData interface{}) error {
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

	res, err := a.esClient.Index(
		a.index,
		bytes.NewReader(data),
		a.esClient.Index.WithContext(context.Background()),
	)
	if err != nil {
		return fmt.Errorf("failed to index audit log: %w", err)
	}
	defer res.Body.Close()

	log.Printf("Audit trail logged: %+v", auditLog)
	return nil
}
