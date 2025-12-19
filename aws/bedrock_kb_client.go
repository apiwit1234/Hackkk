package aws

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"teletubpax-api/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	rttypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

type KnowledgeBaseClient interface {
	QueryKnowledgeBase(ctx context.Context, question string, enableRelateDocument bool) (string, []string, error)
	QueryMultipleKnowledgeBases(ctx context.Context, question string, enableRelateDocument bool) (string, []string, error)
}

type BedrockKBClient struct {
	client             *bedrockagentruntime.Client
	runtimeClient      *bedrockruntime.Client
	knowledgeBaseIds   []string
	generativeModelId  string
	region             string
	systemInstructions string
}

func NewBedrockKBClient(cfg aws.Config, knowledgeBaseIds []string, generativeModelId string, region string, systemInstructions string) *BedrockKBClient {
	return &BedrockKBClient{
		client:             bedrockagentruntime.NewFromConfig(cfg),
		runtimeClient:      bedrockruntime.NewFromConfig(cfg),
		knowledgeBaseIds:   knowledgeBaseIds,
		generativeModelId:  generativeModelId,
		region:             region,
		systemInstructions: systemInstructions,
	}
}

func (c *BedrockKBClient) QueryKnowledgeBase(ctx context.Context, question string, enableRelateDocument bool) (string, []string, error) {
	// Use the first knowledge base for backward compatibility
	if len(c.knowledgeBaseIds) == 0 {
		return "", nil, fmt.Errorf("no knowledge base IDs configured")
	}
	return c.queryKnowledgeBaseById(ctx, c.knowledgeBaseIds[0], question, enableRelateDocument)
}

func (c *BedrockKBClient) queryKnowledgeBaseById(ctx context.Context, knowledgeBaseId string, question string, enableRelateDocument bool) (string, []string, error) {
	// Build the correct model identifier based on model type
	var modelArn string
	if strings.HasPrefix(c.generativeModelId, "arn:") {
		// Already an ARN, use as-is
		modelArn = c.generativeModelId
	} else if strings.Contains(c.generativeModelId, "anthropic.claude") && strings.Contains(c.generativeModelId, "haiku") {
		// For Claude Haiku models, use cross-region inference profile ID (not ARN)
		modelArn = "us.anthropic.claude-haiku-4-5-20251001-v1:0"
	} else {
		// Standard foundation model ARN
		modelArn = fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/%s", c.region, c.generativeModelId)
	}

	kbConfig := &types.KnowledgeBaseRetrieveAndGenerateConfiguration{
		KnowledgeBaseId: aws.String(knowledgeBaseId),
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

func (c *BedrockKBClient) QueryMultipleKnowledgeBases(ctx context.Context, question string, enableRelateDocument bool) (string, []string, error) {
	if len(c.knowledgeBaseIds) == 0 {
		return "", nil, fmt.Errorf("no knowledge base IDs configured")
	}

	type kbResult struct {
		answer    string
		documents []string
		err       error
		kbId      string
	}

	results := make(chan kbResult, len(c.knowledgeBaseIds))
	var wg sync.WaitGroup

	// Query all knowledge bases in parallel
	for _, kbId := range c.knowledgeBaseIds {
		wg.Add(1)
		go func(knowledgeBaseId string) {
			defer wg.Done()
			answer, docs, err := c.queryKnowledgeBaseById(ctx, knowledgeBaseId, question, enableRelateDocument)
			results <- kbResult{
				answer:    answer,
				documents: docs,
				err:       err,
				kbId:      knowledgeBaseId,
			}
		}(kbId)
	}

	// Wait for all queries to complete
	wg.Wait()
	close(results)

	// Collect and combine results
	var combinedAnswer strings.Builder
	var allDocuments []string
	documentSet := make(map[string]bool)
	successCount := 0
	var lastError error

	for result := range results {
		if result.err != nil {
			lastError = result.err
			continue
		}

		successCount++

		// Combine answers from different KBs
		if result.answer != "" && result.answer != "ไม่พบคำตอบที่เกี่ยวข้องกับคำถามของคุณ" {
			if combinedAnswer.Len() > 0 {
				combinedAnswer.WriteString("\n\n")
			}
			combinedAnswer.WriteString(result.answer)
		}

		// Deduplicate documents
		for _, doc := range result.documents {
			if !documentSet[doc] {
				documentSet[doc] = true
				allDocuments = append(allDocuments, doc)
			}
		}
	}

	// If all queries failed, return the last error
	if successCount == 0 {
		if lastError != nil {
			return "", nil, lastError
		}
		return "", nil, fmt.Errorf("all knowledge base queries failed")
	}

	// Return combined results
	finalAnswer := combinedAnswer.String()
	if finalAnswer == "" {
		finalAnswer = "ไม่พบคำตอบที่เกี่ยวข้องกับคำถามของคุณ"
		return finalAnswer, allDocuments, nil
	}

	// Synthesize multiple answers into one coherent response
	fmt.Printf("DEBUG: Starting synthesis for question: %s\n", question)
	fmt.Printf("DEBUG: Combined answers length: %d characters\n", len(finalAnswer))

	synthesizedAnswer, err := c.synthesizeAnswers(ctx, question, finalAnswer)
	if err != nil {
		// If synthesis fails, log the error and return the combined answer as fallback
		fmt.Printf("ERROR: Synthesis failed: %v. Returning combined answers.\n", err)
		return finalAnswer, allDocuments, nil
	}

	fmt.Printf("DEBUG: Synthesis successful. Result length: %d characters\n", len(synthesizedAnswer))
	return synthesizedAnswer, allDocuments, nil
}

func (c *BedrockKBClient) synthesizeAnswers(ctx context.Context, question string, combinedAnswers string) (string, error) {
	fmt.Printf("DEBUG: synthesizeAnswers called with modelId: %s\n", c.generativeModelId)

	// Create synthesis prompt
	userMessage := fmt.Sprintf(`You have received multiple answers from different knowledge bases for the same question. Synthesize them into ONE clear, coherent answer.

Original Question: %s

Multiple Answers:
%s

Instructions:
1. Remove "Sorry, I am unable to assist" messages unless ALL answers contain them
2. Prefer information from the most recent documents (higher version numbers, later dates)
3. Remove duplicate information
4. Combine complementary details into a single coherent response
5. If answers contradict, choose the most recent/authoritative one
6. Maintain the same language as the original question
7. Be concise and direct
8. No Fluff: Do NOT use phrases like "Based on the document...", "The system found...", or "According to...". Start with the answer immediately.
	8.1 Check if the user's input ends with or contains specific question particles indicating a need for exact data:
  		**Keywords:** ไร, อะไร, ไหน, ที่ไหน, หรือไม่, ไหม, มั๊ย, เท่าไหร่, กี่บาท, ยัง (Yet), ใคร (Who).
		**Action:** Start with the answer immediately. No filler.
    	**Constraint:** Maximum 25 words.
    	**Example:** "ดอกเบี้ย 5% ต่อปี สำหรับลูกค้าใหม่"
	8.2 Provide ONLY the final synthesized answer:`, question, combinedAnswers)

	fmt.Printf("DEBUG: Calling Bedrock Converse API...\n")

	// Get the correct model identifier (inference profile for Claude Haiku)
	modelId := c.generativeModelId
	if strings.Contains(c.generativeModelId, "anthropic.claude") && strings.Contains(c.generativeModelId, "haiku") {
		// Use cross-region inference profile ID for Claude Haiku
		modelId = "us.anthropic.claude-haiku-4-5-20251001-v1:0"
	}

	fmt.Printf("DEBUG: Using model ID: %s\n", modelId)

	// Use Bedrock Runtime Converse API for direct model invocation
	converseInput := &bedrockruntime.ConverseInput{
		ModelId: aws.String(modelId),
		Messages: []rttypes.Message{
			{
				Role: rttypes.ConversationRoleUser,
				Content: []rttypes.ContentBlock{
					&rttypes.ContentBlockMemberText{
						Value: userMessage,
					},
				},
			},
		},
		InferenceConfig: &rttypes.InferenceConfiguration{
			MaxTokens:   aws.Int32(2048),
			Temperature: aws.Float32(0.3), // Lower temperature for more focused synthesis
		},
	}

	output, err := c.runtimeClient.Converse(ctx, converseInput)
	if err != nil {
		fmt.Printf("ERROR: Converse API call failed: %v\n", err)
		return "", fmt.Errorf("synthesis converse API failed: %w", err)
	}

	fmt.Printf("DEBUG: Converse API call successful, extracting response...\n")

	// Extract the response text
	if output.Output != nil {
		if msg, ok := output.Output.(*rttypes.ConverseOutputMemberMessage); ok {
			if len(msg.Value.Content) > 0 {
				if textBlock, ok := msg.Value.Content[0].(*rttypes.ContentBlockMemberText); ok {
					fmt.Printf("DEBUG: Successfully extracted synthesized text\n")
					return textBlock.Value, nil
				}
			}
		}
	}

	fmt.Printf("ERROR: Failed to extract text from Converse response\n")
	return "", fmt.Errorf("no synthesis output received")
}

func (c *BedrockKBClient) getModelArn() string {
	if strings.HasPrefix(c.generativeModelId, "arn:") {
		return c.generativeModelId
	} else if strings.Contains(c.generativeModelId, "anthropic.claude") && strings.Contains(c.generativeModelId, "haiku") {
		return "us.anthropic.claude-haiku-4-5-20251001-v1:0"
	}
	return fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/%s", c.region, c.generativeModelId)
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
		return errors.NewKnowledgeBaseError(fmt.Sprintf("resource not found: %v", err), err)
	}

	if contains(errMsg, "ServiceUnavailableException") || contains(errMsg, "InternalServerException") {
		return errors.NewAWSServiceError("knowledge base service unavailable", err)
	}

	if contains(errMsg, "TimeoutException") || contains(errMsg, "timeout") {
		return errors.NewAWSServiceError("knowledge base query timeout", err)
	}

	return errors.NewKnowledgeBaseError("knowledge base query failed", err)
}
