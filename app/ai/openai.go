package ai

import (
	"context"
	"encoding/json"
	"strings"
)

// The Chat Completions API, read on 2026-11-21 from
// https://developers.openai.com/api/reference/resources/chat (still documented
// as current; the page recommends Responses for new work, and every server that
// speaks "the OpenAI format" speaks Chat Completions).
const (
	openAIEndpoint = "https://api.openai.com/v1/chat/completions"
	// DefaultOpenAIModel is used when the credential names none.
	DefaultOpenAIModel = "gpt-4.1-mini"
)

// Chat talks to an OpenAI Chat Completions endpoint: OpenAI itself or any
// compatible server.
type Chat struct {
	// Endpoint is the full URL of the chat completions route.
	Endpoint string

	kind   string
	key    Key
	model  string
	http   *HTTP
	tokens string
}

// NewOpenAI builds the provider for api.openai.com.
func NewOpenAI(key Key, model string, h *HTTP) *Chat {
	if model == "" {
		model = DefaultOpenAIModel
	}

	return &Chat{Endpoint: openAIEndpoint, kind: KindOpenAI, key: key, model: model, http: h, tokens: "max_completion_tokens"}
}

// NewCompatible builds the provider for a server that speaks the OpenAI Chat
// Completions format at baseURL (already checked with CheckBaseURL). Such
// servers know max_tokens, the older and wider-supported name.
func NewCompatible(key Key, baseURL, model string, h *HTTP) *Chat {
	return &Chat{
		Endpoint: strings.TrimRight(baseURL, "/") + "/chat/completions", kind: KindOpenAICompatible,
		key: key, model: model, http: h, tokens: "max_tokens",
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Complete implements Provider.
func (c *Chat) Complete(ctx context.Context, req Request) (Response, error) {
	messages := make([]chatMessage, 0, len(req.Messages)+1)
	if req.System != "" {
		messages = append(messages, chatMessage{Role: "system", Content: req.System})
	}

	for _, m := range req.Messages {
		messages = append(messages, chatMessage(m))
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	body := map[string]any{"model": c.model, "messages": messages, c.tokens: maxTokens}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	if len(req.JSONSchema) > 0 {
		body["response_format"] = map[string]any{
			"type":        "json_schema",
			"json_schema": map[string]any{"name": "response", "schema": json.RawMessage(req.JSONSchema)},
		}
	}

	var out chatResponse

	headers := map[string]string{"Authorization": "Bearer " + c.key.Reveal()}
	if err := c.http.postJSON(ctx, c.Endpoint, headers, body, &out); err != nil {
		return Response{}, err
	}

	if len(out.Choices) == 0 {
		return Response{}, &ProviderError{Type: "no_choices", Class: ErrUnavailable}
	}

	return Response{
		Text: out.Choices[0].Message.Content, InputTokens: out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens, Model: out.Model, Provider: c.kind,
	}, nil
}

// Check implements Provider with a one-turn, 16-token request.
func (c *Chat) Check(ctx context.Context) error {
	_, err := c.Complete(ctx, checkRequest())

	return err
}
