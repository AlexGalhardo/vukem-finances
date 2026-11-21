package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

const stopMaxTokens = "max_tokens"

// The Messages API, read on 2026-11-21 from
// https://platform.claude.com/docs/en/api/messages
const (
	anthropicEndpoint = "https://api.anthropic.com/v1/messages"
	anthropicVersion  = "2023-06-01"
	// DefaultAnthropicModel is used when the credential names none; the owner
	// can set any model the key may use.
	DefaultAnthropicModel = "claude-opus-5-5"
	// anthropicMaxTokens counts thinking too: a low default cuts the answer off.
	anthropicMaxTokens = 16000
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
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Complete implements Provider. Temperature is not sent: current models answer
// 400 to it. A refused or cut-off answer comes with HTTP 200 and is an error here.
func (a *Anthropic) Complete(ctx context.Context, req Request) (Response, error) {
	body := anthropicRequest{Model: a.model, MaxTokens: req.MaxTokens, System: req.System}
	if body.MaxTokens <= 0 {
		body.MaxTokens = anthropicMaxTokens
	}

	for _, m := range req.Messages {
		body.Messages = append(body.Messages, anthropicMessage(m))
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

	if out.StopReason == "refusal" || out.StopReason == stopMaxTokens {
		return Response{}, &ProviderError{Status: http.StatusOK, Type: out.StopReason, Class: ErrBadRequest}
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

// Check implements Provider with a one-turn, 16-token request. Hitting those
// 16 tokens is the normal outcome and still proves that the key works.
func (a *Anthropic) Check(ctx context.Context) error {
	_, err := a.Complete(ctx, checkRequest())

	if pe, ok := errors.AsType[*ProviderError](err); ok && pe.Type == stopMaxTokens {
		return nil
	}

	return err
}
