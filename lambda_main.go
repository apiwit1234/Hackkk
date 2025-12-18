// +build lambda

// Lambda entry point for AWS deployment
// This file is used when building for Lambda (go build -tags lambda)
// For local development, use main.go instead

package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"

	"teletubpax-api/aws"
	"teletubpax-api/config"
	"teletubpax-api/routing"
	"teletubpax-api/services"
)

var httpLambda *httpadapter.HandlerAdapterV2

func init() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Configuration loaded: Region=%s, Model=%s, KB=%s",
		cfg.AWSRegion, cfg.EmbeddingModelId, cfg.KnowledgeBaseId)

	// Initialize AWS SDK config
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %v", err)
	}

	// Create AWS clients
	embeddingClient := aws.NewBedrockEmbeddingClient(awsCfg, cfg.EmbeddingModelId)
	kbClient := aws.NewBedrockKBClient(awsCfg, cfg.KnowledgeBaseId)

	// Create service
	questionSearchService := services.NewBedrockQuestionSearchService(
		embeddingClient,
		kbClient,
		cfg,
	)

	// Setup routes
	router := routing.SetupRoutes(questionSearchService, cfg.MaxQuestionLength)

	// Create Lambda adapter for API Gateway V2 (HTTP API)
	httpLambda = httpadapter.NewV2(router)
}

func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return httpLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
