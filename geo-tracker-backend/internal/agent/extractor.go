package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/adoreme/geo-tracker/internal/config"
	"github.com/adoreme/geo-tracker/internal/providers"
)

type GEOSignal struct {
	BrandMentioned       bool     `json:"brand_mentioned"`
	Sentiment            string   `json:"sentiment"`           // positive|neutral|negative|not_mentioned
	MentionCount         int      `json:"mention_count"`
	RecommendationRank   *int     `json:"recommendation_rank"` // nil if not mentioned
	CompetitorsMentioned []string `json:"competitors_mentioned"`
	CitedURLs            []string `json:"cited_urls"`
	Summary              string   `json:"summary"`
	ReasoningNote        string   `json:"reasoning_note"`
}

const systemPrompt = `You are a brand visibility extraction agent. 
Analyze the provided AI response and extract structured signals for the brand: "{{BRAND}}".
Return ONLY valid JSON. No markdown fences.

JSON Schema:
{
  "brand_mentioned": boolean,
  "sentiment": "positive"|"neutral"|"negative"|"not_mentioned",
  "mention_count": integer,
  "recommendation_rank": integer|null,
  "competitors_mentioned": string[],
  "cited_urls": string[],
  "summary": "2-3 sentence summary of the mention",
  "reasoning_note": "Internal reasoning for these values"
}

Rules:
1. brand_mentioned is true if the brand "{{BRAND}}" is mentioned in a relevant way.
2. sentiment is relative to "{{BRAND}}".
3. recommendation_rank is the 1-based index if brands are ranked/listed. Null if not listed.
4. competitors_mentioned: list other brands mentioned in the same context.
5. cited_urls: extract any URLs mentioned in the text.`

func Extract(ctx context.Context, cfg config.ProviderConfig, providerType, rawText, brand string) (GEOSignal, error) {
	sPrompt := strings.ReplaceAll(systemPrompt, "{{BRAND}}", brand)
	userPrompt := fmt.Sprintf("Response to analyze:\n\n%s", rawText)

	resText, err := providers.Extract(ctx, cfg, providerType, sPrompt, userPrompt)
	if err != nil {
		return GEOSignal{}, err
	}

	// Clean markdown fences if any
	resText = strings.TrimPrefix(resText, "```json")
	resText = strings.TrimPrefix(resText, "```")
	resText = strings.TrimSuffix(resText, "```")
	resText = strings.TrimSpace(resText)

	var signal GEOSignal
	if err := json.Unmarshal([]byte(resText), &signal); err != nil {
		return GEOSignal{}, fmt.Errorf("unmarshal extraction: %w: %s", err, resText)
	}

	return signal, nil
}
