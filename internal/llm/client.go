package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	BaseURL     string
	APIKey      string
	Model       string
	Temperature float64
	TopP        float64
	HTTPClient  *http.Client
	MaxRetries  int
}

func NewClient(baseURL, apiKey, model string, timeoutSec, maxRetries int, temperature, topP float64) *Client {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o"
	}
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	if maxRetries < 0 {
		maxRetries = 2
	}
	if temperature <= 0 {
		temperature = 0.85
	}
	if topP <= 0 {
		topP = 0.95
	}
	return &Client{
		BaseURL:     baseURL,
		APIKey:      apiKey,
		Model:       model,
		Temperature: temperature,
		TopP:        topP,
		HTTPClient:  &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		MaxRetries:  maxRetries,
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
	Temperature    float64   `json:"temperature"`
	TopP           float64   `json:"top_p"`
	ResponseFormat *Format   `json:"response_format,omitempty"`
}

type Format struct {
	Type string `json:"type"`
}

type StateDelta struct {
	HealthChange   int            `json:"health_delta"`
	AddItems       []string       `json:"inventory_add"`
	RemoveItems    []string       `json:"inventory_remove"`
	Stats          map[string]int `json:"stats_delta"`
	Bonds          map[string]int `json:"bonds_delta"`
	Karma          int            `json:"karma_delta"`
	FatePoints     int            `json:"fate_points_delta"`
	Reputation     map[string]int `json:"reputation_delta"`
	AddFlags       []string       `json:"add_flags"`
	RemoveFlags    []string       `json:"remove_flags"`
	AddEntities    []EntityDelta  `json:"add_entities"`
	UpdateEntities []EntityDelta  `json:"update_entities"`
}

type EntityDelta struct {
	Name         string   `json:"name"`
	RelationToPC string   `json:"relation_to_pc"`
	Gender       string   `json:"gender"`
	Role         string   `json:"role"`
	Status       string   `json:"status"`
	Appearance   string   `json:"appearance"`
	Traits       []string `json:"traits"`
}

type FormatHints struct {
	ChapterTitle  string      `json:"chapter_title"`
	SceneBreak    bool        `json:"scene_break"`
	Monologue     interface{} `json:"monologue"`
	DialogueLines interface{} `json:"dialogue_lines"`
}

type ResponseFormat struct {
	StoryText   string      `json:"story_text"`
	Choices     []string    `json:"choices"`
	StateUpdate StateDelta  `json:"state_update"`
	FormatHints FormatHints `json:"format_hints"`
	TwistAdded  bool        `json:"twist_added"`
	ImageScene  string      `json:"image_scene"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type GenerateResult struct {
	Response *ResponseFormat
	Usage    Usage
}

func (c *Client) GenerateTurn(ctx context.Context, messages []Message) (*GenerateResult, error) {
	return c.GenerateTurnWithModel(ctx, messages, "")
}

// GenerateText sends messages to the LLM and returns the raw text response.
// Unlike GenerateTurn, it does not require JSON output — useful for
// review/rewrite passes where the LLM should return plain text.
func (c *Client) GenerateText(ctx context.Context, messages []Message, modelOverride string) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		text, err := c.generateTextOnce(ctx, messages, modelOverride)
		if err == nil {
			return text, nil
		}
		lastErr = err
		if attempt < c.MaxRetries {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}
	return "", fmt.Errorf("llm text generation failed after %d retries: %w", c.MaxRetries, lastErr)
}

func (c *Client) generateTextOnce(ctx context.Context, messages []Message, modelOverride string) (string, error) {
	model := c.Model
	if modelOverride != "" {
		model = modelOverride
	}
	slog.Info("llm_text_request", "model", model, "override", modelOverride != "")
	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.3, // low temp for review/rewrite
		TopP:        0.9,
		// No ResponseFormat — plain text output
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API error: %d - %s", resp.StatusCode, string(body))
	}

	bodyStr := strings.TrimSpace(string(body))
	bodyStr = strings.Replace(bodyStr, "data: [DONE]", "", -1)

	var chatResp ChatResponse
	if err := json.Unmarshal([]byte(bodyStr), &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %v", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in LLM response")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

func (c *Client) GenerateTurnWithModel(ctx context.Context, messages []Message, modelOverride string) (*GenerateResult, error) {
	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		res, err := c.generateOnce(ctx, messages, modelOverride)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if attempt < c.MaxRetries {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}
	return nil, fmt.Errorf("llm generation failed after %d retries: %w", c.MaxRetries, lastErr)
}

func (c *Client) generateOnce(ctx context.Context, messages []Message, modelOverride string) (*GenerateResult, error) {
	model := c.Model
	if modelOverride != "" {
		model = modelOverride
	}
	slog.Info("llm_json_request", "model", model, "override", modelOverride != "")
	reqBody := ChatRequest{
		Model:          model,
		Messages:       messages,
		Temperature:    c.Temperature,
		TopP:           c.TopP,
		ResponseFormat: &Format{Type: "json_object"},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API error: %d - %s", resp.StatusCode, string(body))
	}

	bodyStr := strings.TrimSpace(string(body))
	bodyStr = strings.Replace(bodyStr, "data: [DONE]", "", -1)

	var chatResp ChatResponse
	if err := json.Unmarshal([]byte(bodyStr), &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response wrapper: %v\nBody: %s", err, bodyStr)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in LLM response")
	}

	content := chatResp.Choices[0].Message.Content

	// Tolerate empty objects where arrays are expected.
	reUpdate := regexp.MustCompile(`"update_entities"\s*:\s*\{\s*\}`)
	content = reUpdate.ReplaceAllString(content, `"update_entities":[]`)
	reAdd := regexp.MustCompile(`"add_entities"\s*:\s*\{\s*\}`)
	content = reAdd.ReplaceAllString(content, `"add_entities":[]`)

	var parsedResponse ResponseFormat
	if err := json.Unmarshal([]byte(content), &parsedResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from LLM content: %v\nContent: %s", err, content)
	}

	if len(parsedResponse.Choices) != 4 {
		return nil, fmt.Errorf("LLM did not return exactly 4 choices")
	}

	return &GenerateResult{
		Response: &parsedResponse,
		Usage: Usage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
	}, nil
}
