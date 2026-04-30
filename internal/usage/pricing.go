// Package usage tracks LLM token spend per model call and stores it for
// per-session and per-week cost reporting.
//
// Pricing is hard-coded per model; keep it in sync with
// https://www.anthropic.com/pricing when a new model is added or prices
// change. Cost is computed at write time so past rows stay correct across
// price changes.
package usage

import "github.com/simonnordberg/veckomenyn/internal/llm"

// Prices are USD per million tokens for one model tier.
type Prices struct {
	Input        float64 // base input tokens
	CacheWrite5m float64 // cache_creation_input_tokens (5-minute TTL)
	CacheRead    float64 // cache_read_input_tokens
	Output       float64
}

// pricing covers the models routable via providers.AnthropicModel. Values
// follow Anthropic's standard tier (5-minute cache write at 1.25x input,
// cache read at 0.1x input).
var pricing = map[string]Prices{
	"claude-opus-4-7":   {Input: 15.00, CacheWrite5m: 18.75, CacheRead: 1.50, Output: 75.00},
	"claude-sonnet-4-6": {Input: 3.00, CacheWrite5m: 3.75, CacheRead: 0.30, Output: 15.00},
	"claude-haiku-4-5":  {Input: 1.00, CacheWrite5m: 1.25, CacheRead: 0.10, Output: 5.00},
}

// Cost returns the USD cost of a single Messages response at the given
// model's prices. Unknown models return 0; callers should check KnownModel
// and log when pricing is missing so gaps get noticed.
func Cost(model string, u llm.Usage) float64 {
	p, ok := pricing[model]
	if !ok {
		return 0
	}
	const perMillion = 1_000_000.0
	return float64(u.InputTokens)*p.Input/perMillion +
		float64(u.CacheCreationInputTokens)*p.CacheWrite5m/perMillion +
		float64(u.CacheReadInputTokens)*p.CacheRead/perMillion +
		float64(u.OutputTokens)*p.Output/perMillion
}

// KnownModel reports whether Cost has pricing data for model.
func KnownModel(model string) bool {
	_, ok := pricing[model]
	return ok
}
