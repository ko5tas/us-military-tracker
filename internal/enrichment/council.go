package enrichment

import (
	"context"
	"sync"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/enrichment/providers"
)

// CouncilResponse holds the result of a single AI provider's completion.
type CouncilResponse struct {
	Provider string
	Response string
	Err      error
	Latency  time.Duration
}

// RunCouncil dispatches the same prompt to all AI providers in parallel and
// collects their responses. Every member is called regardless of whether
// others succeed or fail.
func RunCouncil(ctx context.Context, members []providers.Completer, systemPrompt, userPrompt string) []CouncilResponse {
	if len(members) == 0 {
		return []CouncilResponse{}
	}

	results := make([]CouncilResponse, len(members))
	var wg sync.WaitGroup

	for i, member := range members {
		wg.Add(1)
		go func(idx int, m providers.Completer) {
			defer wg.Done()

			start := time.Now()
			resp, err := m.Complete(ctx, systemPrompt, userPrompt)
			elapsed := time.Since(start)

			results[idx] = CouncilResponse{
				Provider: m.Name(),
				Response: resp,
				Err:      err,
				Latency:  elapsed,
			}
		}(i, member)
	}

	wg.Wait()
	return results
}

// SuccessfulResponses filters council results to only those where Err is nil
// and Response is non-empty.
func SuccessfulResponses(results []CouncilResponse) []CouncilResponse {
	var out []CouncilResponse
	for _, r := range results {
		if r.Err == nil && r.Response != "" {
			out = append(out, r)
		}
	}
	return out
}
