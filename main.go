package main

import (
	"context"
	"log"
	"net/http"
	"os"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"

	"teletubpax-api/aws"
	"teletubpax-api/config"
	"teletubpax-api/logger"
	"teletubpax-api/routing"
	"teletubpax-api/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize AWS SDK config
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %v", err)
	}

	// Initialize CloudWatch Logger for local/container development
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local"
	}
	logGroupName := "/teletubpax-api/local"
	logStreamName := hostname

	cwLogger, err := logger.NewCloudWatchLogger(awsCfg, logGroupName, logStreamName)
	if err != nil {
		log.Printf("Failed to initialize CloudWatch logger, using standard logger: %v", err)
		logger.Initialize(&logger.StandardLogger{})
	} else {
		logger.Initialize(cwLogger)
	}

	// Set log level to ERROR for container/production
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "ERROR" // Default to ERROR
	}
	
	switch logLevel {
	case "DEBUG":
		logger.SetLogLevel(logger.DEBUG)
	case "INFO":
		logger.SetLogLevel(logger.INFO)
	case "WARN":
		logger.SetLogLevel(logger.WARN)
	case "ERROR":
		logger.SetLogLevel(logger.ERROR)
	default:
		logger.SetLogLevel(logger.ERROR)
	}

	log.Printf("Logger initialized with level: %s", logLevel)
	log.Printf("Configuration loaded: Region=%s, Model=%s, KB=%s", cfg.AWSRegion, cfg.EmbeddingModelId, cfg.KnowledgeBaseId)

	// Create AWS clients
	embeddingClient := aws.NewBedrockEmbeddingClient(awsCfg, cfg.EmbeddingModelId)
	kbClient := aws.NewBedrockKBClient(awsCfg, cfg.KnowledgeBaseId, cfg.GenerativeModelId, cfg.AWSRegion, cfg.SystemInstructions)
	log.Println("AWS Bedrock clients initialized")

	// Create service
	questionSearchService := services.NewBedrockQuestionSearchService(
		embeddingClient,
		kbClient,
		cfg,
	)
	log.Println("Question search service created")

	// Setup routes with service
	router := routing.SetupRoutes(questionSearchService, cfg.MaxQuestionLength)
	
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		logger.Error("Server failed", map[string]interface{}{"error": err.Error()})
		log.Fatal(err)
	}
}
