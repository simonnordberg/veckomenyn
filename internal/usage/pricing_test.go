package usage

import (
	"math"
	"testing"

	"github.com/simonnordberg/veckomenyn/internal/llm"
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestCost_SonnetInputOnly_OneMillionTokensIsThreeDollars(t *testing.T) {
	u := llm.Usage{InputTokens: 1_000_000}
	got := Cost("claude-sonnet-4-6", u)
	if !approxEqual(got, 3.0) {
		t.Errorf("1M sonnet input tokens: got %v want 3.0", got)
	}
}

func TestCost_SonnetOutputOnly_OneMillionTokensIsFifteenDollars(t *testing.T) {
	u := llm.Usage{OutputTokens: 1_000_000}
	got := Cost("claude-sonnet-4-6", u)
	if !approxEqual(got, 15.0) {
		t.Errorf("1M sonnet output tokens: got %v want 15.0", got)
	}
}

func TestCost_OpusInputOnly_OneMillionTokensIsFifteenDollars(t *testing.T) {
	u := llm.Usage{InputTokens: 1_000_000}
	got := Cost("claude-opus-4-7", u)
	if !approxEqual(got, 15.0) {
		t.Errorf("1M opus input tokens: got %v want 15.0", got)
	}
}

func TestCost_HaikuInputOnly_OneMillionTokensIsOneDollar(t *testing.T) {
	u := llm.Usage{InputTokens: 1_000_000}
	got := Cost("claude-haiku-4-5", u)
	if !approxEqual(got, 1.0) {
		t.Errorf("1M haiku input tokens: got %v want 1.0", got)
	}
}

func TestCost_AllFourBucketsSum(t *testing.T) {
	// Sonnet: input $3, cache write 5m $3.75, cache read $0.30, output $15
	// 1000 of each: (3 + 3.75 + 0.30 + 15) / 1000 = 0.02205
	u := llm.Usage{
		InputTokens:              1000,
		CacheCreationInputTokens: 1000,
		CacheReadInputTokens:     1000,
		OutputTokens:             1000,
	}
	got := Cost("claude-sonnet-4-6", u)
	if !approxEqual(got, 0.02205) {
		t.Errorf("all four buckets: got %v want 0.02205", got)
	}
}

func TestCost_UnknownModel_ReturnsZero(t *testing.T) {
	u := llm.Usage{InputTokens: 1_000_000}
	if got := Cost("claude-something-unreleased", u); got != 0 {
		t.Errorf("unknown model: got %v want 0", got)
	}
}

func TestKnownModel(t *testing.T) {
	if !KnownModel("claude-sonnet-4-6") {
		t.Error("claude-sonnet-4-6 should be known")
	}
	if !KnownModel("claude-opus-4-7") {
		t.Error("claude-opus-4-7 should be known")
	}
	if !KnownModel("claude-haiku-4-5") {
		t.Error("claude-haiku-4-5 should be known")
	}
	if KnownModel("claude-phantom-9-9") {
		t.Error("unreleased model should be unknown")
	}
}
