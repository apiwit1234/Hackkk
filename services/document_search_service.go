package services

import (
	"context"
	"time"

	"teletubpax-api/aws"
	"teletubpax-api/config"
	"teletubpax-api/logger"
	"teletubpax-api/utils"
)

type DocumentSearchService interface {
	SearchDocumentsByKeyword(ctx context.Context, keyword string) ([]string, error)
}

type BedrockDocumentSearchService struct {
	knowledgeBaseClient aws.KnowledgeBaseClient
	config              *config.Config
}

func NewBedrockDocumentSearchService(
	knowledgeBaseClient aws.KnowledgeBaseClient,
	cfg *config.Config,
) *BedrockDocumentSearchService {
	return &BedrockDocumentSearchService{
		knowledgeBaseClient: knowledgeBaseClient,
		config:              cfg,
	}
}

func (s *BedrockDocumentSearchService) SearchDocumentsByKeyword(ctx context.Context, keyword string) ([]string, error) {
	log := logger.WithContext(ctx)
	log.Info("Document search request received", map[string]interface{}{
		"keyword_length": len(keyword),
		"keyword":        keyword,
	})
	startTime := time.Now()

	var relatedDocuments []string
	retryConfig := utils.RetryConfig{
		MaxAttempts:       s.config.RetryAttempts,
		InitialBackoff:    100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxBackoff:        2 * time.Second,
	}

	err := utils.RetryWithBackoff(ctx, retryConfig, func() error {
		_, docs, err := s.knowledgeBaseClient.QueryKnowledgeBase(ctx, keyword, true)
		if err != nil {
			log.Error("Knowledge base query failed", map[string]interface{}{
				"error": err.Error(),
			})
			return err
		}
		relatedDocuments = docs
		return nil
	})

	if err != nil {
		duration := time.Since(startTime)
		log.Error("Document search failed after retries", map[string]interface{}{
			"error":       err.Error(),
			"duration_ms": duration.Milliseconds(),
			"retry_count": s.config.RetryAttempts,
		})
		return nil, err
	}

	duration := time.Since(startTime)
	log.Info("Document search completed successfully", map[string]interface{}{
		"duration_ms":    duration.Milliseconds(),
		"document_count": len(relatedDocuments),
	})

	return relatedDocuments, nil
}
