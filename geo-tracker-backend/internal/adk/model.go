package adk

import (
	"context"
	"fmt"

	"github.com/adoreme/geo-tracker/internal/config"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

// NewADKModel returns an ADK model instance for the given model string,
// using the provider and API key from ADKConfig.
//
// provider = "gemini"    → uses google.golang.org/adk/model/gemini
//
// provider = "anthropic" → currently unsupported in ADK Go v1.4, returns error
//
// The model string must match the chosen provider's naming convention:
//   gemini:    "gemini-2.0-flash", "gemini-1.5-pro", etc.
func NewADKModel(ctx context.Context, cfg config.ADKConfig, modelStr string) (adkmodel.LLM, error) {
	switch cfg.Provider {
	case "gemini":
		return gemini.NewModel(ctx, modelStr, &genai.ClientConfig{
			APIKey: cfg.APIKey,
		})
	case "anthropic":
		return nil, fmt.Errorf("anthropic provider not yet supported in ADK Go v1.4")
	default:
		return nil, fmt.Errorf("unsupported adk.provider %q: must be 'gemini' or 'anthropic'", cfg.Provider)
	}
}
