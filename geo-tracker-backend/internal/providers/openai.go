package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/adoreme/geo-tracker/internal/config"
)

type openAIProvider struct {
	name string
	cfg  config.ProviderConfig
	url  string
}

func NewOpenAIProvider(cfg config.ProviderConfig) Provider {
	return &openAIProvider{
		name: "chatgpt",
		cfg:  cfg,
		url:  "https://api.openai.com/v1/chat/completions",
	}
}

func (p *openAIProvider) Name() string {
	return p.name
}

func (p *openAIProvider) Probe(ctx context.Context, prompt string) (ProbeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(p.cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	start := time.Now()
	// If using Gemini via native endpoint, the payload structure is different from OpenAI
	if p.name == "gemini" {
		payload := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]string{
						{"text": prompt},
					},
				},
			},
			"tools": []map[string]interface{}{
				{
					"google_search": map[string]interface{}{},
				},
			},
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return ProbeResponse{}, err
		}

		// Correct URL for native Gemini generateContent
		url := fmt.Sprintf("%s:generateContent?key=%s", p.url, p.cfg.APIKey)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return ProbeResponse{}, err
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return ProbeResponse{}, err
		}
		defer resp.Body.Close()

		latency := int(time.Since(start).Milliseconds())

		if resp.StatusCode != http.StatusOK {
			return ProbeResponse{}, fmt.Errorf("gemini native api error: %s", resp.Status)
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
			PromptFeedback struct {
				BlockReason string `json:"blockReason"`
			} `json:"promptFeedback"`
			UsageMetadata struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
			} `json:"usageMetadata"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return ProbeResponse{}, err
		}

		if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
			return ProbeResponse{
				RawText:      result.Candidates[0].Content.Parts[0].Text,
				TokensInput:  result.UsageMetadata.PromptTokenCount,
				TokensOutput: result.UsageMetadata.CandidatesTokenCount,
				LatencyMS:    latency,
				ModelVersion: p.cfg.ProbeModel, // Native API doesn't always return exact version string in same way
			}, nil
		}

		// Handle blocked responses or empty candidates
		if result.PromptFeedback.BlockReason != "" {
			return ProbeResponse{}, fmt.Errorf("gemini blocked prompt: %s", result.PromptFeedback.BlockReason)
		}
		if len(result.Candidates) > 0 && result.Candidates[0].FinishReason != "" {
			return ProbeResponse{}, fmt.Errorf("gemini empty candidate: finish reason %s", result.Candidates[0].FinishReason)
		}

		return ProbeResponse{}, fmt.Errorf("empty response from gemini native")
	}

	payload := map[string]interface{}{
		"model": p.cfg.ProbeModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return ProbeResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return ProbeResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ProbeResponse{}, err
	}
	defer resp.Body.Close()

	latency := int(time.Since(start).Milliseconds())

	if resp.StatusCode != http.StatusOK {
		return ProbeResponse{}, fmt.Errorf("%s api error: %s", p.name, resp.Status)
	}

	var result struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `db:"prompt_tokens" json:"prompt_tokens"`
			CompletionTokens int `db:"completion_tokens" json:"completion_tokens"`
		} `json:"usage"`
		Citations []string `json:"citations"` // For Perplexity
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ProbeResponse{}, err
	}

	if len(result.Choices) == 0 {
		return ProbeResponse{}, fmt.Errorf("empty response from %s", p.name)
	}

	return ProbeResponse{
		RawText:      result.Choices[0].Message.Content,
		CitedURLs:    result.Citations,
		TokensInput:  result.Usage.PromptTokens,
		TokensOutput: result.Usage.CompletionTokens,
		LatencyMS:    latency,
		ModelVersion: result.Model,
	}, nil
}

type perplexityProvider struct {
	openAIProvider
}

func NewPerplexityProvider(cfg config.ProviderConfig) Provider {
	return &perplexityProvider{
		openAIProvider: openAIProvider{
			name: "perplexity",
			cfg:  cfg,
			url:  "https://api.perplexity.ai/chat/completions",
		},
	}
}

type geminiProvider struct {
	openAIProvider
}

func NewGeminiProvider(cfg config.ProviderConfig) Provider {
	return &geminiProvider{
		openAIProvider: openAIProvider{
			name: "gemini",
			cfg:  cfg,
			url:  fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s", cfg.ProbeModel),
		},
	}
}
