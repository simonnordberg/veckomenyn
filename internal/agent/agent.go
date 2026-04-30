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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/simonnordberg/veckomenyn/internal/llm"
	"github.com/simonnordberg/veckomenyn/internal/providers"
	"github.com/simonnordberg/veckomenyn/internal/shopping"
	"github.com/simonnordberg/veckomenyn/internal/store"
	"github.com/simonnordberg/veckomenyn/internal/usage"
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
	tools    []Tool
	toolDefs []llm.ToolDef
	recorder *usage.Recorder
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
		recorder:  usage.NewRecorder(db, log),
	}
	a.tools = registerTools(db, shop, log)
	a.toolDefs = make([]llm.ToolDef, 0, len(a.tools))
	for _, t := range a.tools {
		a.toolDefs = append(a.toolDefs, t.Def())
	}
	return a
}

// resolveProvider reads the llm_provider setting and builds the matching
// provider from saved config. Called once per Run so changes take effect
// on the next turn without a restart.
func (a *Agent) resolveProvider(ctx context.Context) (llm.Provider, string, error) {
	s, err := store.GetHouseholdSettings(ctx, a.db)
	if err != nil {
		return nil, "", fmt.Errorf("read settings: %w", err)
	}

	switch providers.Kind(s.LLMProvider) {
	case providers.KindAnthropic:
		key := a.providers.AnthropicAPIKey(ctx)
		if key == "" {
			return nil, "", fmt.Errorf("Anthropic API key not configured (set in Settings -> Integrations)")
		}
		p, err := llm.NewAnthropic(key)
		if err != nil {
			return nil, "", err
		}
		return p, a.providers.AnthropicModel(ctx), nil

	case providers.KindOpenAI:
		cfg, ok := a.providers.OpenAIConfig(ctx)
		if !ok {
			return nil, "", fmt.Errorf("OpenAI API key not configured (set in Settings -> Integrations)")
		}
		p, err := llm.NewOpenAI(cfg.BaseURL, cfg.Model, cfg.APIKey)
		if err != nil {
			return nil, "", err
		}
		return p, cfg.Model, nil

	case providers.KindOpenAICompat:
		cfg, ok := a.providers.OpenAICompatConfig(ctx)
		if !ok {
			return nil, "", fmt.Errorf("OpenAI-compatible provider not configured (set in Settings -> Integrations)")
		}
		p, err := llm.NewOpenAI(cfg.BaseURL, cfg.Model, cfg.APIKey)
		if err != nil {
			return nil, "", err
		}
		return p, cfg.Model, nil

	default:
		return nil, "", fmt.Errorf("unknown LLM provider %q (set in Settings -> Integrations)", s.LLMProvider)
	}
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
	history []llm.Message,
	userMessage string,
	emit func(Event),
) ([]llm.Message, error) {
	msgs := append(history, llm.NewUserMessage(llm.TextBlock(userMessage)))

	provider, model, err := a.resolveProvider(ctx)
	if err != nil {
		emit(Event{Type: "error", Result: err.Error(), IsError: true})
		return msgs, err
	}

	system := a.buildSystemBlocks(ctx)

	for i := 0; i < a.cfg.MaxIters; i++ {
		params := llm.RunParams{
			Model:     model,
			MaxTokens: defaultMaxTok,
			System:    system,
			Tools:     a.toolDefs,
			Messages:  msgs,
		}

		result, err := provider.RunStream(ctx, params, func(ev llm.StreamEvent) {
			switch ev.Kind {
			case llm.EventTextDelta:
				emit(Event{Type: "text", Text: ev.Text})
			case llm.EventToolCallStart:
				emit(Event{Type: "tool_call_started", Tool: ev.ToolName, ToolID: ev.ToolID})
			}
		})
		if err != nil {
			emit(Event{Type: "error", Result: err.Error(), IsError: true})
			return msgs, err
		}

		a.recorder.Record(ctx, ConversationIDFrom(ctx), WeekIDFrom(ctx), model, result.Usage)

		msgs = append(msgs, result.AssistantMessage)

		var toolResults []llm.ContentBlock
		for _, block := range result.AssistantMessage.Content {
			if block.Type != "tool_use" {
				continue
			}
			emit(Event{
				Type:   "tool_call",
				Tool:   block.ToolName,
				ToolID: block.ToolID,
				Input:  string(block.ToolInput),
			})
			out, toolErr := a.callTool(ctx, block.ToolName, block.ToolInput)
			isErr := toolErr != nil
			resultStr := out
			if isErr {
				resultStr = toolErr.Error()
			}
			emit(Event{
				Type:    "tool_result",
				Tool:    block.ToolName,
				ToolID:  block.ToolID,
				Result:  resultStr,
				IsError: isErr,
			})
			toolResults = append(toolResults, llm.ToolResultBlock(block.ToolID, resultStr, isErr))
		}

		if result.StopReason != llm.StopReasonToolUse {
			emit(Event{Type: "done"})
			return msgs, nil
		}
		if len(toolResults) == 0 {
			emit(Event{Type: "error", Result: "model requested tool use but emitted no tool blocks", IsError: true})
			return msgs, errors.New("empty tool batch")
		}

		msgs = append(msgs, llm.NewUserMessage(toolResults...))
	}

	emit(Event{Type: "error", Result: "max iterations reached", IsError: true})
	return msgs, fmt.Errorf("max iterations (%d) reached", a.cfg.MaxIters)
}

func (a *Agent) buildSystemBlocks(ctx context.Context) []llm.SystemBlock {
	blocks := []llm.SystemBlock{{Text: systemPrompt}}
	if b := languageBlock(ctx, a.db, a.log); b.Text != "" {
		blocks = append(blocks, b)
	}
	if b := currentPlanBlock(ctx, a.db, a.log); b.Text != "" {
		blocks = append(blocks, b)
	}
	return blocks
}

func languageBlock(ctx context.Context, db *pgxpool.Pool, log *slog.Logger) llm.SystemBlock {
	s, err := store.GetHouseholdSettings(ctx, db)
	if err != nil {
		log.Warn("language block: falling back to no directive", "err", err)
		return llm.SystemBlock{}
	}
	lang := s.Language
	if lang == "" {
		lang = "sv"
	}
	switch lang {
	case "sv":
		return llm.SystemBlock{Text: `<language>sv</language>
Write all user-facing output in Swedish: chat replies, recipe_md bodies, dish names, notes, and summaries. Use Swedish cooking terminology (sjud, fräs, stek, etc.). Keep tool call arguments as-is (dates, IDs, statuses, category slugs remain English/ISO). Do not translate existing recipe content; only new content you generate from this point.`}
	case "en":
		return llm.SystemBlock{Text: `<language>en</language>
Write all user-facing output in English. Do not translate existing content; leave it in whatever language it was recorded.`}
	default:
		return llm.SystemBlock{}
	}
}

func currentPlanBlock(ctx context.Context, db *pgxpool.Pool, log *slog.Logger) llm.SystemBlock {
	id := WeekIDFrom(ctx)
	if id == 0 {
		return llm.SystemBlock{}
	}
	var iso, start, end, status string
	var dinnerCount int
	err := db.QueryRow(ctx, `
		SELECT w.iso_week, w.start_date::text, w.end_date::text, w.status,
		       COALESCE((SELECT COUNT(*) FROM week_dinners wd WHERE wd.week_id = w.id), 0)
		FROM weeks w WHERE w.id = $1`, id).Scan(&iso, &start, &end, &status, &dinnerCount)
	if err != nil {
		log.Warn("current plan block: lookup failed", "week_id", id, "err", err)
		return llm.SystemBlock{}
	}
	text := fmt.Sprintf(`<current-plan>
  id: %d
  iso_week: %s
  dates: %s -> %s
  status: %s
  dinners: %d
</current-plan>
This chat is tied to the plan above. Assume every user request refers to it. When calling week-scoped tools (add_dinner, update_week, update_dinner, delete_dinner, add_exception, update_exception, delete_exception, record_retrospective, get_week), omit week_id or pass id=%d; any other value will be refused. If the user wants to edit a different plan, tell them to open it and try there.`, id, iso, start, end, status, dinnerCount, id)
	return llm.SystemBlock{Text: text}
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
