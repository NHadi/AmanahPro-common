package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger structure
type Logger struct {
	zapLogger *zap.Logger
	service   string
}

// elasticCore structure for writing logs to Elasticsearch
type elasticCore struct {
	client *elastic.Client
	index  string
	level  zapcore.Level
}

// Write sends log entries to Elasticsearch
func (ec *elasticCore) Write(p []byte) (n int, err error) {
	// Parse log entry
	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err != nil {
		return 0, fmt.Errorf("failed to unmarshal log entry: %w", err)
	}

	// Write to Elasticsearch
	_, err = ec.client.Index().
		Index(ec.index).
		BodyJson(logEntry).
		Do(context.Background())
	if err != nil {
		return 0, fmt.Errorf("failed to send log to Elasticsearch: %w", err)
	}

	return len(p), nil
}

// Sync satisfies zapcore.WriteSyncer interface
func (ec *elasticCore) Sync() error {
	return nil
}

// InitializeLogger initializes a logger with Elasticsearch integration
func InitializeLogger(serviceName, elasticURL, indexName string, level zapcore.Level) (*Logger, error) {
	client, err := elastic.NewClient(elastic.SetURL(elasticURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %v", err)
	}

	// Create custom core
	core := &elasticCore{
		client: client,
		index:  indexName,
		level:  level,
	}

	// Encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.LevelKey = "level"
	encoderConfig.MessageKey = "message"
	encoderConfig.CallerKey = "caller"

	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(core),
		level,
	))

	return &Logger{
		zapLogger: logger,
		service:   serviceName,
	}, nil
}

// MaskSensitiveData masks sensitive fields (e.g., passwords, tokens)
func MaskSensitiveData(data string) string {
	sensitiveKeys := []string{"password", "token"}
	for _, key := range sensitiveKeys {
		re := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*".+?"`, key))
		data = re.ReplaceAllString(data, fmt.Sprintf(`"%s":"[REDACTED]"`, key))
	}
	return data
}

// Info logs informational messages
func (l *Logger) Info(ctx context.Context, message string, fields ...zapcore.Field) {
	l.zapLogger.Info(message, append(fields, l.addCommonFields(ctx)...)...)
}

// Error logs error messages
func (l *Logger) Error(ctx context.Context, message string, fields ...zapcore.Field) {
	l.zapLogger.Error(message, append(fields, l.addCommonFields(ctx)...)...)
}

// Add common fields to logs
func (l *Logger) addCommonFields(ctx context.Context) []zapcore.Field {
	fields := []zapcore.Field{
		zap.String("service", l.service),
	}

	if traceID, ok := ctx.Value("trace_id").(string); ok {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	return fields
}
