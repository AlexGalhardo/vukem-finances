package ai

import (
	"context"
	"encoding/json"
	"strings"
)

// The Messages API, read on 2026-11-21 from
// https://platform.claude.com/docs/en/api/messages
const (
	anthropicEndpoint = "https://api.anthropic.com/v1/messages"
	anthropicVersion  = "2023-06-01"
	// DefaultAnthropicModel is used when the credential names none; the owner
	// can set any model the key may use.
	DefaultAnthropicModel = "claude-sonnet-4-6"
)

// Anthropic talks to the Anthropic Messages API.
type Anthropic struct {
	// Endpoint is the URL of the Messages API; tests point it at a local server.
	Endpoint string

	key   Key
	model string
	http  *HTTP
}

// NewAnthropic builds the provider.
func NewAnthropic(key Key, model string, h *HTTP) *Anthropic {
	if model == "" {
		model = DefaultAnthropicModel
	}

	return &Anthropic{Endpoint: anthropicEndpoint, key: key, model: model, http: h}
}

type anthropicRequest struct {
	Model        string             `json:"model"`
	MaxTokens    int                `json:"max_tokens"`
	System       string             `json:"system,omitempty"`
	Messages     []anthropicMessage `json:"messages"`
	Temperature  *float64           `json:"temperature,omitempty"`
	OutputConfig *anthropicOutput   `json:"output_config,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicOutput struct {
	Format struct {
		Type   string          `json:"type"`
		Schema json.RawMessage `json:"schema"`
	} `json:"format"`
}

type anthropicResponse struct {
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Complete implements Provider.
func (a *Anthropic) Complete(ctx context.Context, req Request) (Response, error) {
	body := anthropicRequest{Model: a.model, MaxTokens: req.MaxTokens, System: req.System}
	if body.MaxTokens <= 0 {
		body.MaxTokens = 1024
	}

	for _, m := range req.Messages {
		body.Messages = append(body.Messages, anthropicMessage{Role: m.Role, Content: m.Content})
	}

	if req.Temperature > 0 {
		body.Temperature = &req.Temperature
	}

	if len(req.JSONSchema) > 0 {
		body.OutputConfig = &anthropicOutput{}
		body.OutputConfig.Format.Type = "json_schema"
		body.OutputConfig.Format.Schema = req.JSONSchema
	}

	var out anthropicResponse

	headers := map[string]string{"x-api-key": a.key.Reveal(), "anthropic-version": anthropicVersion}
	if err := a.http.postJSON(ctx, a.Endpoint, headers, body, &out); err != nil {
		return Response{}, err
	}

	var text strings.Builder

	for _, block := range out.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	return Response{
		Text: text.String(), InputTokens: out.Usage.InputTokens, OutputTokens: out.Usage.OutputTokens,
		Model: out.Model, Provider: KindAnthropic,
	}, nil
}

// Check implements Provider with a one-turn, 16-token request.
func (a *Anthropic) Check(ctx context.Context) error {
	_, err := a.Complete(ctx, checkRequest())

	return err
}
