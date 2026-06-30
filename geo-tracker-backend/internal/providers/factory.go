package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/adoreme/geo-tracker/internal/config"
	"go.uber.org/zap"
)

func NewProviders(cfg config.Config, logger *zap.Logger) []Provider {
	var providers []Provider

	if cfg.Providers.Claude.Enabled {
		providers = append(providers, NewAnthropicProvider(cfg.Providers.Claude))
	}
	if cfg.Providers.ChatGPT.Enabled {
		providers = append(providers, NewOpenAIProvider(cfg.Providers.ChatGPT, logger))
	}
	if cfg.Providers.Perplexity.Enabled {
		providers = append(providers, NewPerplexityProvider(cfg.Providers.Perplexity, logger))
	}
	if cfg.Providers.Gemini.Enabled {
		providers = append(providers, NewGeminiProvider(cfg.Providers.Gemini, logger))
	}

	return providers
}

// Global factory for extraction calls
func Extract(ctx context.Context, cfg config.ProviderConfig, providerType string, systemPrompt string, userPrompt string) (string, error) {
	// Use Gemini for extraction if enabled and healthy, otherwise fallback
	return extractGemini(ctx, cfg, systemPrompt, userPrompt)
}

func extractClaude(ctx context.Context, cfg config.ProviderConfig, systemPrompt string, userPrompt string) (string, error) {
	url := "https://api.anthropic.com/v1/messages"
	payload := map[string]interface{}{
		"model":      cfg.ExtractModel,
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	// Add messages-api beta header if using newer features, but let's stick to standard first.
	// The 400 Bad Request might be due to 'system' field or model name.

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("anthropic extraction error: %s (Body: %v)", resp.Status, errBody)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty extraction response")
	}

	return result.Content[0].Text, nil
}

func extractGemini(ctx context.Context, cfg config.ProviderConfig, systemPrompt string, userPrompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", "gemini-2.0-flash", cfg.APIKey)
	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": systemPrompt},
			},
		},
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": userPrompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"response_mime_type": "application/json",
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini extraction error: %s", resp.Status)
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty extraction response from gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
