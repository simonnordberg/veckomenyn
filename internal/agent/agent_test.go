package agent

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// ephemeral reports whether a cache_control breakpoint is set on a given block.
// The SDK's CacheControlEphemeralParam has Type="" when zero and "ephemeral"
// when constructed via NewCacheControlEphemeralParam, so the Type field is a
// reliable presence check.
func ephemeralOnText(b anthropic.ContentBlockParamUnion) bool {
	return b.OfText != nil && b.OfText.CacheControl.Type == "ephemeral"
}

func ephemeralOnToolResult(b anthropic.ContentBlockParamUnion) bool {
	return b.OfToolResult != nil && b.OfToolResult.CacheControl.Type == "ephemeral"
}

func TestSetRollingCacheBreakpoint_EmptyMessages(t *testing.T) {
	var msgs []anthropic.MessageParam
	setRollingCacheBreakpoint(msgs) // must not panic
}

func TestSetRollingCacheBreakpoint_LastBlockIsUserText(t *testing.T) {
	msgs := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("hello")),
	}
	setRollingCacheBreakpoint(msgs)

	last := msgs[0].Content[0]
	if !ephemeralOnText(last) {
		t.Fatalf("expected cache_control=ephemeral on the user text block")
	}
}

func TestSetRollingCacheBreakpoint_LastBlockIsToolResult(t *testing.T) {
	msgs := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("buy milk")),
		// assistant reply omitted; we only care about the final user message
		anthropic.NewUserMessage(
			anthropic.NewToolResultBlock("tool-1", "added", false),
			anthropic.NewToolResultBlock("tool-2", "done", false),
		),
	}
	setRollingCacheBreakpoint(msgs)

	first := msgs[0].Content[0]
	if ephemeralOnText(first) {
		t.Error("earlier user text must not be marked; only the final block should be")
	}

	lastMsg := msgs[len(msgs)-1]
	if ephemeralOnToolResult(lastMsg.Content[0]) {
		t.Error("non-final tool_result block must not be marked")
	}
	if !ephemeralOnToolResult(lastMsg.Content[len(lastMsg.Content)-1]) {
		t.Fatal("expected cache_control=ephemeral on the final tool_result block")
	}
}

func TestSetRollingCacheBreakpoint_IdempotentOnRepeatCalls(t *testing.T) {
	msgs := []anthropic.MessageParam{
		anthropic.NewUserMessage(
			anthropic.NewToolResultBlock("tool-1", "done", false),
		),
	}
	setRollingCacheBreakpoint(msgs)
	setRollingCacheBreakpoint(msgs)

	last := msgs[0].Content[0]
	if !ephemeralOnToolResult(last) {
		t.Fatal("expected cache_control=ephemeral after repeated calls")
	}
}
