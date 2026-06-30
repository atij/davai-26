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

type anthropicProvider struct {
	cfg config.ProviderConfig
}

func NewAnthropicProvider(cfg config.ProviderConfig) Provider {
	return &anthropicProvider{cfg: cfg}
}

func (p *anthropicProvider) Name() string {
	return "claude"
}

// BatchResponse structure for Anthropic API
type batchResponse struct {
	ID               string `json:"id"`
	ProcessingStatus string `json:"processing_status"`
	ResultsURL       string `json:"results_url"`
}

func (p *anthropicProvider) Probe(ctx context.Context, prompt string) (ProbeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(p.cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	start := time.Now()
	url := "https://api.anthropic.com/v1/messages"
	payload := map[string]interface{}{
		"model":      p.cfg.ProbeModel,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"tools": []map[string]interface{}{
			{
				"type": "web_search_20250305",
				"name": "web_search",
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return ProbeResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return ProbeResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ProbeResponse{}, err
	}
	defer resp.Body.Close()

	latency := int(time.Since(start).Milliseconds())

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errBody)
		return ProbeResponse{}, fmt.Errorf("anthropic api error: %s: %s (%s)", resp.Status, errBody.Error.Message, errBody.Error.Type)
	}

	var result struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Content []struct {
			Type      string `json:"type"`
			Text      string `json:"text"`
			ID        string `json:"id"`
			Name      string `json:"name"`
			Input     any    `json:"input"`
			Citations []struct {
				Citation struct {
					URL string `json:"url"`
				} `json:"citation"`
			} `json:"citations"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ProbeResponse{}, err
	}

	if len(result.Content) == 0 {
		return ProbeResponse{}, fmt.Errorf("empty response from anthropic")
	}

	var rawText string
	var citedURLs []string
	for _, block := range result.Content {
		if block.Type == "text" {
			rawText += block.Text
			for _, c := range block.Citations {
				if c.Citation.URL != "" {
					citedURLs = append(citedURLs, c.Citation.URL)
				}
			}
		} else if block.Type == "tool_use" {
			// Log tool usage at debug level
			// We don't have a logger injected here, but we follow the principle of not info-logging raw text
			// Since this is internal/providers, we just process it.
		}
	}

	return ProbeResponse{
		RawText:      rawText,
		CitedURLs:    citedURLs,
		TokensInput:  result.Usage.InputTokens,
		TokensOutput: result.Usage.OutputTokens,
		LatencyMS:    latency,
		ModelVersion: result.Model,
	}, nil
}

// Support for Batching (as specified in system prompt, we implement the batch capability)
// Note: The Provider interface currently only specifies Probe.
// We keep Batch logic here for future enablement or for specific high-volume commands.
func (p *anthropicProvider) createBatch(ctx context.Context, prompts []string) (string, error) {
	url := "https://api.anthropic.com/v1/messages/batches"
	
	requests := make([]map[string]interface{}, len(prompts))
	for i, prompt := range prompts {
		requests[i] = map[string]interface{}{
			"custom_id": fmt.Sprintf("req-%d", i),
			"params": map[string]interface{}{
				"model":      p.cfg.ProbeModel,
				"max_tokens": 1024,
				"messages": []map[string]string{
					{"role": "user", "content": prompt},
				},
			},
		}
	}

	payload := map[string]interface{}{"requests": requests}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var batch batchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return "", err
	}

	return batch.ID, nil
}
