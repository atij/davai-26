package scoring

import (
	"math"

	"github.com/adoreme/geo-tracker/internal/db"
)

// CalcStabilityScore calculates stability for a prompt × provider × brand across samples.
func CalcStabilityScore(samples []db.Result) db.StabilityScore {
	if len(samples) == 0 {
		return db.StabilityScore{}
	}

	mentions := 0
	var ranks []int
	for _, s := range samples {
		if s.BrandMentioned {
			mentions++
			if s.RecommendationRank != nil {
				ranks = append(ranks, *s.RecommendationRank)
			} else {
				// Mentioned but no rank? Penalty or default rank.
				ranks = append(ranks, 10) // Assume rank 10 if mentioned but not ranked
			}
		} else {
			// Not mentioned - max penalty for rank
			ranks = append(ranks, 20)
		}
	}

	mentionRate := float64(mentions) / float64(len(samples)) * 100

	// Calculate rank variance
	var variance float64
	if len(ranks) > 0 {
		var sum float64
		for _, r := range ranks {
			sum += float64(r)
		}
		mean := sum / float64(len(ranks))
		for _, r := range ranks {
			variance += math.Pow(float64(r)-mean, 2)
		}
		variance /= float64(len(ranks))
	}

	// stability_score = mention_rate × (1 - normalized_rank_variance)
	// Normalizing variance: 0 to ~10 (rough estimate for rank variations 1-20)
	normalizedVar := variance / 20.0
	if normalizedVar > 1 {
		normalizedVar = 1
	}

	stability := mentionRate * (1 - normalizedVar)

	return db.StabilityScore{
		RunID:          samples[0].RunID,
		PromptID:       samples[0].PromptID,
		Provider:       samples[0].Provider,
		Brand:          samples[0].Brand,
		SampleCount:    len(samples),
		MentionRate:    mentionRate,
		RankVariance:   variance,
		StabilityScore: stability,
	}
}

type VisibilityScore struct {
	Brand            string
	RunID            int64
	Score            float64 // 0-100 composite
	MentionRate      float64
	FirstRecRate     float64
	SentimentScore   float64 // -1 to 1
	CitationScore    float64
	StabilityScore   float64
	ProviderCoverage float64
}

// CalcVisibilityScore calculates the composite visibility score for a brand.
func CalcVisibilityScore(brand string, runID int64, results []db.Result, stabilities []db.StabilityScore) VisibilityScore {
	if len(results) == 0 {
		return VisibilityScore{Brand: brand, RunID: runID}
	}

	var mentions, firstRecs, cited int
	var sentimentSum float64
	providers := make(map[string]bool)
	mentionedProviders := make(map[string]bool)

	for _, r := range results {
		providers[r.Provider] = true
		if r.BrandMentioned {
			mentions++
			mentionedProviders[r.Provider] = true
			if r.RecommendationRank != nil && *r.RecommendationRank == 1 {
				firstRecs++
			}
			switch r.Sentiment {
			case "positive":
				sentimentSum += 1
			case "negative":
				sentimentSum -= 1
			}
			if len(r.CitedURLs) > 0 {
				cited++
			}
		}
	}

	mentionRate := float64(mentions) / float64(len(results)) * 100
	firstRecRate := float64(firstRecs) / float64(len(results)) * 100
	citationScore := float64(cited) / float64(len(results)) * 100
	sentimentScore := 0.0
	if mentions > 0 {
		sentimentScore = sentimentSum / float64(mentions)
	}

	providerCoverage := float64(len(mentionedProviders)) / float64(len(providers)) * 100

	var avgStability float64
	if len(stabilities) > 0 {
		var sum float64
		for _, s := range stabilities {
			sum += s.StabilityScore
		}
		avgStability = sum / float64(len(stabilities))
	}

	// Composite formula from tasks.md:
	// Score = (MentionRate × 0.35)
	//       + (FirstRecRate × 0.25)
	//       + ((SentimentScore + 1) / 2 × 100 × 0.15)
	//       + (CitationScore × 0.10)
	//       + (StabilityScore × 0.10)
	//       + (ProviderCoverage × 0.05)
	score := (mentionRate * 0.35) +
		(firstRecRate * 0.25) +
		((sentimentScore+1)/2 * 100 * 0.15) +
		(citationScore * 0.10) +
		(avgStability * 0.10) +
		(providerCoverage * 0.05)

	return VisibilityScore{
		Brand:            brand,
		RunID:            runID,
		Score:            score,
		MentionRate:      mentionRate,
		FirstRecRate:     firstRecRate,
		SentimentScore:   sentimentScore,
		CitationScore:    citationScore,
		StabilityScore:   avgStability,
		ProviderCoverage: providerCoverage,
	}
}
