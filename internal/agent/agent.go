// Package agent runs the meal-planning Claude agent. It owns the system prompt,
// the tool registry, and the multi-turn loop that handles tool_use.
package agent

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/simonnordberg/veckomenyn/internal/providers"
	"github.com/simonnordberg/veckomenyn/internal/shopping"
	"github.com/simonnordberg/veckomenyn/internal/store"
)

//go:embed system.md
var systemPrompt string

const (
	defaultMaxIters = 60 // cart-build flows can take many tool iterations
	defaultMaxTok   = 16000
)

type Config struct {
	MaxIters int // safety cap on tool-use loop iterations
}

type Agent struct {
	cfg       Config
	db        *pgxpool.Pool
	providers *providers.Store
	log       *slog.Logger
	tools     []Tool
	toolDefs  []anthropic.ToolUnionParam
}

func New(cfg Config, db *pgxpool.Pool, provStore *providers.Store, shop shopping.Provider, log *slog.Logger) *Agent {
	if cfg.MaxIters == 0 {
		cfg.MaxIters = defaultMaxIters
	}
	a := &Agent{
		cfg:       cfg,
		db:        db,
		providers: provStore,
		log:       log,
	}
	a.tools = registerTools(db, shop, log)
	a.toolDefs = make([]anthropic.ToolUnionParam, 0, len(a.tools))
	for _, t := range a.tools {
		def := t.Def()
		a.toolDefs = append(a.toolDefs, anthropic.ToolUnionParam{OfTool: &def})
	}
	return a
}

// resolveClient builds an Anthropic client from the current DB provider row.
// Called once per Run so API-key changes in Settings take effect on the
// next turn without a restart.
func (a *Agent) resolveClient(ctx context.Context) (anthropic.Client, error) {
	key := a.providers.AnthropicAPIKey(ctx)
	if key == "" {
		return anthropic.Client{}, fmt.Errorf("anthropic API key not configured (set in Settings → Integrations)")
	}
	return anthropic.NewClient(option.WithAPIKey(key)), nil
}

// Event is a coarse-grained update for callers (chat handlers, tests).
// Keep this stable; the SSE stream will serialize these directly.
type Event struct {
	Type    string `json:"type"` // "text", "tool_call", "tool_result", "error", "done"
	Text    string `json:"text,omitempty"`
	Tool    string `json:"tool,omitempty"`
	ToolID  string `json:"tool_id,omitempty"`
	Input   string `json:"input,omitempty"`  // raw JSON
	Result  string `json:"result,omitempty"` // text result or error message
	IsError bool   `json:"is_error,omitempty"`
}

// Run executes one user turn: sends the user message, processes any tool calls,
// and returns when the model stops asking for tools. Events are emitted for
// text chunks and tool activity so the caller can stream them to a UI.
//
// Message history is persisted as we go. history is loaded from the DB by the
// caller and passed in; Run returns the updated history (new user + assistant
// messages appended) for the caller to persist.
func (a *Agent) Run(
	ctx context.Context,
	history []anthropic.MessageParam,
	userMessage string,
	emit func(Event),
) ([]anthropic.MessageParam, error) {
	msgs := append(history, anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)))

	client, err := a.resolveClient(ctx)
	if err != nil {
		emit(Event{Type: "error", Result: err.Error(), IsError: true})
		return msgs, err
	}

	model := anthropic.Model(a.providers.AnthropicModel(ctx))

	// Load the current language preference once per turn. It goes into a
	// second system block so the cacheable first block (prompt + tools) stays
	// stable as the setting flips between turns.
	langBlock := languageBlock(ctx, a.db, a.log)
	planBlock := currentPlanBlock(ctx, a.db, a.log)

	for i := 0; i < a.cfg.MaxIters; i++ {
		systemBlocks := []anthropic.TextBlockParam{{
			Text:         systemPrompt,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}}
		if langBlock.Text != "" {
			systemBlocks = append(systemBlocks, langBlock)
		}
		if planBlock.Text != "" {
			systemBlocks = append(systemBlocks, planBlock)
		}
		params := anthropic.MessageNewParams{
			Model:     model,
			MaxTokens: defaultMaxTok,
			System:    systemBlocks,
			Tools:     a.toolDefs,
			Messages:  msgs,
		}

		// Stream the LLM response so text shows token-by-token in the UI.
		// Tool-use still happens per iteration; we accumulate the full message,
		// then run any tool_use blocks and loop.
		stream := client.Messages.NewStreaming(ctx, params)
		resp := anthropic.Message{}

		// Track which text block index we're currently streaming so we can
		// emit a fresh text event when the model opens a new one (rare but
		// possible after a tool_use).
		toolUseStarted := map[int]bool{}
		for stream.Next() {
			event := stream.Current()
			if err := resp.Accumulate(event); err != nil {
				emit(Event{Type: "error", Result: "accumulate: " + err.Error(), IsError: true})
				return msgs, fmt.Errorf("accumulate: %w", err)
			}
			switch ev := event.AsAny().(type) {
			case anthropic.ContentBlockStartEvent:
				if tu, ok := ev.ContentBlock.AsAny().(anthropic.ToolUseBlock); ok {
					toolUseStarted[int(ev.Index)] = true
					// Input isn't populated yet on start; we emit tool_call
					// when the block stops (see below). Just note the intent.
					emit(Event{Type: "tool_call_started", Tool: tu.Name, ToolID: tu.ID})
				}
			case anthropic.ContentBlockDeltaEvent:
				if delta, ok := ev.Delta.AsAny().(anthropic.TextDelta); ok {
					if delta.Text != "" {
						emit(Event{Type: "text", Text: delta.Text})
					}
				}
			}
		}
		if err := stream.Err(); err != nil {
			emit(Event{Type: "error", Result: err.Error(), IsError: true})
			return msgs, fmt.Errorf("stream: %w", err)
		}

		msgs = append(msgs, resp.ToParam())

		// Tool calls are resolved from the accumulated message so we get the
		// final parsed input strings rather than reassembling deltas ourselves.
		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, block := range resp.Content {
			if v, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				input := v.JSON.Input.Raw()
				emit(Event{
					Type:   "tool_call",
					Tool:   v.Name,
					ToolID: v.ID,
					Input:  input,
				})
				result, toolErr := a.callTool(ctx, v.Name, []byte(input))
				isErr := toolErr != nil
				resultStr := result
				if isErr {
					resultStr = toolErr.Error()
				}
				emit(Event{
					Type:    "tool_result",
					Tool:    v.Name,
					ToolID:  v.ID,
					Result:  resultStr,
					IsError: isErr,
				})
				toolResults = append(toolResults, anthropic.NewToolResultBlock(v.ID, resultStr, isErr))
			}
		}

		if resp.StopReason != anthropic.StopReasonToolUse {
			emit(Event{Type: "done"})
			return msgs, nil
		}
		if len(toolResults) == 0 {
			emit(Event{Type: "error", Result: "model requested tool use but emitted no tool blocks", IsError: true})
			return msgs, errors.New("empty tool batch")
		}

		msgs = append(msgs, anthropic.NewUserMessage(toolResults...))
	}

	emit(Event{Type: "error", Result: "max iterations reached", IsError: true})
	return msgs, fmt.Errorf("max iterations (%d) reached", a.cfg.MaxIters)
}

// languageBlock returns a system-prompt fragment directing the model to write
// generated content in the configured language. Falls back to no directive
// (i.e. English) when settings can't be read.
func languageBlock(ctx context.Context, db *pgxpool.Pool, log *slog.Logger) anthropic.TextBlockParam {
	s, err := store.GetHouseholdSettings(ctx, db)
	if err != nil {
		log.Warn("language block: falling back to no directive", "err", err)
		return anthropic.TextBlockParam{}
	}
	lang := s.Language
	if lang == "" {
		lang = "sv"
	}
	var text string
	switch lang {
	case "sv":
		text = `<language>sv</language>
Write all user-facing output in Swedish: chat replies, recipe_md bodies, dish names, notes, and summaries. Use Swedish cooking terminology (sjud, fräs, stek, etc.). Keep tool call arguments as-is (dates, IDs, statuses, category slugs remain English/ISO). Do not translate existing recipe content; only new content you generate from this point.`
	case "en":
		text = `<language>en</language>
Write all user-facing output in English. Do not translate existing content; leave it in whatever language it was recorded.`
	default:
		return anthropic.TextBlockParam{}
	}
	return anthropic.TextBlockParam{Text: text}
}

// currentPlanBlock injects a short fact sheet about the plan this chat is
// bound to. The model uses it to default all tool calls to that plan without
// having to ask the user which plan they mean. Empty when no plan is in
// scope (e.g. a general /chat request with no plan context).
func currentPlanBlock(ctx context.Context, db *pgxpool.Pool, log *slog.Logger) anthropic.TextBlockParam {
	id := WeekIDFrom(ctx)
	if id == 0 {
		return anthropic.TextBlockParam{}
	}
	var iso, start, end, status string
	var dinnerCount int
	err := db.QueryRow(ctx, `
		SELECT w.iso_week, w.start_date::text, w.end_date::text, w.status,
		       COALESCE((SELECT COUNT(*) FROM week_dinners wd WHERE wd.week_id = w.id), 0)
		FROM weeks w WHERE w.id = $1`, id).Scan(&iso, &start, &end, &status, &dinnerCount)
	if err != nil {
		log.Warn("current plan block: lookup failed", "week_id", id, "err", err)
		return anthropic.TextBlockParam{}
	}
	text := fmt.Sprintf(`<current-plan>
  id: %d
  iso_week: %s
  dates: %s → %s
  status: %s
  dinners: %d
</current-plan>
This chat is tied to the plan above. Assume every user request refers to it. When calling week-scoped tools (add_dinner, update_week, update_dinner, delete_dinner, add_exception, record_retrospective, get_week), omit week_id or pass id=%d; any other value will be refused. If the user wants to edit a different plan, tell them to open it and try there.`, id, iso, start, end, status, dinnerCount, id)
	return anthropic.TextBlockParam{Text: text}
}

func (a *Agent) callTool(ctx context.Context, name string, input json.RawMessage) (string, error) {
	for _, t := range a.tools {
		if t.Name() == name {
			a.log.Debug("tool call", "tool", name, "input", string(input))
			out, err := t.Call(ctx, input)
			if err != nil {
				a.log.Warn("tool error", "tool", name, "err", err)
			}
			return out, err
		}
	}
	return "", fmt.Errorf("unknown tool: %s", name)
}
