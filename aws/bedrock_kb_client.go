package aws

import (
	"context"
	"fmt"
	"strings"
	"teletubpax-api/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
)

type KnowledgeBaseClient interface {
	QueryKnowledgeBase(ctx context.Context, question string, enableRelateDocument bool) (string, []string, error)
}

type BedrockKBClient struct {
	client             *bedrockagentruntime.Client
	knowledgeBaseId    string
	generativeModelId  string
	region             string
	systemInstructions string
}

func NewBedrockKBClient(cfg aws.Config, knowledgeBaseId string, generativeModelId string, region string, systemInstructions string) *BedrockKBClient {
	return &BedrockKBClient{
		client:             bedrockagentruntime.NewFromConfig(cfg),
		knowledgeBaseId:    knowledgeBaseId,
		generativeModelId:  generativeModelId,
		region:             region,
		systemInstructions: systemInstructions,
	}
}

func (c *BedrockKBClient) QueryKnowledgeBase(ctx context.Context, question string, enableRelateDocument bool) (string, []string, error) {
	modelArn := fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/%s", c.region, c.generativeModelId)

	kbConfig := &types.KnowledgeBaseRetrieveAndGenerateConfiguration{
		KnowledgeBaseId: aws.String(c.knowledgeBaseId),
		ModelArn:        aws.String(modelArn),
	}

	// Add system instructions if provided
	if c.systemInstructions != "" {
		kbConfig.GenerationConfiguration = &types.GenerationConfiguration{
			PromptTemplate: &types.PromptTemplate{
				TextPromptTemplate: aws.String(c.systemInstructions + "\n\nQuestion: $query$\n\nContext: $search_results$"),
			},
		}
	}

	input := &bedrockagentruntime.RetrieveAndGenerateInput{
		Input: &types.RetrieveAndGenerateInput{
			Text: aws.String(question),
		},
		RetrieveAndGenerateConfiguration: &types.RetrieveAndGenerateConfiguration{
			Type:                       types.RetrieveAndGenerateTypeKnowledgeBase,
			KnowledgeBaseConfiguration: kbConfig,
		},
	}

	output, err := c.client.RetrieveAndGenerate(ctx, input)
	if err != nil {
		return "", nil, c.handleAWSError(err)
	}

	var relatedDocuments []string
	if enableRelateDocument && output.Citations != nil && len(output.Citations) > 0 {
		for _, citation := range output.Citations {
			if citation.RetrievedReferences != nil {
				for _, ref := range citation.RetrievedReferences {
					if ref.Location != nil && ref.Location.S3Location != nil {
						if ref.Location.S3Location.Uri != nil {
							s3Uri := *ref.Location.S3Location.Uri
							publicUrl := c.convertS3UriToPublicUrl(s3Uri)
							relatedDocuments = append(relatedDocuments, publicUrl)
						}
					}
				}
			}
		}
	}

	if output.Output != nil && output.Output.Text != nil {
		return *output.Output.Text, relatedDocuments, nil
	}

	return "ไม่พบคำตอบที่เกี่ยวข้องกับคำถามของคุณ", relatedDocuments, nil
}



func (c *BedrockKBClient) convertS3UriToPublicUrl(s3Uri string) string {
	s3Uri = strings.TrimPrefix(s3Uri, "s3://")
	parts := strings.SplitN(s3Uri, "/", 2)
	if len(parts) != 2 {
		return s3Uri
	}
	bucket := parts[0]
	key := parts[1]
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, c.region, key)
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
