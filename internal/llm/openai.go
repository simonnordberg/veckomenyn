package llm

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

type OpenAIProvider struct {
	client openai.Client
	model  string
}

func NewOpenAI(baseURL, model, apiKey string) (*OpenAIProvider, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("base URL required")
	}
	if model == "" {
		return nil, fmt.Errorf("model required")
	}
	opts := []option.RequestOption{
		option.WithBaseURL(baseURL),
	}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	} else {
		opts = append(opts, option.WithAPIKey("not-required"))
	}
	return &OpenAIProvider{
		client: openai.NewClient(opts...),
		model:  model,
	}, nil
}

func (o *OpenAIProvider) RunStream(ctx context.Context, params RunParams, emit func(StreamEvent)) (RunResult, error) {
	msgs := toOpenAIMessages(params.System, params.Messages)
	tools := toOpenAITools(params.Tools)

	reqParams := openai.ChatCompletionNewParams{
		Model:    o.model,
		Messages: msgs,
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		},
	}
	if params.MaxTokens > 0 {
		reqParams.MaxCompletionTokens = openai.Int(int64(params.MaxTokens))
	}
	if len(tools) > 0 {
		reqParams.Tools = tools
	}

	stream := o.client.Chat.Completions.NewStreaming(ctx, reqParams)
	defer func() { _ = stream.Close() }()

	acc := openai.ChatCompletionAccumulator{}
	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				emit(StreamEvent{Kind: EventTextDelta, Text: choice.Delta.Content})
			}
			for _, tc := range choice.Delta.ToolCalls {
				if tc.ID != "" {
					emit(StreamEvent{
						Kind:     EventToolCallStart,
						ToolName: tc.Function.Name,
						ToolID:   tc.ID,
					})
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return RunResult{}, fmt.Errorf("stream: %w", err)
	}

	completion := acc.ChatCompletion
	assistantMsg := fromOpenAIMessage(completion)

	var stopReason StopReason
	if len(completion.Choices) > 0 && completion.Choices[0].FinishReason == "tool_calls" {
		stopReason = StopReasonToolUse
	} else {
		stopReason = StopReasonEndTurn
	}

	return RunResult{
		StopReason:       stopReason,
		AssistantMessage: assistantMsg,
		Usage: Usage{
			InputTokens:  completion.Usage.PromptTokens,
			OutputTokens: completion.Usage.CompletionTokens,
		},
	}, nil
}

func toOpenAIMessages(system []SystemBlock, msgs []Message) []openai.ChatCompletionMessageParamUnion {
	var out []openai.ChatCompletionMessageParamUnion

	if len(system) > 0 {
		var text string
		for i, s := range system {
			if i > 0 {
				text += "\n\n"
			}
			text += s.Text
		}
		out = append(out, openai.SystemMessage(text))
	}

	for _, m := range msgs {
		switch m.Role {
		case RoleUser:
			for _, b := range m.Content {
				switch b.Type {
				case "text":
					out = append(out, openai.UserMessage(b.Text))
				case "tool_result":
					out = append(out, openai.ToolMessage(b.ToolID, b.Text))
				}
			}
		case RoleAssistant:
			var toolCalls []openai.ChatCompletionMessageToolCallUnionParam
			var text string
			for _, b := range m.Content {
				switch b.Type {
				case "text":
					text += b.Text
				case "tool_use":
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnionParam{
						OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
							ID: b.ToolID,
							Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
								Name:      b.ToolName,
								Arguments: string(b.ToolInput),
							},
						},
					})
				}
			}
			msg := openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(text),
				},
			}
			if len(toolCalls) > 0 {
				msg.ToolCalls = toolCalls
			}
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &msg,
			})
		}
	}

	return out
}

func toOpenAITools(defs []ToolDef) []openai.ChatCompletionToolUnionParam {
	if len(defs) == 0 {
		return nil
	}
	out := make([]openai.ChatCompletionToolUnionParam, len(defs))
	for i, td := range defs {
		params := shared.FunctionParameters{
			"type":       "object",
			"properties": td.InputSchema.Properties,
		}
		if len(td.InputSchema.Required) > 0 {
			params["required"] = td.InputSchema.Required
		}
		out[i] = openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        td.Name,
					Description: openai.String(td.Description),
					Parameters:  params,
				},
			},
		}
	}
	return out
}

func fromOpenAIMessage(completion openai.ChatCompletion) Message {
	if len(completion.Choices) == 0 {
		return NewAssistantMessage()
	}
	choice := completion.Choices[0]
	var blocks []ContentBlock

	if choice.Message.Content != "" {
		blocks = append(blocks, TextBlock(choice.Message.Content))
	}
	for _, tc := range choice.Message.ToolCalls {
		blocks = append(blocks, ToolUseBlock(
			tc.ID,
			tc.Function.Name,
			json.RawMessage(tc.Function.Arguments),
		))
	}

	return NewAssistantMessage(blocks...)
}
