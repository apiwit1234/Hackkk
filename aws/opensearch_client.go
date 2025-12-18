package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"teletubpax-api/errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
)

type OpenSearchClient interface {
	GetLastUpdateDocuments(ctx context.Context) ([]map[string]interface{}, error)
	CompareDocumentVersions(ctx context.Context, newerContent, olderContent, topic string) (string, error)
}

type BedrockOpenSearchClient struct {
	client            *bedrockagentruntime.Client
	knowledgeBaseId   string
	region            string
	kbClient          KnowledgeBaseClient
	generativeModelId string
}

func NewBedrockOpenSearchClient(cfg aws.Config, knowledgeBaseId string, region string, kbClient KnowledgeBaseClient, generativeModelId string) *BedrockOpenSearchClient {
	return &BedrockOpenSearchClient{
		client:            bedrockagentruntime.NewFromConfig(cfg),
		knowledgeBaseId:   knowledgeBaseId,
		region:            region,
		kbClient:          kbClient,
		generativeModelId: generativeModelId,
	}
}

func (c *BedrockOpenSearchClient) GetLastUpdateDocuments(ctx context.Context) ([]map[string]interface{}, error) {
	// Use Bedrock Agent Runtime Retrieve API to get documents from the knowledge base
	// This retrieves documents from the underlying OpenSearch index
	input := &bedrockagentruntime.RetrieveInput{
		KnowledgeBaseId: aws.String(c.knowledgeBaseId),
		RetrievalQuery: &types.KnowledgeBaseQuery{
			Text: aws.String("*"), // Query all documents
		},
		RetrievalConfiguration: &types.KnowledgeBaseRetrievalConfiguration{
			VectorSearchConfiguration: &types.KnowledgeBaseVectorSearchConfiguration{
				NumberOfResults: aws.Int32(100), // Adjust as needed
			},
		},
	}

	output, err := c.client.Retrieve(ctx, input)
	if err != nil {
		return nil, c.handleAWSError(err)
	}

	// Parse the response and extract document details
	var documents []map[string]interface{}

	if output.RetrievalResults != nil {
		for _, result := range output.RetrievalResults {
			doc := make(map[string]interface{})

			// Extract content
			if result.Content != nil && result.Content.Text != nil {
				doc["content"] = *result.Content.Text
			}

			// Extract score
			if result.Score != nil {
				doc["score"] = *result.Score
			}

			var publicUrl string

			// Extract location information
			if result.Location != nil {
				location := make(map[string]interface{})

				if result.Location.S3Location != nil {
					s3Location := make(map[string]interface{})
					if result.Location.S3Location.Uri != nil {
						s3Uri := *result.Location.S3Location.Uri
						publicUrl = c.convertS3UriToPublicUrl(s3Uri)
						s3Location["uri"] = s3Uri
						s3Location["publicUrl"] = publicUrl
					}
					location["s3Location"] = s3Location
				}

				if result.Location.Type != "" {
					location["type"] = string(result.Location.Type)
				}

				doc["location"] = location
			}

			// Extract and parse date from URL path (e.g., content/2025/05/)
			yearMonth := c.extractYearMonthFromUrl(publicUrl)
			doc["yearMonth"] = yearMonth
			doc["sortKey"] = c.createSortKey(yearMonth)

			// Extract version number from filename (e.g., -1, -2)
			versionNumber := c.extractVersionNumber(publicUrl)
			doc["version"] = versionNumber

			// Extract last modified date from metadata
			var lastModified time.Time
			if result.Metadata != nil {
				metadata := make(map[string]interface{})
				for key, value := range result.Metadata {
					// Convert document.Interface to string representation
					if valueBytes, err := json.Marshal(value); err == nil {
						var jsonValue interface{}
						if err := json.Unmarshal(valueBytes, &jsonValue); err == nil {
							metadata[key] = jsonValue

							// Try to extract last modified date
							if strings.Contains(strings.ToLower(key), "modified") ||
								strings.Contains(strings.ToLower(key), "updated") ||
								key == "lastModified" || key == "last_modified" {
								if dateStr, ok := jsonValue.(string); ok {
									if parsedTime, err := time.Parse(time.RFC3339, dateStr); err == nil {
										lastModified = parsedTime
									}
								}
							}
						} else {
							metadata[key] = string(valueBytes)
						}
					}
				}
				doc["metadata"] = metadata
			}

			doc["lastModified"] = lastModified
			doc["lastModifiedUnix"] = lastModified.Unix()

			documents = append(documents, doc)
		}
	}

	// Sort with multiple criteria:
	// 1. Year/Month (newest first)
	// 2. Last modified date (newest first)
	// 3. Version number (highest version first: -2, -1, no version)
	sort.Slice(documents, func(i, j int) bool {
		// Primary: Sort by year/month
		sortKeyI := documents[i]["sortKey"].(string)
		sortKeyJ := documents[j]["sortKey"].(string)

		if sortKeyI != sortKeyJ {
			return sortKeyI > sortKeyJ // Descending (newest first)
		}

		// Secondary: Sort by last modified date
		lastModI := documents[i]["lastModifiedUnix"].(int64)
		lastModJ := documents[j]["lastModifiedUnix"].(int64)

		if lastModI != lastModJ {
			return lastModI > lastModJ // Descending (newest first)
		}

		// Tertiary: Sort by version number
		versionI := documents[i]["version"].(int)
		versionJ := documents[j]["version"].(int)

		return versionI > versionJ // Descending (highest version first)
	})

	// Return only the last 10 newest documents
	if len(documents) > 10 {
		documents = documents[:10]
	}

	// Transform to simplified response format
	simplifiedDocs := make([]map[string]interface{}, 0, len(documents))
	for _, doc := range documents {
		simplified := make(map[string]interface{})

		// 1. lastModifyDate
		if lastMod, ok := doc["lastModified"].(time.Time); ok && !lastMod.IsZero() {
			simplified["lastModifyDate"] = lastMod.Format(time.RFC3339)
		} else {
			// Use yearMonth as fallback
			if yearMonth, ok := doc["yearMonth"].(string); ok {
				simplified["lastModifyDate"] = yearMonth
			} else {
				simplified["lastModifyDate"] = ""
			}
		}

		// 2. link (public URL)
		if location, ok := doc["location"].(map[string]interface{}); ok {
			if s3Location, ok := location["s3Location"].(map[string]interface{}); ok {
				if publicUrl, ok := s3Location["publicUrl"].(string); ok {
					simplified["link"] = publicUrl
				}
			}
		}

		// 3. topic (extracted from filename)
		var topic string
		var publicUrl string
		if location, ok := doc["location"].(map[string]interface{}); ok {
			if s3Location, ok := location["s3Location"].(map[string]interface{}); ok {
				if url, ok := s3Location["publicUrl"].(string); ok {
					topic = c.extractTopicFromUrl(url)
					publicUrl = url
					simplified["topic"] = topic
					simplified["link"] = publicUrl
				}
			}
		}

		// 4. version - current version number
		currentVersion := 0
		if version, ok := doc["version"].(int); ok {
			currentVersion = version
			simplified["version"] = currentVersion
		}

		// 5. changeSummary - compare with older version if exists
		simplified["changeSummary"] = ""

		simplifiedDocs = append(simplifiedDocs, simplified)
	}

	return simplifiedDocs, nil
}

// extractYearMonthFromUrl extracts year/month from URL path like "content/2025/05/"
func (c *BedrockOpenSearchClient) extractYearMonthFromUrl(url string) string {
	// Pattern to match year/month in the URL path (e.g., /2025/05/)
	re := regexp.MustCompile(`/(\d{4})/(\d{2})/`)
	matches := re.FindStringSubmatch(url)

	if len(matches) >= 3 {
		year := matches[1]
		month := matches[2]
		return fmt.Sprintf("%s/%s", year, month)
	}

	return "0000/00" // Default for documents without date in path
}

// createSortKey creates a sortable key from year/month string
func (c *BedrockOpenSearchClient) createSortKey(yearMonth string) string {
	// Convert "2025/05" to "202505" for easy sorting
	return strings.ReplaceAll(yearMonth, "/", "")
}

// extractVersionNumber extracts version number from filename
// Examples:
//   - "file-1-2.pdf" -> 2
//   - "file-1-1.pdf" -> 1
//   - "file-1.pdf" -> 0
//   - "Horaland1-2.pdf" -> 2
func (c *BedrockOpenSearchClient) extractVersionNumber(url string) int {
	// Extract filename from URL
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

	// Pattern to match version number at the end: -1, -2, etc.
	re := regexp.MustCompile(`-(\d+)$`)
	matches := re.FindStringSubmatch(filename)

	if len(matches) >= 2 {
		if version, err := strconv.Atoi(matches[1]); err == nil {
			return version
		}
	}

	return 0 // No version number found
}

// extractTopicFromUrl extracts the topic/title from the filename in URL
// Examples:
//   - "https://.../สื่อความสาขา-_-Horaland1-2.pdf" -> "สื่อความสาขา-_-Horaland1"
//   - "https://.../การขอลดค่างวด-waive.pdf" -> "การขอลดค่างวด-waive"
func (c *BedrockOpenSearchClient) extractTopicFromUrl(url string) string {
	// Extract filename from URL
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

	// Remove version number suffix if present (e.g., -1, -2)
	re := regexp.MustCompile(`-(\d+)$`)
	filename = re.ReplaceAllString(filename, "")

	return filename
}

func (c *BedrockOpenSearchClient) convertS3UriToPublicUrl(s3Uri string) string {
	s3Uri = strings.TrimPrefix(s3Uri, "s3://")
	parts := strings.SplitN(s3Uri, "/", 2)
	if len(parts) != 2 {
		return s3Uri
	}
	bucket := parts[0]
	key := parts[1]
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, c.region, key)
}

// CompareDocumentVersions uses Bedrock to compare two document versions and summarize changes
func (c *BedrockOpenSearchClient) CompareDocumentVersions(ctx context.Context, newerContent, olderContent, topic string) (string, error) {
	// Create a prompt for Bedrock to compare the documents
	prompt := fmt.Sprintf(`Compare these two versions of the document "%s" and provide a summary of what changed.

Older Version:
%s

Newer Version:
%s

Please provide a concise summary of the changes in JSON format with these fields:
{
  "version": "version number or identifier",
  "changeSummary": "brief description of what changed"
}

Focus on the main differences and keep the summary brief and clear.`, topic, olderContent, newerContent)

	// Use the KB client to query Bedrock
	answer, _, err := c.kbClient.QueryKnowledgeBase(ctx, prompt, false)
	if err != nil {
		return "", err
	}

	return answer, nil
}

func (c *BedrockOpenSearchClient) handleAWSError(err error) error {
	errMsg := err.Error()

	if contains(errMsg, "ValidationException") || contains(errMsg, "invalid") {
		return errors.NewValidationError(fmt.Sprintf("invalid OpenSearch query: %v", err))
	}

	if contains(errMsg, "ThrottlingException") || contains(errMsg, "TooManyRequestsException") {
		return errors.NewThrottlingError("OpenSearch service throttled", err)
	}

	if contains(errMsg, "AccessDeniedException") || contains(errMsg, "UnauthorizedException") {
		return errors.NewAWSServiceError("invalid or missing AWS credentials", err)
	}

	if contains(errMsg, "ResourceNotFoundException") {
		return errors.NewAWSServiceError("knowledge base not found", err)
	}

	if contains(errMsg, "ServiceUnavailableException") || contains(errMsg, "InternalServerException") {
		return errors.NewAWSServiceError("OpenSearch service unavailable", err)
	}

	if contains(errMsg, "TimeoutException") || contains(errMsg, "timeout") {
		return errors.NewAWSServiceError("OpenSearch query timeout", err)
	}

	return errors.NewAWSServiceError("OpenSearch query failed", err)
}
