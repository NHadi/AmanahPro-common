package middleware

import (
	"context"
	"fmt"
	"log"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitializeLogger sets up a Zap logger with Elasticsearch integration.
func InitializeLogger(serviceName, elasticURL, indexName string) (*zap.Logger, error) {
	// Create an Elasticsearch client
	client, err := elastic.NewClient(
		elastic.SetURL(elasticURL),    // Set the Elasticsearch URL
		elastic.SetSniff(false),       // Disable sniffing
		elastic.SetHealthcheck(false), // Disable health checks to avoid the ticker issue
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Test connection to Elasticsearch
	info, code, err := client.Ping(elasticURL).Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to ping Elasticsearch: %w", err)
	}
	log.Printf("Connected to Elasticsearch [Version: %s] with status code %d\n", info.Version.Number, code)

	// Define an Elasticsearch syncer
	esSyncer := &elasticSyncer{
		client: client,
		index:  indexName,
	}

	// Configure Zap encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.LevelKey = "level"
	encoderConfig.MessageKey = "message"
	encoderConfig.CallerKey = "caller"

	// Create a Zap core with Elasticsearch sync
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(esSyncer),
		zapcore.InfoLevel, // Adjust logging level as needed
	)

	// Create the logger
	logger := zap.New(core, zap.Fields(zap.String("service", serviceName)))

	return logger, nil
}

// elasticSyncer sends logs to Elasticsearch
type elasticSyncer struct {
	client *elastic.Client
	index  string
}

func (es *elasticSyncer) Write(p []byte) (int, error) {
	// Send log entry to Elasticsearch
	_, err := es.client.Index().
		Index(es.index).
		BodyString(string(p)).
		Do(context.Background())
	if err != nil {
		return 0, fmt.Errorf("failed to write log to Elasticsearch: %w", err)
	}
	return len(p), nil
}

func (es *elasticSyncer) Sync() error {
	// No-op, Elasticsearch handles sync internally
	return nil
}
