package llm

import (
	"encoding/json"
	"testing"
)

func TestNewUserMessage(t *testing.T) {
	m := NewUserMessage(TextBlock("hello"))
	if m.Role != RoleUser {
		t.Fatalf("role = %q, want %q", m.Role, RoleUser)
	}
	if len(m.Content) != 1 {
		t.Fatalf("len(content) = %d, want 1", len(m.Content))
	}
	if m.Content[0].Type != "text" {
		t.Fatalf("type = %q, want text", m.Content[0].Type)
	}
	if m.Content[0].Text != "hello" {
		t.Fatalf("text = %q, want hello", m.Content[0].Text)
	}
}

func TestNewAssistantMessage(t *testing.T) {
	m := NewAssistantMessage(TextBlock("reply"))
	if m.Role != RoleAssistant {
		t.Fatalf("role = %q, want %q", m.Role, RoleAssistant)
	}
	if m.Content[0].Text != "reply" {
		t.Fatalf("text = %q, want reply", m.Content[0].Text)
	}
}

func TestNewUserMessage_ToolResult(t *testing.T) {
	m := NewUserMessage(ToolResultBlock("call-1", "done", false))
	b := m.Content[0]
	if b.Type != "tool_result" {
		t.Fatalf("type = %q, want tool_result", b.Type)
	}
	if b.ToolID != "call-1" {
		t.Fatalf("tool_id = %q, want call-1", b.ToolID)
	}
	if b.Text != "done" {
		t.Fatalf("text = %q, want done", b.Text)
	}
	if b.IsError {
		t.Fatal("expected IsError=false")
	}
}

func TestNewUserMessage_ToolResult_Error(t *testing.T) {
	m := NewUserMessage(ToolResultBlock("call-2", "failed", true))
	b := m.Content[0]
	if !b.IsError {
		t.Fatal("expected IsError=true")
	}
}

func TestToolUseBlock(t *testing.T) {
	input := json.RawMessage(`{"query":"milk"}`)
	b := ToolUseBlock("call-3", "willys_search", input)
	if b.Type != "tool_use" {
		t.Fatalf("type = %q, want tool_use", b.Type)
	}
	if b.ToolID != "call-3" {
		t.Fatalf("tool_id = %q, want call-3", b.ToolID)
	}
	if b.ToolName != "willys_search" {
		t.Fatalf("tool_name = %q, want willys_search", b.ToolName)
	}
	if string(b.ToolInput) != `{"query":"milk"}` {
		t.Fatalf("tool_input = %s, want {\"query\":\"milk\"}", b.ToolInput)
	}
}

func TestToolDef(t *testing.T) {
	d := ToolDef{
		Name:        "read_preferences",
		Description: "Returns preferences",
		InputSchema: ToolSchema{
			Properties: map[string]any{
				"weeks": map[string]any{"type": "integer"},
			},
			Required: []string{"weeks"},
		},
	}
	if d.Name != "read_preferences" {
		t.Fatalf("name = %q", d.Name)
	}
	if len(d.InputSchema.Required) != 1 || d.InputSchema.Required[0] != "weeks" {
		t.Fatalf("required = %v", d.InputSchema.Required)
	}
}

func TestUsage_ZeroValue(t *testing.T) {
	var u Usage
	if u.InputTokens != 0 || u.OutputTokens != 0 {
		t.Fatal("zero-value usage should have all zeros")
	}
	if u.CacheCreationInputTokens != 0 || u.CacheReadInputTokens != 0 {
		t.Fatal("zero-value cache fields should be zero")
	}
}

func TestStopReasonConstants(t *testing.T) {
	if StopReasonEndTurn == StopReasonToolUse {
		t.Fatal("stop reason constants must differ")
	}
	if StopReasonEndTurn == "" {
		t.Fatal("StopReasonEndTurn must not be empty")
	}
	if StopReasonToolUse == "" {
		t.Fatal("StopReasonToolUse must not be empty")
	}
}

func TestStreamEventKindConstants(t *testing.T) {
	kinds := []StreamEventKind{
		EventTextDelta,
		EventToolCallStart,
		EventToolCallDelta,
		EventToolCallComplete,
	}
	seen := map[StreamEventKind]bool{}
	for _, k := range kinds {
		if k == "" {
			t.Fatal("event kind must not be empty")
		}
		if seen[k] {
			t.Fatalf("duplicate event kind: %q", k)
		}
		seen[k] = true
	}
}

func TestMessage_MultipleBlocks(t *testing.T) {
	m := NewUserMessage(
		ToolResultBlock("call-1", "ok", false),
		ToolResultBlock("call-2", "also ok", false),
	)
	if len(m.Content) != 2 {
		t.Fatalf("len(content) = %d, want 2", len(m.Content))
	}
}
