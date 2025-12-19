//go:build lambda
// +build lambda

// Lambda entry point for AWS deployment
// This file is used when building for Lambda (go build -tags lambda)
// For local development, use main.go instead

package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"teletubpax-api/aws"
	"teletubpax-api/config"
	"teletubpax-api/logger"
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

	// Initialize AWS SDK config
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %v", err)
	}

	// Initialize Standard Logger for Lambda (CloudWatch handles logs automatically)
	logger.Initialize(&logger.StandardLogger{})
	logger.SetLogLevel(logger.ERROR) // Only log errors in Lambda

	log.Printf("Lambda initialization started for function: %s", os.Getenv("AWS_LAMBDA_FUNCTION_NAME"))

	// Create AWS clients
	embeddingClient := aws.NewBedrockEmbeddingClient(awsCfg, cfg.EmbeddingModelId)
	kbClient := aws.NewBedrockKBClient(awsCfg, cfg.KnowledgeBaseIds, cfg.GenerativeModelId, cfg.AWSRegion, cfg.QuestionSearchInstructions)
	openSearchClient := aws.NewBedrockOpenSearchClient(awsCfg, cfg.KnowledgeBaseIds[0], cfg.AWSRegion, kbClient, cfg.GenerativeModelId, cfg.DocumentComparisonInstructions)

	// Create services
	questionSearchService := services.NewBedrockQuestionSearchService(
		embeddingClient,
		kbClient,
		cfg,
	)

	documentDetailsService := services.NewOpenSearchDocumentService(
		openSearchClient,
		cfg,
	)

	// Setup routes
	router := routing.SetupRoutes(questionSearchService, documentDetailsService, cfg.MaxQuestionLength)

	// Create Lambda adapter for API Gateway V2 (HTTP API)
	httpLambda = httpadapter.NewV2(router)

	log.Println("Lambda initialization completed successfully")
}

func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	// Get response from HTTP adapter
	resp, err := httpLambda.ProxyWithContext(ctx, req)

	// Ensure CORS headers are always present in Lambda response
	if resp.Headers == nil {
		resp.Headers = make(map[string]string)
	}

	// Add CORS headers to response (these will be merged with any existing headers)
	resp.Headers["Access-Control-Allow-Origin"] = "*"
	resp.Headers["Access-Control-Allow-Methods"] = "GET, POST, OPTIONS"
	resp.Headers["Access-Control-Allow-Headers"] = "Content-Type, Authorization"
	resp.Headers["Access-Control-Max-Age"] = "3600"
	resp.Headers["Content-Type"] = "application/json"

	return resp, err
}

func main() {
	lambda.Start(Handler)
}
