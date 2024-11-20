package middleware

import (
	"context"
	"fmt"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitializeLogger sets up a Zap logger with Elasticsearch integration.
func InitializeLogger(serviceName, elasticURL, indexName string) (*zap.Logger, error) {
	// Create an Elasticsearch client
	client, err := elastic.NewClient(elastic.SetURL(elasticURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

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
	_, err := es.client.Index().
		Index(es.index).
		BodyString(string(p)).
		Do(context.Background())
	return len(p), err
}

func (es *elasticSyncer) Sync() error {
	// No-op, Elasticsearch handles sync internally
	return nil
}
