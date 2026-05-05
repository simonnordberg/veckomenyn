package llm

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func ephemeralOnText(b anthropic.ContentBlockParamUnion) bool {
	return b.OfText != nil && b.OfText.CacheControl.Type == "ephemeral"
}

func ephemeralOnToolResult(b anthropic.ContentBlockParamUnion) bool {
	return b.OfToolResult != nil && b.OfToolResult.CacheControl.Type == "ephemeral"
}

func TestSetCacheBreakpoints_EmptyMessages(t *testing.T) {
	system := []anthropic.TextBlockParam{{Text: "you are helpful"}}
	var msgs []anthropic.MessageParam
	setCacheBreakpoints(system, msgs)
	if system[0].CacheControl.Type != "ephemeral" {
		t.Fatal("system block should have cache_control=ephemeral")
	}
}

func TestSetCacheBreakpoints_LastBlockIsUserText(t *testing.T) {
	system := []anthropic.TextBlockParam{{Text: "sys"}}
	msgs := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("hello")),
	}
	setCacheBreakpoints(system, msgs)

	last := msgs[0].Content[0]
	if !ephemeralOnText(last) {
		t.Fatal("expected cache_control=ephemeral on the user text block")
	}
}

func TestSetCacheBreakpoints_LastBlockIsToolResult(t *testing.T) {
	system := []anthropic.TextBlockParam{{Text: "sys"}}
	msgs := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("buy milk")),
		anthropic.NewUserMessage(
			anthropic.NewToolResultBlock("tool-1", "added", false),
			anthropic.NewToolResultBlock("tool-2", "done", false),
		),
	}
	setCacheBreakpoints(system, msgs)

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

func TestSetCacheBreakpoints_IdempotentOnRepeatCalls(t *testing.T) {
	system := []anthropic.TextBlockParam{{Text: "sys"}}
	msgs := []anthropic.MessageParam{
		anthropic.NewUserMessage(
			anthropic.NewToolResultBlock("tool-1", "done", false),
		),
	}
	setCacheBreakpoints(system, msgs)
	setCacheBreakpoints(system, msgs)

	last := msgs[0].Content[0]
	if !ephemeralOnToolResult(last) {
		t.Fatal("expected cache_control=ephemeral after repeated calls")
	}
}

func TestSetCacheBreakpoints_ClearsPreviousBreakpointsOnAdvance(t *testing.T) {
	system := []anthropic.TextBlockParam{{Text: "sys"}}
	msgs := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("hello")),
	}
	setCacheBreakpoints(system, msgs)

	msgs = append(msgs,
		anthropic.NewUserMessage(anthropic.NewToolResultBlock("tool-1", "done", false)),
	)
	setCacheBreakpoints(system, msgs)

	if ephemeralOnText(msgs[0].Content[0]) {
		t.Error("previous breakpoint on the user text block should have been cleared")
	}
	if !ephemeralOnToolResult(msgs[1].Content[0]) {
		t.Fatal("expected cache_control=ephemeral on the new final tool_result block")
	}
}

func TestSetCacheBreakpoints_NeverExceedsOneMessageLevelBreakpoint(t *testing.T) {
	system := []anthropic.TextBlockParam{{Text: "sys"}}
	var msgs []anthropic.MessageParam
	for i := 0; i < 10; i++ {
		msgs = append(msgs,
			anthropic.NewUserMessage(anthropic.NewToolResultBlock("tool", "ok", false)),
		)
		setCacheBreakpoints(system, msgs)
	}
	count := 0
	for _, m := range msgs {
		for _, b := range m.Content {
			if ephemeralOnText(b) || ephemeralOnToolResult(b) {
				count++
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 message-level breakpoint after 10 iterations, got %d", count)
	}
}
