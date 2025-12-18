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
	client            *bedrockagentruntime.Client
	knowledgeBaseId   string
	generativeModelId string
	region            string
}

func NewBedrockKBClient(cfg aws.Config, knowledgeBaseId string, generativeModelId string, region string) *BedrockKBClient {
	return &BedrockKBClient{
		client:            bedrockagentruntime.NewFromConfig(cfg),
		knowledgeBaseId:   knowledgeBaseId,
		generativeModelId: generativeModelId,
		region:            region,
	}
}

func (c *BedrockKBClient) QueryKnowledgeBase(ctx context.Context, question string) (string, error) {
	// Build model ARN
	modelArn := fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/%s", c.region, c.generativeModelId)

	// Use RetrieveAndGenerate to get AI-generated answer based on retrieved documents
	input := &bedrockagentruntime.RetrieveAndGenerateInput{
		Input: &types.RetrieveAndGenerateInput{
			Text: aws.String(question),
		},
		RetrieveAndGenerateConfiguration: &types.RetrieveAndGenerateConfiguration{
			Type: types.RetrieveAndGenerateTypeKnowledgeBase,
			KnowledgeBaseConfiguration: &types.KnowledgeBaseRetrieveAndGenerateConfiguration{
				KnowledgeBaseId: aws.String(c.knowledgeBaseId),
				ModelArn:        aws.String(modelArn),
			},
		},
	}

	output, err := c.client.RetrieveAndGenerate(ctx, input)
	if err != nil {
		return "", c.handleAWSError(err)
	}

	// Extract generated answer
	if output.Output != nil && output.Output.Text != nil {
		return *output.Output.Text, nil
	}

	return "ไม่พบคำตอบที่เกี่ยวข้องกับคำถามของคุณ", nil
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
