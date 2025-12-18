package services

import (
	"context"
	"log"
	"time"

	"teletubpax-api/aws"
	"teletubpax-api/config"
	"teletubpax-api/utils"
)

type QuestionSearchService interface {
	SearchAnswer(ctx context.Context, question string) (string, error)
}

type BedrockQuestionSearchService struct {
	embeddingClient     aws.EmbeddingClient
	knowledgeBaseClient aws.KnowledgeBaseClient
	config              *config.Config
}

func NewBedrockQuestionSearchService(
	embeddingClient aws.EmbeddingClient,
	knowledgeBaseClient aws.KnowledgeBaseClient,
	cfg *config.Config,
) *BedrockQuestionSearchService {
	return &BedrockQuestionSearchService{
		embeddingClient:     embeddingClient,
		knowledgeBaseClient: knowledgeBaseClient,
		config:              cfg,
	}
}

func (s *BedrockQuestionSearchService) SearchAnswer(ctx context.Context, question string) (string, error) {
	// Log incoming request for audit
	log.Printf("[AUDIT] Question search request: %s", question)
	startTime := time.Now()

	// Query knowledge base with retry logic
	var answer string
	retryConfig := utils.RetryConfig{
		MaxAttempts:       s.config.RetryAttempts,
		InitialBackoff:    100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxBackoff:        2 * time.Second,
	}

	err := utils.RetryWithBackoff(ctx, retryConfig, func() error {
		// Query knowledge base directly with question text
		ans, err := s.knowledgeBaseClient.QueryKnowledgeBase(ctx, question)
		if err != nil {
			log.Printf("[ERROR] Knowledge base query failed: %v", err)
			return err
		}
		answer = ans
		return nil
	})

	if err != nil {
		log.Printf("[ERROR] Question search failed after retries: %v", err)
		return "", err
	}

	// Log successful response
	duration := time.Since(startTime)
	log.Printf("[INFO] Question search completed successfully in %v", duration)

	return answer, nil
}
