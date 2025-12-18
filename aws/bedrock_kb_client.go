package aws

import (
	"context"
	"fmt"
	"teletubpax-api/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
)

type KnowledgeBaseClient interface {
	QueryKnowledgeBase(ctx context.Context, question string) (string, error)
}

type BedrockKBClient struct {
	client          *bedrockagentruntime.Client
	knowledgeBaseId string
}

func NewBedrockKBClient(cfg aws.Config, knowledgeBaseId string) *BedrockKBClient {
	return &BedrockKBClient{
		client:          bedrockagentruntime.NewFromConfig(cfg),
		knowledgeBaseId: knowledgeBaseId,
	}
}

func (c *BedrockKBClient) QueryKnowledgeBase(ctx context.Context, question string) (string, error) {
	input := &bedrockagentruntime.RetrieveInput{
		KnowledgeBaseId: aws.String(c.knowledgeBaseId),
		RetrievalQuery: &types.KnowledgeBaseQuery{
			Text: aws.String(question),
		},
	}

	output, err := c.client.Retrieve(ctx, input)
	if err != nil {
		return "", c.handleAWSError(err)
	}

	if output.RetrievalResults == nil || len(output.RetrievalResults) == 0 {
		return "", nil
	}

	// Find the result with the highest confidence score
	bestResult := output.RetrievalResults[0]
	highestScore := getScore(bestResult)

	for _, result := range output.RetrievalResults[1:] {
		score := getScore(result)
		if score > highestScore {
			highestScore = score
			bestResult = result
		}
	}

	// Extract answer text from the best result
	if bestResult.Content != nil && bestResult.Content.Text != nil {
		return *bestResult.Content.Text, nil
	}

	return "", nil
}

func getScore(result types.KnowledgeBaseRetrievalResult) float64 {
	if result.Score != nil {
		return float64(*result.Score)
	}
	return 0.0
}

func (c *BedrockKBClient) handleAWSError(err error) error {
	errMsg := err.Error()

	if contains(errMsg, "ValidationException") || contains(errMsg, "invalid") {
		return errors.NewValidationError(fmt.Sprintf("invalid knowledge base query: %v", err))
	}

	if contains(errMsg, "ThrottlingException") || contains(errMsg, "TooManyRequestsException") {
		return errors.NewThrottlingError("knowledge base service throttled", err)
	}

	if contains(errMsg, "AccessDeniedException") || contains(errMsg, "UnauthorizedException") {
		return errors.NewAWSServiceError("invalid or missing AWS credentials", err)
	}

	if contains(errMsg, "ResourceNotFoundException") {
		return errors.NewKnowledgeBaseError("knowledge base not found", err)
	}

	if contains(errMsg, "ServiceUnavailableException") || contains(errMsg, "InternalServerException") {
		return errors.NewAWSServiceError("knowledge base service unavailable", err)
	}

	if contains(errMsg, "TimeoutException") || contains(errMsg, "timeout") {
		return errors.NewAWSServiceError("knowledge base query timeout", err)
	}

	return errors.NewKnowledgeBaseError("knowledge base query failed", err)
}
