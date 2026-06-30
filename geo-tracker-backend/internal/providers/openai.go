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

type openAIProvider struct {
	name   string
	cfg    config.ProviderConfig
	url    string
	logger *zap.Logger
}

func NewOpenAIProvider(cfg config.ProviderConfig, logger *zap.Logger) Provider {
	return &openAIProvider{
		name:   "chatgpt",
		cfg:    cfg,
		url:    "https://api.openai.com/v1/responses",
		logger: logger,
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
				FinishReason      string `json:"finishReason"`
				GroundingMetadata struct {
					GroundingChunks []struct {
						Web struct {
							URI   string `json:"uri"`
							Title string `json:"title"`
						} `json:"web"`
					} `json:"groundingChunks"`
				} `json:"groundingMetadata"`
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
			var citedURLs []string
			for _, chunk := range result.Candidates[0].GroundingMetadata.GroundingChunks {
				if chunk.Web.URI != "" {
					citedURLs = append(citedURLs, chunk.Web.URI)
				}
			}

			return ProbeResponse{
				RawText:      result.Candidates[0].Content.Parts[0].Text,
				CitedURLs:    ResolveRedirects(citedURLs),
				TokensInput:  result.UsageMetadata.PromptTokenCount,
				TokensOutput: result.UsageMetadata.CandidatesTokenCount,
				LatencyMS:    latency,
				ModelVersion: p.cfg.ProbeModel,
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

	// Perplexity implementation
	if p.name == "perplexity" {
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
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
			Citations []string `json:"citations"`
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

	// ChatGPT (OpenAI Responses API) implementation
	payload := map[string]interface{}{
		"model": p.cfg.ProbeModel,
		"input": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"tools": []map[string]interface{}{
			{
				"type": "web_search",
			},
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
		var errBody struct {
			Error struct {
				Message string `json:"message"`
				Code    string `json:"code"`
				Param   string `json:"param"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		// Try to decode error body, but don't fail if it's not JSON
		var bodyBytes bytes.Buffer
		_, _ = bodyBytes.ReadFrom(resp.Body)
		_ = json.Unmarshal(bodyBytes.Bytes(), &errBody)
		
		errMsg := errBody.Error.Message
		if errMsg == "" {
			errMsg = bodyBytes.String()
		}
		return ProbeResponse{}, fmt.Errorf("%s api error: %s: %s (Type: %s, Code: %s)", p.name, resp.Status, errMsg, errBody.Error.Type, errBody.Error.Code)
	}

	var rawBody bytes.Buffer
	_, _ = rawBody.ReadFrom(resp.Body)

	var result struct {
		ID     string `json:"id"`
		Object string `json:"object"`
		Model  string `json:"model"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				Annotations []struct {
					Type        string `json:"type"`
					URLCitation struct {
						URL   string `json:"url"`
						Title string `json:"title"`
					} `json:"url_citation"`
				} `json:"annotations"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(rawBody.Bytes(), &result); err != nil {
		return ProbeResponse{}, fmt.Errorf("decode %s response: %w (body: %s)", p.name, err, rawBody.String())
	}

	var rawText string
	var citedURLs []string
	for _, out := range result.Output {
		if out.Type == "message" {
			for _, content := range out.Content {
				if content.Type == "text" || content.Type == "output_text" {
					rawText += content.Text
					for _, ann := range content.Annotations {
						if ann.Type == "url_citation" && ann.URLCitation.URL != "" {
							citedURLs = append(citedURLs, ann.URLCitation.URL)
						}
					}
				}
			}
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

type perplexityProvider struct {
	openAIProvider
}

func NewPerplexityProvider(cfg config.ProviderConfig, logger *zap.Logger) Provider {
	return &perplexityProvider{
		openAIProvider: openAIProvider{
			name:   "perplexity",
			cfg:    cfg,
			url:    "https://api.perplexity.ai/chat/completions",
			logger: logger,
		},
	}
}

type geminiProvider struct {
	openAIProvider
}

func NewGeminiProvider(cfg config.ProviderConfig, logger *zap.Logger) Provider {
	return &geminiProvider{
		openAIProvider: openAIProvider{
			name:   "gemini",
			cfg:    cfg,
			url:    fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s", cfg.ProbeModel),
			logger: logger,
		},
	}
}
