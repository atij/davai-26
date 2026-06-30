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

Rules:
1. brand_mentioned is true if the brand "{{BRAND}}" is mentioned by name in the text.
   - NOTE: Be case-insensitive.
   - NOTE: "Adore Me" might appear as "AdoreMe".
2. sentiment is relative to "{{BRAND}}".
3. recommendation_rank is the 1-based index if brands are ranked/listed. Null if not listed.
4. competitors_mentioned: list other brands mentioned in the same context (e.g. Skims, Savage X Fenty).
5. cited_urls: extract any URLs mentioned in the text.

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
}`

type MultiBrandSignal map[string]GEOSignal // key: brand name

const multiBrandSystemPrompt = `You are a brand visibility extraction agent. 
Analyze the provided AI response and extract structured signals for the following brands: {{BRANDS}}.

Return ONLY valid JSON. No markdown fences.
The root object should be keyed by the brand name. 
If a brand is not mentioned, set "brand_mentioned": false and other fields to default values (null/empty).

JSON Schema:
{
  "Brand Name": {
    "brand_mentioned": boolean,
    "sentiment": "positive"|"neutral"|"negative"|"not_mentioned",
    "mention_count": integer,
    "recommendation_rank": integer|null,
    "competitors_mentioned": string[],
    "cited_urls": string[],
    "summary": "2-3 sentence summary of the mention",
    "reasoning_note": "Internal reasoning"
  }
}

Rules:
1. brand_mentioned is true if the brand name is found in the text (case-insensitive).
2. competitors_mentioned: list brands mentioned OTHER than the brands being extracted (e.g. Skims, Savage X Fenty).
3. cited_urls: extract any specific source links or URLs mentioned in the text.`

func ExtractMultiBrand(ctx context.Context, cfg config.ProviderConfig, providerType, rawText string, brands []string) (MultiBrandSignal, error) {
	brandsList := strings.Join(brands, ", ")
	sPrompt := strings.ReplaceAll(multiBrandSystemPrompt, "{{BRANDS}}", brandsList)
	userPrompt := fmt.Sprintf("Response to analyze:\n\n%s", rawText)

	resText, err := providers.Extract(ctx, cfg, providerType, sPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	// Clean markdown fences
	resText = strings.TrimPrefix(resText, "```json")
	resText = strings.TrimPrefix(resText, "```")
	resText = strings.TrimSuffix(resText, "```")
	resText = strings.TrimSpace(resText)

	var signal MultiBrandSignal
	if err := json.Unmarshal([]byte(resText), &signal); err != nil {
		return nil, fmt.Errorf("unmarshal multi-brand extraction: %w: %s", err, resText)
	}

	return signal, nil
}

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
