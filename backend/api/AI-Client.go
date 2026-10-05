package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type AILocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type AIContext struct {
	Location     *AILocation   `json:"location"`
	SelectedDate string        `json:"selectedDate"`
	ActiveLayers []interface{} `json:"activeLayers"`
	NearbyEvents []interface{} `json:"nearbyEvents"`
}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIChatRequest struct {
	Message string      `json:"message"`
	History []AIMessage `json:"history"`
	Context AIContext   `json:"context"`
}

type AIChatResponse struct {
	Configured bool   `json:"configured"`
	Provider   string `json:"provider,omitempty"`
	Reply      string `json:"reply"`
}

type AIProvider interface {
	Name() string
	Chat(ctx context.Context, system string, messages []AIMessage) (string, error)
}

type AIService struct {
	provider AIProvider
}

const embeddedGeminiKey = "AQ.Ab8RN6KdQl_y_HNGRYjPKuKSGGyMe0nFz3XRK8tQdKrkhan8Dg"

func firstEnv(names ...string) string {
	for _, n := range names {
		if v := strings.TrimSpace(os.Getenv(n)); v != "" {
			return v
		}
	}
	return ""
}

func NewAIService() *AIService {
	geminiKey := firstEnv("GEMINI_API_KEY", "GOOGLE_API_KEY")
	if geminiKey == "" {
		geminiKey = embeddedGeminiKey
	}
	anthropicKey := firstEnv("ANTHROPIC_API_KEY")
	model := firstEnv("AI_MODEL")

	provider := strings.ToLower(firstEnv("AI_PROVIDER"))
	if provider == "" {
		if geminiKey != "" {
			provider = "gemini"
		} else if anthropicKey != "" {
			provider = "anthropic"
		}
	}

	switch provider {
	case "gemini":
		if geminiKey != "" {
			if model == "" {
				model = "gemini-2.5-flash"
			}
			return &AIService{provider: NewGeminiProvider(geminiKey, model)}
		}
	case "anthropic":
		if anthropicKey != "" {
			if model == "" {
				model = "claude-sonnet-5-5"
			}
			return &AIService{provider: NewAnthropicProvider(anthropicKey, model)}
		}
	}
	return &AIService{}
}

func (s *AIService) Configured() bool { return s.provider != nil }

func (s *AIService) ProviderName() string {
	if s.provider == nil {
		return ""
	}
	return s.provider.Name()
}

const notConfiguredMessage = "The AI service is not configured on the server, so no real answer can be given. " +
	"Set the GEMINI_API_KEY environment variable (free key: aistudio.google.com/app/apikey) and restart the backend. " +
	"Your map context was received correctly (location, date, layers, nearby events)."

const systemPrompt = `You are the "Earth System Detective" assistant inside a NASA Earth observation web map.

Rules:
- Base your answers ONLY on the context JSON provided (selected location, selected date, active NASA GIBS layers, nearby NASA EONET events) and on well-established general scientific knowledge.
- You CANNOT see the satellite imagery pixels. Never claim to have observed a specific feature in the imagery.
- Do not invent measurements, dates, event details or causes. If the context does not contain what is needed, say so plainly and suggest what the user could check (another layer, another date, a nearby event).
- Remember that different NASA layers have different temporal resolutions (daily imagery vs. multi-day composites). Mention this when comparing dates.
- Natural events in the context are a snapshot of currently open EONET events, not necessarily events on the selected date.
- Clearly separate "what the provided data shows" from "general background / possible explanations".
- Be concise and clear. Plain text, no markdown tables.`

func (s *AIService) Chat(ctx context.Context, req AIChatRequest) (*AIChatResponse, error) {
	if s.provider == nil {
		return &AIChatResponse{Configured: false, Reply: notConfiguredMessage}, nil
	}

	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		return nil, errors.New("message cannot be empty")
	}
	if len(req.Message) > 4000 {
		return nil, errors.New("message is too long (max 4000 characters)")
	}

	ctxJSON, err := json.MarshalIndent(req.Context, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not encode context: %w", err)
	}
	system := systemPrompt + "\n\nCurrent map context:\n" + string(ctxJSON)

	history := make([]AIMessage, 0, len(req.History)+1)
	for _, m := range req.History {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		content := strings.TrimSpace(m.Content)
		if (role != "user" && role != "assistant") || content == "" {
			continue
		}
		history = append(history, AIMessage{Role: role, Content: content})
	}
	if len(history) > 20 {
		history = history[len(history)-20:]
	}
	for len(history) > 0 && history[0].Role != "user" {
		history = history[1:]
	}
	history = append(history, AIMessage{Role: "user", Content: req.Message})

	reply, err := s.provider.Chat(ctx, system, history)
	if err != nil {
		return nil, err
	}
	return &AIChatResponse{Configured: true, Provider: s.provider.Name(), Reply: reply}, nil
}

type GeminiProvider struct {
	client *resty.Client
	apiKey string
	model  string
}

func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	return &GeminiProvider{
		client: resty.New().SetTimeout(60 * time.Second),
		apiKey: apiKey,
		model:  model,
	}
}

func (p *GeminiProvider) Name() string { return "gemini:" + p.model }

func (p *GeminiProvider) Chat(ctx context.Context, system string, messages []AIMessage) (string, error) {
	contents := make([]map[string]interface{}, 0, len(messages))
	for _, m := range messages {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]interface{}{
			"role":  role,
			"parts": []map[string]string{{"text": m.Content}},
		})
	}

	body := map[string]interface{}{
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{{"text": system}},
		},
		"contents": contents,

		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 2048,
			"temperature":     0.4,
		},
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		PromptFeedback *struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/" + p.model + ":generateContent"
	resp, err := p.client.R().
		SetContext(ctx).
		SetHeader("x-goog-api-key", p.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		SetResult(&result).
		SetError(&result).
		Post(url)
	if err != nil {
		return "", fmt.Errorf("gemini request failed: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 429 {
			return "", errors.New("Gemini free-tier rate limit reached. Wait a minute and try again")
		}
		msg := resp.Status()
		if result.Error != nil && result.Error.Message != "" {
			msg = result.Error.Message
		}
		return "", fmt.Errorf("gemini api error (%d): %s", resp.StatusCode(), msg)
	}

	if result.PromptFeedback != nil && result.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("gemini blocked the request (%s)", result.PromptFeedback.BlockReason)
	}

	var sb strings.Builder
	finish := ""
	if len(result.Candidates) > 0 {
		finish = result.Candidates[0].FinishReason
		for _, part := range result.Candidates[0].Content.Parts {
			sb.WriteString(part.Text)
		}
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		if finish == "MAX_TOKENS" {
			return "", errors.New("gemini ran out of output tokens before answering; try a shorter question")
		}
		return "", errors.New("gemini returned an empty response")
	}
	return text, nil
}

type AnthropicProvider struct {
	client *resty.Client
	apiKey string
	model  string
}

func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	return &AnthropicProvider{
		client: resty.New().SetTimeout(60 * time.Second),
		apiKey: apiKey,
		model:  model,
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic:" + p.model }

func (p *AnthropicProvider) Chat(ctx context.Context, system string, messages []AIMessage) (string, error) {
	body := map[string]interface{}{
		"model":      p.model,
		"max_tokens": 1000,
		"system":     system,
		"messages":   messages,
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	resp, err := p.client.R().
		SetContext(ctx).
		SetHeader("x-api-key", p.apiKey).
		SetHeader("anthropic-version", "2023-06-01").
		SetHeader("content-type", "application/json").
		SetBody(body).
		SetResult(&result).
		Post("https://api.anthropic.com/v1/messages")
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	if resp.IsError() {
		return "", fmt.Errorf("llm api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	var sb strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return "", errors.New("llm returned an empty response")
	}
	return text, nil
}
