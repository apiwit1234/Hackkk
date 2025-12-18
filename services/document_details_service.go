package services

import (
	"context"
	"time"

	"teletubpax-api/aws"
	"teletubpax-api/config"
	"teletubpax-api/logger"
)

type DocumentDetailsService interface {
	GetLastUpdateDocuments(ctx context.Context) ([]map[string]interface{}, error)
}

type OpenSearchDocumentService struct {
	openSearchClient aws.OpenSearchClient
	config           *config.Config
}

func NewOpenSearchDocumentService(
	openSearchClient aws.OpenSearchClient,
	cfg *config.Config,
) *OpenSearchDocumentService {
	return &OpenSearchDocumentService{
		openSearchClient: openSearchClient,
		config:           cfg,
	}
}

func (s *OpenSearchDocumentService) GetLastUpdateDocuments(ctx context.Context) ([]map[string]interface{}, error) {
	log := logger.WithContext(ctx)
	log.Info("Fetching last updated documents from OpenSearch", map[string]interface{}{})
	startTime := time.Now()

	// Query OpenSearch for documents
	documents, err := s.openSearchClient.GetLastUpdateDocuments(ctx)
	if err != nil {
		duration := time.Since(startTime)
		log.Error("Failed to fetch documents from OpenSearch", map[string]interface{}{
			"error":       err.Error(),
			"duration_ms": duration.Milliseconds(),
		})
		return nil, err
	}

	// For each document, check if there's an older version and compare
	for i, doc := range documents {
		topic, _ := doc["topic"].(string)
		currentVersion, _ := doc["version"].(int)

		// Find older version with same topic
		olderDoc := s.findOlderVersion(documents, topic, currentVersion, i)

		if olderDoc != nil {
			// Compare versions using Bedrock
			newerContent, _ := doc["content"].(string)
			olderContent, _ := olderDoc["content"].(string)

			if newerContent != "" && olderContent != "" {
				changeSummary, err := s.openSearchClient.CompareDocumentVersions(ctx, newerContent, olderContent, topic)
				if err != nil {
					log.Warn("Failed to compare document versions", map[string]interface{}{
						"topic": topic,
						"error": err.Error(),
					})
					documents[i]["changeSummary"] = "Unable to compare versions"
				} else {
					documents[i]["changeSummary"] = changeSummary
				}
			}
		}

		// Remove content field from final response (not needed in API response)
		delete(documents[i], "content")
	}

	duration := time.Since(startTime)
	log.Info("Documents retrieved successfully", map[string]interface{}{
		"duration_ms":    duration.Milliseconds(),
		"document_count": len(documents),
	})

	return documents, nil
}

// findOlderVersion finds an older version of the same topic
func (s *OpenSearchDocumentService) findOlderVersion(documents []map[string]interface{}, topic string, currentVersion int, currentIndex int) map[string]interface{} {
	for i, doc := range documents {
		if i == currentIndex {
			continue // Skip the current document
		}

		docTopic, _ := doc["topic"].(string)
		docVersion, _ := doc["version"].(int)

		// Same topic but older version
		if docTopic == topic && docVersion < currentVersion {
			return doc
		}
	}
	return nil
}
