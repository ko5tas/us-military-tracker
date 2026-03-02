package enrichment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/enrichment/providers"
)

// mockProvider implements providers.Completer for testing.
type mockProvider struct {
	name     string
	response string
	err      error
	delay    time.Duration
}

func (m *mockProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.response, m.err
}

func (m *mockProvider) Name() string {
	return m.name
}

// Compile-time check that mockProvider implements Completer.
var _ providers.Completer = (*mockProvider)(nil)

func TestRunCouncil(t *testing.T) {
	members := []providers.Completer{
		&mockProvider{name: "alpha", response: "response-alpha"},
		&mockProvider{name: "beta", response: "response-beta"},
		&mockProvider{name: "gamma", response: "response-gamma"},
	}

	results := RunCouncil(context.Background(), members, "system", "user")

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Collect results into a map for order-independent verification.
	byProvider := make(map[string]CouncilResponse)
	for _, r := range results {
		byProvider[r.Provider] = r
	}

	for _, name := range []string{"alpha", "beta", "gamma"} {
		r, ok := byProvider[name]
		if !ok {
			t.Errorf("missing result for provider %q", name)
			continue
		}
		if r.Err != nil {
			t.Errorf("provider %q returned unexpected error: %v", name, r.Err)
		}
		want := "response-" + name
		if r.Response != want {
			t.Errorf("provider %q: got response %q, want %q", name, r.Response, want)
		}
		if r.Latency <= 0 {
			t.Errorf("provider %q: expected positive latency, got %v", name, r.Latency)
		}
	}
}

func TestRunCouncilHandlesFailures(t *testing.T) {
	members := []providers.Completer{
		&mockProvider{name: "good", response: "ok"},
		&mockProvider{name: "bad", err: errors.New("provider down")},
	}

	results := RunCouncil(context.Background(), members, "system", "user")

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	var successes, failures int
	for _, r := range results {
		if r.Err != nil {
			failures++
			if r.Provider != "bad" {
				t.Errorf("expected failing provider to be %q, got %q", "bad", r.Provider)
			}
		} else {
			successes++
			if r.Provider != "good" {
				t.Errorf("expected succeeding provider to be %q, got %q", "good", r.Provider)
			}
			if r.Response != "ok" {
				t.Errorf("expected response %q, got %q", "ok", r.Response)
			}
		}
	}

	if successes != 1 {
		t.Errorf("expected 1 success, got %d", successes)
	}
	if failures != 1 {
		t.Errorf("expected 1 failure, got %d", failures)
	}
}

func TestSuccessfulResponses(t *testing.T) {
	input := []CouncilResponse{
		{Provider: "ok", Response: "data", Err: nil},
		{Provider: "fail", Response: "", Err: errors.New("oops")},
		{Provider: "empty", Response: "", Err: nil},
		{Provider: "also-ok", Response: "more data", Err: nil},
	}

	filtered := SuccessfulResponses(input)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 successful responses, got %d", len(filtered))
	}

	if filtered[0].Provider != "ok" {
		t.Errorf("first successful provider: got %q, want %q", filtered[0].Provider, "ok")
	}
	if filtered[1].Provider != "also-ok" {
		t.Errorf("second successful provider: got %q, want %q", filtered[1].Provider, "also-ok")
	}
}

func TestRunCouncilEmpty(t *testing.T) {
	results := RunCouncil(context.Background(), []providers.Completer{}, "system", "user")

	if len(results) != 0 {
		t.Fatalf("expected 0 results for empty members, got %d", len(results))
	}

	// Also test with nil slice.
	results = RunCouncil(context.Background(), nil, "system", "user")

	if len(results) != 0 {
		t.Fatalf("expected 0 results for nil members, got %d", len(results))
	}
}
