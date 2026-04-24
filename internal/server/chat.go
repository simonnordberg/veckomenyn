package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/simonnordberg/veckomenyn/internal/agent"
)

type chatRequest struct {
	ConversationID *int64 `json:"conversation_id"`
	WeekID         *int64 `json:"week_id"`
	Message        string `json:"message"`
}

// handleChat runs one user turn against the agent and streams the events back
// as SSE. The client POSTs and then reads the response body as a text/event-stream.
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		http.Error(w, "message required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if req.WeekID != nil {
		ctx = agent.WithWeekID(ctx, *req.WeekID)
	}

	convID, err := s.ensureConversation(ctx, req.ConversationID, req.WeekID, req.Message)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	ctx = agent.WithConversationID(ctx, convID)

	history, err := s.loadHistory(ctx, convID)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	sendSSE(w, flusher, "meta", map[string]any{"conversation_id": convID})

	if err := s.saveMessage(ctx, convID, "user", req.Message); err != nil {
		s.log.Error("save user msg", "err", err)
	}

	var assistantBuf strings.Builder
	_, runErr := s.agent.Run(ctx, history, req.Message, func(ev agent.Event) {
		sendSSE(w, flusher, "event", ev)
		if ev.Type == "text" {
			assistantBuf.WriteString(ev.Text)
		}
	})

	cancelled := runErr != nil && errors.Is(ctx.Err(), context.Canceled)
	if runErr != nil && !cancelled {
		s.log.Warn("agent run error", "conv", convID, "err", runErr)
	} else if cancelled {
		s.log.Info("agent run cancelled by client", "conv", convID)
		// Stream is probably already gone, but try to signal clients that
		// reconnect fast enough.
		sendSSE(w, flusher, "event", agent.Event{Type: "cancelled"})
	}

	// Persist whatever text made it through before the client bailed. Use a
	// detached context with a short timeout; the request ctx is cancelled.
	if assistantBuf.Len() > 0 {
		saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer saveCancel()
		if err := s.saveMessage(saveCtx, convID, "assistant", assistantBuf.String()); err != nil {
			s.log.Error("save assistant msg", "err", err)
		}
	}

	sendSSE(w, flusher, "end", map[string]any{"conversation_id": convID})
}

// loadHistory returns a simple text-only replay of the conversation. Tool
// calls made in prior turns are intentionally dropped; the agent re-reads
// current state via tools each turn, so the history only needs user/assistant
// text for context.
func (s *Server) loadHistory(ctx context.Context, convID int64) ([]anthropic.MessageParam, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT role, content_json
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at, id`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []anthropic.MessageParam
	for rows.Next() {
		var role string
		var raw []byte
		if err := rows.Scan(&role, &raw); err != nil {
			return nil, err
		}
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		switch role {
		case "user":
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
		case "assistant":
			out = append(out, anthropic.NewAssistantMessage(anthropic.NewTextBlock(text)))
		}
	}
	return out, rows.Err()
}

func (s *Server) saveMessage(ctx context.Context, convID int64, role, text string) error {
	payload, err := json.Marshal(text)
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx,
		`INSERT INTO messages (conversation_id, role, content_json) VALUES ($1, $2, $3::jsonb)`,
		convID, role, payload)
	return err
}

// ensureConversation returns the conversation id to write to:
//   - explicit conversation_id → that one (bumps updated_at)
//   - week_id given → the latest existing conversation for that week, or a
//     fresh one attached to that week
//   - neither → a fresh unattached conversation
func (s *Server) ensureConversation(ctx context.Context, id *int64, weekID *int64, firstMessage string) (int64, error) {
	if id != nil && *id > 0 {
		var exists bool
		if err := s.db.Pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM conversations WHERE id=$1)`, *id).Scan(&exists); err != nil {
			return 0, err
		}
		if !exists {
			return 0, fmt.Errorf("conversation %d not found", *id)
		}
		_, _ = s.db.Pool.Exec(ctx, `UPDATE conversations SET updated_at = now() WHERE id = $1`, *id)
		return *id, nil
	}
	if weekID != nil && *weekID > 0 {
		var existing int64
		err := s.db.Pool.QueryRow(ctx,
			`SELECT id FROM conversations WHERE week_id = $1 ORDER BY updated_at DESC LIMIT 1`,
			*weekID).Scan(&existing)
		if err == nil {
			_, _ = s.db.Pool.Exec(ctx, `UPDATE conversations SET updated_at = now() WHERE id = $1`, existing)
			return existing, nil
		}
		// Not found → fall through to create with week_id set.
		var newID int64
		err = s.db.Pool.QueryRow(ctx,
			`INSERT INTO conversations (week_id, title) VALUES ($1, $2) RETURNING id`,
			*weekID, conversationTitle(firstMessage)).Scan(&newID)
		return newID, err
	}
	var newID int64
	err := s.db.Pool.QueryRow(ctx,
		`INSERT INTO conversations (title) VALUES ($1) RETURNING id`,
		conversationTitle(firstMessage)).Scan(&newID)
	return newID, err
}

func conversationTitle(msg string) string {
	msg = strings.TrimSpace(msg)
	if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
		msg = msg[:idx]
	}
	if len(msg) > 80 {
		msg = msg[:80] + "…"
	}
	return msg
}

func sendSSE(w http.ResponseWriter, f http.Flusher, event string, data any) {
	b, _ := json.Marshal(data)
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	f.Flush()
}

// ---------------------------------------------------------------------------
// Conversation read endpoints
// ---------------------------------------------------------------------------

type conversationRow struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	StartedAt string `json:"started_at"`
	UpdatedAt string `json:"updated_at"`
}

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Pool.Query(r.Context(), `
		SELECT id, title, started_at::text, updated_at::text
		FROM conversations
		ORDER BY updated_at DESC
		LIMIT 50`)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	defer rows.Close()
	out := []conversationRow{}
	for rows.Next() {
		var c conversationRow
		if err := rows.Scan(&c.ID, &c.Title, &c.StartedAt, &c.UpdatedAt); err != nil {
			s.internalError(w, r, "request", err)
			return
		}
		out = append(out, c)
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": out})
}

type messageRow struct {
	ID        int64  `json:"id"`
	Role      string `json:"role"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

// handleGetWeekConversation returns the most recent conversation attached
// to the given week_id, including its messages. 204 if none exist.
func (s *Server) handleGetWeekConversation(w http.ResponseWriter, r *http.Request) {
	weekID, ok := parsePositiveID(w, r, "id")
	if !ok {
		return
	}
	var c conversationRow
	err := s.db.Pool.QueryRow(r.Context(), `
		SELECT id, title, started_at::text, updated_at::text
		FROM conversations
		WHERE week_id = $1
		ORDER BY updated_at DESC
		LIMIT 1`, weekID).
		Scan(&c.ID, &c.Title, &c.StartedAt, &c.UpdatedAt)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(), `
		SELECT id, role, content_json, created_at::text
		FROM messages
		WHERE conversation_id=$1
		ORDER BY created_at, id`, c.ID)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	defer rows.Close()
	msgs := []messageRow{}
	for rows.Next() {
		var m messageRow
		var raw []byte
		if err := rows.Scan(&m.ID, &m.Role, &raw, &m.CreatedAt); err != nil {
			s.internalError(w, r, "request", err)
			return
		}
		_ = json.Unmarshal(raw, &m.Text)
		msgs = append(msgs, m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversation": c, "messages": msgs})
}

// handleDeleteWeekConversations wipes every conversation attached to a
// week. Used when a conversation has gone sideways and the user wants a
// clean slate. Messages cascade-delete via the conversations_id FK.
func (s *Server) handleDeleteWeekConversations(w http.ResponseWriter, r *http.Request) {
	weekID, ok := parsePositiveID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.db.Pool.Exec(r.Context(),
		`DELETE FROM conversations WHERE week_id = $1`, weekID); err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r, "id")
	if !ok {
		return
	}

	var c conversationRow
	err := s.db.Pool.QueryRow(r.Context(),
		`SELECT id, title, started_at::text, updated_at::text FROM conversations WHERE id=$1`, id).
		Scan(&c.ID, &c.Title, &c.StartedAt, &c.UpdatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(), `
		SELECT id, role, content_json, created_at::text
		FROM messages
		WHERE conversation_id=$1
		ORDER BY created_at, id`, id)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	defer rows.Close()
	msgs := []messageRow{}
	for rows.Next() {
		var m messageRow
		var raw []byte
		if err := rows.Scan(&m.ID, &m.Role, &raw, &m.CreatedAt); err != nil {
			s.internalError(w, r, "request", err)
			return
		}
		_ = json.Unmarshal(raw, &m.Text)
		msgs = append(msgs, m)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"conversation": c,
		"messages":     msgs,
	})
}
