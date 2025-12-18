package errors

import "fmt"

const (
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeEmbedding     = "EMBEDDING_ERROR"
	ErrCodeKnowledgeBase = "KB_ERROR"
	ErrCodeThrottling    = "THROTTLING_ERROR"
	ErrCodeAWSService    = "AWS_SERVICE_ERROR"
)

type BedrockError struct {
	Code    string
	Message string
	Cause   error
}

func (e *BedrockError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *BedrockError) Unwrap() error {
	return e.Cause
}

func NewValidationError(message string) *BedrockError {
	return &BedrockError{
		Code:    ErrCodeValidation,
		Message: message,
	}
}

func NewEmbeddingError(message string, cause error) *BedrockError {
	return &BedrockError{
		Code:    ErrCodeEmbedding,
		Message: message,
		Cause:   cause,
	}
}

func NewKnowledgeBaseError(message string, cause error) *BedrockError {
	return &BedrockError{
		Code:    ErrCodeKnowledgeBase,
		Message: message,
		Cause:   cause,
	}
}

func NewThrottlingError(message string, cause error) *BedrockError {
	return &BedrockError{
		Code:    ErrCodeThrottling,
		Message: message,
		Cause:   cause,
	}
}

func NewAWSServiceError(message string, cause error) *BedrockError {
	return &BedrockError{
		Code:    ErrCodeAWSService,
		Message: message,
		Cause:   cause,
	}
}
