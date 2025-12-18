package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

type Logger interface {
	Debug(message string, fields ...map[string]interface{})
	Info(message string, fields ...map[string]interface{})
	Warn(message string, fields ...map[string]interface{})
	Error(message string, fields ...map[string]interface{})
	WithContext(ctx context.Context) Logger
}

type CloudWatchLogger struct {
	client        *cloudwatchlogs.Client
	logGroupName  string
	logStreamName string
	sequenceToken *string
	ctx           context.Context
	isLambda      bool
}

func NewCloudWatchLogger(cfg aws.Config, logGroupName, logStreamName string) (*CloudWatchLogger, error) {
	client := cloudwatchlogs.NewFromConfig(cfg)
	
	// Check if running in Lambda (Lambda handles log streams automatically)
	isLambda := os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != ""
	
	logger := &CloudWatchLogger{
		client:        client,
		logGroupName:  logGroupName,
		logStreamName: logStreamName,
		ctx:           context.Background(),
		isLambda:      isLambda,
	}

	// Only create log stream if not in Lambda
	if !isLambda {
		if err := logger.ensureLogStream(); err != nil {
			return nil, fmt.Errorf("failed to ensure log stream: %w", err)
		}
	}

	return logger, nil
}

func (l *CloudWatchLogger) ensureLogStream() error {
	// Create log group if it doesn't exist
	_, err := l.client.CreateLogGroup(l.ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(l.logGroupName),
	})
	if err != nil {
		// Ignore if already exists
		if _, ok := err.(*types.ResourceAlreadyExistsException); !ok {
			return err
		}
	}

	// Create log stream if it doesn't exist
	_, err = l.client.CreateLogStream(l.ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(l.logGroupName),
		LogStreamName: aws.String(l.logStreamName),
	})
	if err != nil {
		// Ignore if already exists
		if _, ok := err.(*types.ResourceAlreadyExistsException); !ok {
			return err
		}
	}

	return nil
}

func (l *CloudWatchLogger) WithContext(ctx context.Context) Logger {
	return &CloudWatchLogger{
		client:        l.client,
		logGroupName:  l.logGroupName,
		logStreamName: l.logStreamName,
		sequenceToken: l.sequenceToken,
		ctx:           ctx,
		isLambda:      l.isLambda,
	}
}

func (l *CloudWatchLogger) Debug(message string, fields ...map[string]interface{}) {
	if !shouldLog(DEBUG) {
		return
	}
	l.log(DEBUG, message, fields...)
}

func (l *CloudWatchLogger) Info(message string, fields ...map[string]interface{}) {
	if !shouldLog(INFO) {
		return
	}
	l.log(INFO, message, fields...)
}

func (l *CloudWatchLogger) Warn(message string, fields ...map[string]interface{}) {
	if !shouldLog(WARN) {
		return
	}
	l.log(WARN, message, fields...)
}

func (l *CloudWatchLogger) Error(message string, fields ...map[string]interface{}) {
	if !shouldLog(ERROR) {
		return
	}
	l.log(ERROR, message, fields...)
}

func (l *CloudWatchLogger) log(level LogLevel, message string, fields ...map[string]interface{}) {
	timestamp := time.Now().UnixMilli()
	logMessage := l.formatMessage(level, message, fields...)

	// Always log to stdout (for Lambda and local development)
	log.Printf("[%s] %s", level, logMessage)

	// If running in Lambda, CloudWatch Logs are handled automatically
	if l.isLambda {
		return
	}

	// For non-Lambda environments, send to CloudWatch
	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(l.logGroupName),
		LogStreamName: aws.String(l.logStreamName),
		LogEvents: []types.InputLogEvent{
			{
				Message:   aws.String(logMessage),
				Timestamp: aws.Int64(timestamp),
			},
		},
	}

	if l.sequenceToken != nil {
		input.SequenceToken = l.sequenceToken
	}

	output, err := l.client.PutLogEvents(l.ctx, input)
	if err != nil {
		log.Printf("Failed to send log to CloudWatch: %v", err)
		return
	}

	l.sequenceToken = output.NextSequenceToken
}

func (l *CloudWatchLogger) formatMessage(level LogLevel, message string, fields ...map[string]interface{}) string {
	if len(fields) == 0 {
		return message
	}

	formatted := message
	for _, fieldMap := range fields {
		for key, value := range fieldMap {
			formatted += fmt.Sprintf(" | %s=%v", key, value)
		}
	}
	return formatted
}
