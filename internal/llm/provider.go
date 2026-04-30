package llm

import "context"

type SystemBlock struct {
	Text string
}

type RunParams struct {
	Model     string
	MaxTokens int
	System    []SystemBlock
	Tools     []ToolDef
	Messages  []Message
}

type RunResult struct {
	StopReason       StopReason
	Usage            Usage
	AssistantMessage Message
}

type Provider interface {
	RunStream(ctx context.Context, params RunParams, emit func(StreamEvent)) (RunResult, error)
}
