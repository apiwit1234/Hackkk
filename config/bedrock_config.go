package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AWSRegion         string
	EmbeddingModelId  string
	KnowledgeBaseId   string
	GenerativeModelId string
	MaxQuestionLength int
	RetryAttempts     int
}

func LoadConfig() (*Config, error) {
	// Try BEDROCK_REGION first (for Lambda), then fall back to AWS_REGION (for local)
	region := getEnv("BEDROCK_REGION", "")
	if region == "" {
		region = getEnv("AWS_REGION", "us-east-1")
	}
	
	config := &Config{
		AWSRegion:         region,
		EmbeddingModelId:  "anthropic.claude-sonnet-4-5-20250929-v1:0",
		KnowledgeBaseId:   getEnv("BEDROCK_KB_ID", "R1DHVCY9K7"),
		GenerativeModelId: getEnv("BEDROCK_GENERATIVE_MODEL", "anthropic.claude-sonnet-4-5-20250929-v1:0"),
		MaxQuestionLength: getEnvAsInt("MAX_QUESTION_LENGTH", 1000),
		RetryAttempts:     getEnvAsInt("RETRY_ATTEMPTS", 3),
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
