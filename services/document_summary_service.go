package services

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"teletubpax-api/aws"
	"teletubpax-api/config"
	"teletubpax-api/logger"
)

type DocumentSummaryItem struct {
	Order                    int    `json:"order"`
	Link                     string `json:"link"`
	Summary                  string `json:"summary"`
	DifferenceFromOldVersion string `json:"differenceFromOldVersion"`
}

type DocumentSummaryService interface {
	AnalyzeDocuments(ctx context.Context, documentUrls []string) ([]DocumentSummaryItem, error)
}

type BedrockDocumentSummaryService struct {
	openSearchClient aws.OpenSearchClient
	kbClient         aws.KnowledgeBaseClient
	config           *config.Config
}

func NewBedrockDocumentSummaryService(
	openSearchClient aws.OpenSearchClient,
	kbClient aws.KnowledgeBaseClient,
	cfg *config.Config,
) *BedrockDocumentSummaryService {
	return &BedrockDocumentSummaryService{
		openSearchClient: openSearchClient,
		kbClient:         kbClient,
		config:           cfg,
	}
}

type documentInfo struct {
	url          string
	topic        string
	version      int
	yearMonth    string
	sortKey      string
	order        int
	summary      string
	difference   string
	content      string
	lastModified time.Time
}

func (s *BedrockDocumentSummaryService) AnalyzeDocuments(ctx context.Context, documentUrls []string) ([]DocumentSummaryItem, error) {
	log := logger.WithContext(ctx)
	log.Info("Starting document analysis", map[string]interface{}{
		"document_count": len(documentUrls),
	})
	startTime := time.Now()

	// Step 1: Parse and extract metadata from URLs
	documents := make([]documentInfo, 0, len(documentUrls))
	for _, url := range documentUrls {
		doc := documentInfo{
			url:       url,
			topic:     s.extractTopicFromUrl(url),
			version:   s.extractVersionNumber(url),
			yearMonth: s.extractYearMonthFromUrl(url),
		}
		doc.sortKey = s.createSortKey(doc.yearMonth, doc.version)
		documents = append(documents, doc)
	}

	log.Info("Extracted metadata from URLs", map[string]interface{}{
		"document_count": len(documents),
	})

	// Step 2: Sort documents by date (newest first), then by version (highest first)
	sort.Slice(documents, func(i, j int) bool {
		// Primary: Sort by year/month (newest first)
		if documents[i].yearMonth != documents[j].yearMonth {
			return documents[i].yearMonth > documents[j].yearMonth
		}
		// Secondary: Sort by version (highest first)
		return documents[i].version > documents[j].version
	})

	// Step 3: Assign order numbers
	for i := range documents {
		documents[i].order = i + 1
	}

	// Step 4: For now, skip content retrieval to avoid the loop issue
	// Content retrieval will be added in a future optimization
	// The summaries will be generated based on topic names only
	log.Info("Skipping content retrieval (optimization needed)", map[string]interface{}{})

	// Step 5: Generate summaries based on topic and metadata
	log.Info("Generating summaries based on metadata", map[string]interface{}{})
	for i := range documents {
		// Generate summary from topic name and metadata
		documents[i].summary = s.generateSummaryFromMetadata(documents[i].topic, documents[i].yearMonth, documents[i].version)

		// Find older version for comparison
		olderDoc := s.findOlderVersion(documents, documents[i].topic, documents[i].version, i)
		if olderDoc != nil {
			documents[i].difference = fmt.Sprintf("เวอร์ชัน %d (อัปเดตจากเวอร์ชัน %d)", documents[i].version, olderDoc.version)
		} else {
			if documents[i].version > 0 {
				documents[i].difference = fmt.Sprintf("เวอร์ชัน %d (เวอร์ชันแรก)", documents[i].version)
			} else {
				documents[i].difference = "เอกสารฉบับเดียว"
			}
		}
	}

	// Step 6: Convert to response format
	result := make([]DocumentSummaryItem, 0, len(documents))
	for _, doc := range documents {
		result = append(result, DocumentSummaryItem{
			Order:                    doc.order,
			Link:                     doc.url,
			Summary:                  doc.summary,
			DifferenceFromOldVersion: doc.difference,
		})
	}

	duration := time.Since(startTime)
	log.Info("Document analysis completed", map[string]interface{}{
		"duration_ms":    duration.Milliseconds(),
		"document_count": len(result),
	})

	return result, nil
}

// retrieveDocumentContent retrieves the content of a document from the Knowledge Base
func (s *BedrockDocumentSummaryService) retrieveDocumentContent(ctx context.Context, documentUrl string) (string, error) {
	log := logger.WithContext(ctx)

	// Extract topic to use as search query
	topic := s.extractTopicFromUrl(documentUrl)

	log.Info("Retrieving document content", map[string]interface{}{
		"url":   documentUrl,
		"topic": topic,
	})

	// Use OpenSearch client to retrieve document content directly
	// This is more efficient than querying all knowledge bases
	docs, err := s.openSearchClient.GetLastUpdateDocuments(ctx)
	if err != nil {
		log.Warn("Failed to retrieve documents from OpenSearch", map[string]interface{}{
			"error": err.Error(),
		})
		return "", err
	}

	// Find the matching document by URL
	for _, doc := range docs {
		if link, ok := doc["link"].(string); ok && link == documentUrl {
			if content, ok := doc["content"].(string); ok {
				log.Info("Found document content", map[string]interface{}{
					"url":            documentUrl,
					"content_length": len(content),
				})
				return content, nil
			}
		}
	}

	log.Warn("Document not found in OpenSearch results", map[string]interface{}{
		"url": documentUrl,
	})

	// Fallback: return empty content
	return "", fmt.Errorf("document not found: %s", documentUrl)
}

// generateSummaryFromMetadata generates a summary based on document metadata
func (s *BedrockDocumentSummaryService) generateSummaryFromMetadata(topic string, yearMonth string, version int) string {
	// Clean up topic name for better readability
	cleanTopic := strings.ReplaceAll(topic, "-_-", " - ")
	cleanTopic = strings.ReplaceAll(cleanTopic, "_", " ")
	cleanTopic = strings.ReplaceAll(cleanTopic, "-", " ")

	// Format date
	var dateStr string
	if yearMonth != "0000/00" {
		parts := strings.Split(yearMonth, "/")
		if len(parts) == 2 {
			dateStr = fmt.Sprintf(" (อัปเดต: %s/%s)", parts[1], parts[0])
		}
	}

	// Format version
	var versionStr string
	if version > 0 {
		versionStr = fmt.Sprintf(" [เวอร์ชัน %d]", version)
	}

	return fmt.Sprintf("เอกสาร: %s%s%s", cleanTopic, versionStr, dateStr)
}

// findOlderVersion finds an older version of the same topic
func (s *BedrockDocumentSummaryService) findOlderVersion(documents []documentInfo, topic string, currentVersion int, currentIndex int) *documentInfo {
	for i := range documents {
		if i == currentIndex {
			continue
		}

		// Same topic but older version
		if documents[i].topic == topic && documents[i].version < currentVersion {
			return &documents[i]
		}
	}
	return nil
}

// extractYearMonthFromUrl extracts year/month from URL path
func (s *BedrockDocumentSummaryService) extractYearMonthFromUrl(url string) string {
	re := regexp.MustCompile(`/(\d{4})/(\d{2})/`)
	matches := re.FindStringSubmatch(url)

	if len(matches) >= 3 {
		return fmt.Sprintf("%s/%s", matches[1], matches[2])
	}

	return "0000/00"
}

// createSortKey creates a sortable key from year/month and version
func (s *BedrockDocumentSummaryService) createSortKey(yearMonth string, version int) string {
	ym := strings.ReplaceAll(yearMonth, "/", "")
	return fmt.Sprintf("%s-%03d", ym, version)
}

// extractVersionNumber extracts version number from filename
func (s *BedrockDocumentSummaryService) extractVersionNumber(url string) int {
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return 0
	}
	filename := parts[len(parts)-1]

	// Remove file extension
	filename = strings.TrimSuffix(filename, ".pdf")
	filename = strings.TrimSuffix(filename, ".PDF")
	filename = strings.TrimSuffix(filename, ".doc")
	filename = strings.TrimSuffix(filename, ".docx")
	filename = strings.TrimSuffix(filename, ".txt")

	// Pattern to match version number at the end
	re := regexp.MustCompile(`-(\d+)$`)
	matches := re.FindStringSubmatch(filename)

	if len(matches) >= 2 {
		var version int
		fmt.Sscanf(matches[1], "%d", &version)
		return version
	}

	return 0
}

// extractTopicFromUrl extracts the topic/title from the filename
func (s *BedrockDocumentSummaryService) extractTopicFromUrl(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}
	filename := parts[len(parts)-1]

	// Remove file extension
	filename = strings.TrimSuffix(filename, ".pdf")
	filename = strings.TrimSuffix(filename, ".PDF")
	filename = strings.TrimSuffix(filename, ".doc")
	filename = strings.TrimSuffix(filename, ".docx")
	filename = strings.TrimSuffix(filename, ".txt")

	// Remove version number suffix
	re := regexp.MustCompile(`-(\d+)$`)
	filename = re.ReplaceAllString(filename, "")

	return filename
}

// convertPublicUrlToS3Uri converts public URL to S3 URI
func (s *BedrockDocumentSummaryService) convertPublicUrlToS3Uri(publicUrl string) string {
	// Example: https://bucket.s3.region.amazonaws.com/path/to/file.pdf
	// Convert to: s3://bucket/path/to/file.pdf

	re := regexp.MustCompile(`https://([^.]+)\.s3\.[^.]+\.amazonaws\.com/(.+)`)
	matches := re.FindStringSubmatch(publicUrl)

	if len(matches) >= 3 {
		bucket := matches[1]
		key := matches[2]
		return fmt.Sprintf("s3://%s/%s", bucket, key)
	}

	return publicUrl
}
