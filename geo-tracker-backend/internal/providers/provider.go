package providers

import (
	"context"
)

type ProbeResponse struct {
	RawText      string
	CitedURLs    []string // Perplexity populates; others return empty slice
	TokensInput  int
	TokensOutput int
	LatencyMS    int
	ModelVersion string
}

type Provider interface {
	Name() string
	Probe(ctx context.Context, prompt string) (ProbeResponse, error)
}
