package main

import (
	"context"
	"log"
	"net/http"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"

	"teletubpax-api/aws"
	"teletubpax-api/config"
	"teletubpax-api/routing"
	"teletubpax-api/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Configuration loaded successfully: Region=%s, Model=%s, KB=%s", 
		cfg.AWSRegion, cfg.EmbeddingModelId, cfg.KnowledgeBaseId)

	// Initialize AWS SDK config
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %v", err)
	}
	log.Println("AWS SDK configured successfully")

	// Create AWS clients
	embeddingClient := aws.NewBedrockEmbeddingClient(awsCfg, cfg.EmbeddingModelId)
	kbClient := aws.NewBedrockKBClient(awsCfg, cfg.KnowledgeBaseId, cfg.GenerativeModelId, cfg.AWSRegion)
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
		log.Fatal(err)
	}
}
