package llm

import "encoding/json"

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type StopReason string

const (
	StopReasonEndTurn StopReason = "end_turn"
	StopReasonToolUse StopReason = "tool_use"
)

type StreamEventKind string

const (
	EventTextDelta        StreamEventKind = "text_delta"
	EventToolCallStart    StreamEventKind = "tool_call_start"
	EventToolCallDelta    StreamEventKind = "tool_call_delta"
	EventToolCallComplete StreamEventKind = "tool_call_complete"
)

type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ToolID    string          `json:"tool_id,omitempty"`
	ToolName  string          `json:"tool_name,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type Message struct {
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content"`
}

type ToolSchema struct {
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

type ToolDef struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema ToolSchema `json:"input_schema"`
}

type Usage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens,omitempty"`
}

type StreamEvent struct {
	Kind      StreamEventKind
	Text      string
	ToolName  string
	ToolID    string
	ToolInput string
}

func TextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

func ToolResultBlock(toolID, text string, isError bool) ContentBlock {
	return ContentBlock{
		Type:    "tool_result",
		ToolID:  toolID,
		Text:    text,
		IsError: isError,
	}
}

func ToolUseBlock(toolID, toolName string, input json.RawMessage) ContentBlock {
	return ContentBlock{
		Type:      "tool_use",
		ToolID:    toolID,
		ToolName:  toolName,
		ToolInput: input,
	}
}

func NewUserMessage(blocks ...ContentBlock) Message {
	return Message{Role: RoleUser, Content: blocks}
}

func NewAssistantMessage(blocks ...ContentBlock) Message {
	return Message{Role: RoleAssistant, Content: blocks}
}
