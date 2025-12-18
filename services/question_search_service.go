package services

import (
	"context"
	"time"

	"teletubpax-api/aws"
	"teletubpax-api/config"
	"teletubpax-api/logger"
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
	log := logger.WithContext(ctx)
	log.Info("Question search request received", map[string]interface{}{
		"question_length": len(question),
		"question":        question,
	})
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
			log.Error("Knowledge base query failed", map[string]interface{}{
				"error": err.Error(),
			})
			return err
		}
		answer = ans
		return nil
	})

	if err != nil {
		duration := time.Since(startTime)
		log.Error("Question search failed after retries", map[string]interface{}{
			"error":        err.Error(),
			"duration_ms":  duration.Milliseconds(),
			"retry_count":  s.config.RetryAttempts,
		})
		return "", err
	}

	// Log successful response
	duration := time.Since(startTime)
	log.Info("Question search completed successfully", map[string]interface{}{
		"duration_ms":   duration.Milliseconds(),
		"answer_length": len(answer),
	})

	return answer, nil
}
