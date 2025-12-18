package config

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"
)

//go:embed system_instructions.txt
var systemInstructions string

type Config struct {
	AWSRegion          string
	EmbeddingModelId   string
	KnowledgeBaseId    string
	GenerativeModelId  string
	SystemInstructions string
	MaxQuestionLength  int
	RetryAttempts      int
	OpenSearchEndpoint string
	OpenSearchIndex    string
}

func LoadConfig() (*Config, error) {
	region := getEnv("BEDROCK_REGION", "")
	if region == "" {
		region = getEnv("AWS_REGION", "us-east-1")
	}

	config := &Config{
		AWSRegion:          region,
		EmbeddingModelId:   getEnv("BEDROCK_EMBEDDING_MODEL", "amazon.titan-embed-text-v2:0"),
		KnowledgeBaseId:    "R1DHVCY9K7",                                                                   // Hardcoded Knowledge Base ID
		GenerativeModelId:  getEnv("BEDROCK_GENERATIVE_MODEL", "anthropic.claude-haiku-4-5-20251001-v1:0"), // Claude 3.5 Haiku
		SystemInstructions: strings.TrimSpace(systemInstructions),
		MaxQuestionLength:  getEnvAsInt("MAX_QUESTION_LENGTH", 1000),
		RetryAttempts:      getEnvAsInt("RETRY_ATTEMPTS", 3),
		OpenSearchEndpoint: getEnv("OPENSEARCH_ENDPOINT", ""),
		OpenSearchIndex:    getEnv("OPENSEARCH_INDEX", "bedrock-knowledge-base-default-index"),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.AWSRegion == "" {
		return fmt.Errorf("AWS_REGION is required")
	}
	if c.EmbeddingModelId == "" {
		return fmt.Errorf("BEDROCK_EMBEDDING_MODEL is required")
	}
	if c.KnowledgeBaseId == "" {
		return fmt.Errorf("BEDROCK_KB_ID is required")
	}
	if c.GenerativeModelId == "" {
		return fmt.Errorf("BEDROCK_GENERATIVE_MODEL is required")
	}
	if c.MaxQuestionLength <= 0 {
		return fmt.Errorf("MAX_QUESTION_LENGTH must be positive")
	}
	if c.RetryAttempts < 0 {
		return fmt.Errorf("RETRY_ATTEMPTS must be non-negative")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
