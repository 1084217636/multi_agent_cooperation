package llm

import "errors"

// 定义LLM Provider相关错误
var (
	ErrProviderNotFound   = errors.New("provider not found")
	ErrInvalidAPIKey      = errors.New("invalid API key")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrModelNotSupported  = errors.New("model not supported")
	ErrContextTooLong     = errors.New("context too long")
	ErrNetworkError       = errors.New("network error")
	ErrAPIError           = errors.New("API error")
	ErrTimeout            = errors.New("request timeout")
)
