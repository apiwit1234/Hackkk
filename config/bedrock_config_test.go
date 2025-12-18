package config

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: bedrock-question-search, Property: Configuration validation
// Validates: Requirements 6.4
func TestConfigValidation_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("valid config passes validation", prop.ForAll(
		func(region, modelId, kbId string, maxLen, retries int) bool {
			config := &Config{
				AWSRegion:         region,
				EmbeddingModelId:  modelId,
				KnowledgeBaseId:   kbId,
				MaxQuestionLength: maxLen,
				RetryAttempts:     retries,
			}
			err := config.Validate()
			return err == nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }), // region
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }), // modelId
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }), // kbId
		gen.IntRange(1, 10000),   // maxLen
		gen.IntRange(0, 10),      // retries
	))

	properties.Property("empty region fails validation", prop.ForAll(
		func(modelId, kbId string, maxLen, retries int) bool {
			config := &Config{
				AWSRegion:         "",
				EmbeddingModelId:  modelId,
				KnowledgeBaseId:   kbId,
				MaxQuestionLength: maxLen,
				RetryAttempts:     retries,
			}
			err := config.Validate()
			return err != nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 10000),
		gen.IntRange(0, 10),
	))

	properties.Property("empty embedding model fails validation", prop.ForAll(
		func(region, kbId string, maxLen, retries int) bool {
			config := &Config{
				AWSRegion:         region,
				EmbeddingModelId:  "",
				KnowledgeBaseId:   kbId,
				MaxQuestionLength: maxLen,
				RetryAttempts:     retries,
			}
			err := config.Validate()
			return err != nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 10000),
		gen.IntRange(0, 10),
	))

	properties.Property("empty knowledge base ID fails validation", prop.ForAll(
		func(region, modelId string, maxLen, retries int) bool {
			config := &Config{
				AWSRegion:         region,
				EmbeddingModelId:  modelId,
				KnowledgeBaseId:   "",
				MaxQuestionLength: maxLen,
				RetryAttempts:     retries,
			}
			err := config.Validate()
			return err != nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 10000),
		gen.IntRange(0, 10),
	))

	properties.Property("non-positive max question length fails validation", prop.ForAll(
		func(region, modelId, kbId string, retries int) bool {
			config := &Config{
				AWSRegion:         region,
				EmbeddingModelId:  modelId,
				KnowledgeBaseId:   kbId,
				MaxQuestionLength: 0,
				RetryAttempts:     retries,
			}
			err := config.Validate()
			return err != nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(0, 10),
	))

	properties.Property("negative retry attempts fails validation", prop.ForAll(
		func(region, modelId, kbId string, maxLen int) bool {
			config := &Config{
				AWSRegion:         region,
				EmbeddingModelId:  modelId,
				KnowledgeBaseId:   kbId,
				MaxQuestionLength: maxLen,
				RetryAttempts:     -1,
			}
			err := config.Validate()
			return err != nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.IntRange(1, 10000),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
