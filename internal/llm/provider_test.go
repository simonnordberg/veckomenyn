package llm

import (
	"context"
	"testing"
)

// Compile-time check that the Provider interface is implementable.
type fakeProvider struct{}

func (f *fakeProvider) RunStream(ctx context.Context, params RunParams, emit func(StreamEvent)) (RunResult, error) {
	return RunResult{
		StopReason:       StopReasonEndTurn,
		Usage:            Usage{InputTokens: 10, OutputTokens: 5},
		AssistantMessage: NewAssistantMessage(TextBlock("hi")),
	}, nil
}

var _ Provider = (*fakeProvider)(nil)

func TestProvider_FakeReturnsResult(t *testing.T) {
	var p Provider = &fakeProvider{}
	result, err := p.RunStream(context.Background(), RunParams{
		Model:     "test-model",
		MaxTokens: 1000,
		System:    []SystemBlock{{Text: "you are helpful"}},
		Tools:     []ToolDef{{Name: "test_tool", Description: "a test"}},
		Messages:  []Message{NewUserMessage(TextBlock("hello"))},
	}, func(ev StreamEvent) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StopReason != StopReasonEndTurn {
		t.Fatalf("stop_reason = %q, want %q", result.StopReason, StopReasonEndTurn)
	}
	if result.Usage.InputTokens != 10 {
		t.Fatalf("input_tokens = %d, want 10", result.Usage.InputTokens)
	}
	if result.AssistantMessage.Role != RoleAssistant {
		t.Fatalf("role = %q, want assistant", result.AssistantMessage.Role)
	}
}

func TestRunParams_EmptyToolsAndSystem(t *testing.T) {
	p := RunParams{
		Model:     "m",
		MaxTokens: 100,
		Messages:  []Message{NewUserMessage(TextBlock("hi"))},
	}
	if p.System != nil {
		t.Fatal("nil system should be allowed")
	}
	if p.Tools != nil {
		t.Fatal("nil tools should be allowed")
	}
}
