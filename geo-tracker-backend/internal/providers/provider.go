package providers

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
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

// ResolveRedirects takes a list of URLs and follows redirects (like Gemini grounding URLs)
// to find the final destination.
func ResolveRedirects(urls []string) []string {
	if len(urls) == 0 {
		return urls
	}

	var wg sync.WaitGroup
	resolved := make([]string, len(urls))
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't actually follow, just get the next hop
		},
		Timeout: 5 * time.Second,
	}

	for i, u := range urls {
		wg.Add(1)
		go func(idx int, urlStr string) {
			defer wg.Done()
			// Only resolve if it's a known redirect proxy (like Gemini's)
			if !strings.Contains(urlStr, "grounding-api-redirect") {
				resolved[idx] = urlStr
				return
			}

			resp, err := client.Head(urlStr)
			if err != nil {
				resolved[idx] = urlStr
				return
			}
			defer resp.Body.Close()

			if loc := resp.Header.Get("Location"); loc != "" {
				resolved[idx] = loc
			} else {
				resolved[idx] = urlStr
			}
		}(i, u)
	}

	wg.Wait()
	return resolved
}
