package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"teletubpax-api/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type EmbeddingClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
}

type BedrockEmbeddingClient struct {
	client  *bedrockruntime.Client
	modelId string
}

func NewBedrockEmbeddingClient(cfg aws.Config, modelId string) *BedrockEmbeddingClient {
	return &BedrockEmbeddingClient{
		client:  bedrockruntime.NewFromConfig(cfg),
		modelId: modelId,
	}
}

type titanEmbedRequest struct {
	InputText string `json:"inputText"`
}

type titanEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

func (c *BedrockEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	request := titanEmbedRequest{
		InputText: text,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, errors.NewEmbeddingError("failed to marshal embedding request", err)
	}

	input := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(c.modelId),
		Body:        requestBody,
		ContentType: aws.String("application/json"),
	}

	output, err := c.client.InvokeModel(ctx, input)
	if err != nil {
		return nil, c.handleAWSError(err)
	}

	var response titanEmbedResponse
	if err := json.Unmarshal(output.Body, &response); err != nil {
		return nil, errors.NewEmbeddingError("failed to parse embedding response", err)
	}

	if len(response.Embedding) == 0 {
		return nil, errors.NewEmbeddingError("empty embedding vector returned", nil)
	}

	return response.Embedding, nil
}

func (c *BedrockEmbeddingClient) handleAWSError(err error) error {
	errMsg := err.Error()
	
	if contains(errMsg, "ValidationException") || contains(errMsg, "invalid") {
		return errors.NewValidationError(fmt.Sprintf("invalid input for embedding: %v", err))
	}
	
	if contains(errMsg, "ThrottlingException") || contains(errMsg, "TooManyRequestsException") {
		return errors.NewThrottlingError("embedding service throttled", err)
	}
	
	if contains(errMsg, "AccessDeniedException") || contains(errMsg, "UnauthorizedException") {
		return errors.NewAWSServiceError("invalid or missing AWS credentials", err)
	}
	
	if contains(errMsg, "ServiceUnavailableException") || contains(errMsg, "InternalServerException") {
		return errors.NewAWSServiceError("embedding service unavailable", err)
	}
	
	return errors.NewEmbeddingError("embedding generation failed", err)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
