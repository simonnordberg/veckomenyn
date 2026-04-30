package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicProvider struct {
	client anthropic.Client
}

func NewAnthropic(apiKey string) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key required")
	}
	return &AnthropicProvider{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
	}, nil
}

func (a *AnthropicProvider) RunStream(ctx context.Context, params RunParams, emit func(StreamEvent)) (RunResult, error) {
	sdkSystem := toSDKSystemBlocks(params.System)
	sdkTools := toSDKTools(params.Tools)
	sdkMessages := toSDKMessages(params.Messages)

	setCacheBreakpoints(sdkSystem, sdkMessages)

	sdkParams := anthropic.MessageNewParams{
		Model:     anthropic.Model(params.Model),
		MaxTokens: int64(params.MaxTokens),
		System:    sdkSystem,
		Tools:     sdkTools,
		Messages:  sdkMessages,
	}

	stream := a.client.Messages.NewStreaming(ctx, sdkParams)
	resp := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()
		if err := resp.Accumulate(event); err != nil {
			return RunResult{}, fmt.Errorf("accumulate: %w", err)
		}
		switch ev := event.AsAny().(type) {
		case anthropic.ContentBlockStartEvent:
			if tu, ok := ev.ContentBlock.AsAny().(anthropic.ToolUseBlock); ok {
				emit(StreamEvent{Kind: EventToolCallStart, ToolName: tu.Name, ToolID: tu.ID})
			}
		case anthropic.ContentBlockDeltaEvent:
			if delta, ok := ev.Delta.AsAny().(anthropic.TextDelta); ok {
				if delta.Text != "" {
					emit(StreamEvent{Kind: EventTextDelta, Text: delta.Text})
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return RunResult{}, fmt.Errorf("stream: %w", err)
	}

	assistantMsg := fromSDKMessage(resp)

	var stopReason StopReason
	if resp.StopReason == anthropic.StopReasonToolUse {
		stopReason = StopReasonToolUse
	} else {
		stopReason = StopReasonEndTurn
	}

	return RunResult{
		StopReason:       stopReason,
		AssistantMessage: assistantMsg,
		Usage: Usage{
			InputTokens:              resp.Usage.InputTokens,
			OutputTokens:             resp.Usage.OutputTokens,
			CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
		},
	}, nil
}

func toSDKSystemBlocks(blocks []SystemBlock) []anthropic.TextBlockParam {
	out := make([]anthropic.TextBlockParam, len(blocks))
	for i, b := range blocks {
		out[i] = anthropic.TextBlockParam{Text: b.Text}
	}
	return out
}

func toSDKTools(defs []ToolDef) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, len(defs))
	for i, td := range defs {
		schema := anthropic.ToolInputSchemaParam{Properties: td.InputSchema.Properties}
		if len(td.InputSchema.Required) > 0 {
			schema.Required = td.InputSchema.Required
		}
		p := anthropic.ToolParam{
			Name:        td.Name,
			Description: anthropic.String(td.Description),
			InputSchema: schema,
		}
		out[i] = anthropic.ToolUnionParam{OfTool: &p}
	}
	return out
}

func toSDKMessages(msgs []Message) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, len(msgs))
	for i, m := range msgs {
		blocks := make([]anthropic.ContentBlockParamUnion, len(m.Content))
		for j, b := range m.Content {
			switch b.Type {
			case "text":
				blocks[j] = anthropic.NewTextBlock(b.Text)
			case "tool_result":
				blocks[j] = anthropic.NewToolResultBlock(b.ToolID, b.Text, b.IsError)
			case "tool_use":
				blocks[j] = anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    b.ToolID,
						Name:  b.ToolName,
						Input: json.RawMessage(b.ToolInput),
					},
				}
			}
		}
		switch m.Role {
		case RoleUser:
			out[i] = anthropic.NewUserMessage(blocks...)
		case RoleAssistant:
			out[i] = anthropic.NewAssistantMessage(blocks...)
		}
	}
	return out
}

func fromSDKMessage(resp anthropic.Message) Message {
	var blocks []ContentBlock
	for _, block := range resp.Content {
		switch v := block.AsAny().(type) {
		case anthropic.TextBlock:
			blocks = append(blocks, TextBlock(v.Text))
		case anthropic.ToolUseBlock:
			blocks = append(blocks, ToolUseBlock(v.ID, v.Name, json.RawMessage(v.JSON.Input.Raw())))
		}
	}
	return NewAssistantMessage(blocks...)
}

// setCacheBreakpoints applies Anthropic-specific cache control: ephemeral on
// the first system block and the last content block of the last message.
func setCacheBreakpoints(system []anthropic.TextBlockParam, msgs []anthropic.MessageParam) {
	if len(system) > 0 {
		system[0].CacheControl = anthropic.NewCacheControlEphemeralParam()
	}
	clearMessageCacheBreakpoints(msgs)
	if len(msgs) == 0 {
		return
	}
	lastMsg := &msgs[len(msgs)-1]
	if len(lastMsg.Content) == 0 {
		return
	}
	block := &lastMsg.Content[len(lastMsg.Content)-1]
	switch {
	case block.OfToolResult != nil:
		block.OfToolResult.CacheControl = anthropic.NewCacheControlEphemeralParam()
	case block.OfText != nil:
		block.OfText.CacheControl = anthropic.NewCacheControlEphemeralParam()
	}
}

func clearMessageCacheBreakpoints(msgs []anthropic.MessageParam) {
	var zero anthropic.CacheControlEphemeralParam
	for i := range msgs {
		content := msgs[i].Content
		for j := range content {
			b := &content[j]
			if b.OfToolResult != nil {
				b.OfToolResult.CacheControl = zero
			}
			if b.OfText != nil {
				b.OfText.CacheControl = zero
			}
		}
	}
}
