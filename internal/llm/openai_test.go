package llm

import "testing"

func TestNewOpenAI_EmptyBaseURL(t *testing.T) {
	_, err := NewOpenAI("", "test-model", "sk-key")
	if err == nil {
		t.Fatal("expected error for empty base URL")
	}
}

func TestNewOpenAI_EmptyModel(t *testing.T) {
	_, err := NewOpenAI("http://localhost:11434/v1", "", "sk-key")
	if err == nil {
		t.Fatal("expected error for empty model")
	}
}

func TestNewOpenAI_ValidConfig(t *testing.T) {
	p, err := NewOpenAI("http://localhost:11434/v1", "llama3.1:8b", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("provider should not be nil")
	}
}

func TestNewOpenAI_EmptyAPIKeyAllowed(t *testing.T) {
	p, err := NewOpenAI("http://localhost:11434/v1", "qwen3.6", "")
	if err != nil {
		t.Fatalf("empty API key should be allowed for local backends: %v", err)
	}
	if p == nil {
		t.Fatal("provider should not be nil")
	}
}

func TestOpenAIProvider_ImplementsProvider(t *testing.T) {
	p, _ := NewOpenAI("http://localhost:11434/v1", "test", "")
	var _ Provider = p
}

func TestToOpenAIMessages_SystemConcatenation(t *testing.T) {
	system := []SystemBlock{
		{Text: "You are a meal planner."},
		{Text: "Respond in Swedish."},
	}
	msgs := toOpenAIMessages(system, nil)
	if len(msgs) != 1 {
		t.Fatalf("len = %d, want 1 (system message)", len(msgs))
	}
	if msgs[0].OfSystem == nil {
		t.Fatal("expected system message")
	}
}

func TestToOpenAIMessages_UserAndAssistant(t *testing.T) {
	history := []Message{
		NewUserMessage(TextBlock("plan my week")),
		NewAssistantMessage(TextBlock("sure")),
	}
	msgs := toOpenAIMessages(nil, history)
	if len(msgs) != 2 {
		t.Fatalf("len = %d, want 2", len(msgs))
	}
	if msgs[0].OfUser == nil {
		t.Fatal("first should be user")
	}
	if msgs[1].OfAssistant == nil {
		t.Fatal("second should be assistant")
	}
}

func TestToOpenAIMessages_ToolResults(t *testing.T) {
	history := []Message{
		NewUserMessage(
			ToolResultBlock("call-1", "done", false),
			ToolResultBlock("call-2", "also done", false),
		),
	}
	msgs := toOpenAIMessages(nil, history)
	if len(msgs) != 2 {
		t.Fatalf("len = %d, want 2 (one per tool result)", len(msgs))
	}
	if msgs[0].OfTool == nil || msgs[1].OfTool == nil {
		t.Fatal("both should be tool messages")
	}
}

func TestToOpenAITools(t *testing.T) {
	defs := []ToolDef{
		{
			Name:        "search",
			Description: "Search products",
			InputSchema: ToolSchema{
				Properties: map[string]any{
					"query": map[string]any{"type": "string"},
				},
				Required: []string{"query"},
			},
		},
	}
	tools := toOpenAITools(defs)
	if len(tools) != 1 {
		t.Fatalf("len = %d, want 1", len(tools))
	}
	if tools[0].OfFunction == nil {
		t.Fatal("expected OfFunction to be set")
	}
	if tools[0].OfFunction.Function.Name != "search" {
		t.Fatalf("name = %q, want search", tools[0].OfFunction.Function.Name)
	}
}

func TestToOpenAITools_Empty(t *testing.T) {
	tools := toOpenAITools(nil)
	if tools != nil {
		t.Fatal("nil input should return nil")
	}
}
