// Package providers owns the polymorphic integration registry: LLM providers
// (Anthropic for now; OpenAI later), shopping providers (Willys for now;
// ICA, Matspar, etc. later). Each kind has a stable string id and a
// free-form config_json that callers interpret.
//
// Credentials are stored in the config map by convention-named keys
// (api_key, password, token, secret). The REST layer masks those on read
// so the UI never has to think about it.
package providers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Kind string

const (
	KindAnthropic    Kind = "anthropic"
	KindOpenAI       Kind = "openai"
	KindOpenAICompat Kind = "openai_compat"
	KindWillys       Kind = "willys"
)

// Known tracks the set of provider kinds the app knows about, plus human
// metadata the Settings UI uses to render a form without hard-coding.
type KindInfo struct {
	Kind        Kind    `json:"kind"`
	Category    string  `json:"category"` // "llm" | "shopping"
	DisplayName string  `json:"display_name"`
	Fields      []Field `json:"fields"`
}

type Field struct {
	Key         string        `json:"key"`
	Label       string        `json:"label"`
	Type        string        `json:"type"`              // "text" | "password" | "select"
	Options     []FieldOption `json:"options,omitempty"` // only for "select"
	Default     string        `json:"default,omitempty"`
	Placeholder string        `json:"placeholder,omitempty"`
	Required    bool          `json:"required,omitempty"`
	Hint        string        `json:"hint,omitempty"`
}

type FieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// DefaultAnthropicModel is the fallback used when no Anthropic provider row
// exists yet or the stored value is empty. Kept as a const so agent code can
// reference it without duplicating the string.
const DefaultAnthropicModel = "claude-sonnet-4-6"
const DefaultOpenAIModel = "gpt-5.4"
const openAIBaseURL = "https://api.openai.com/v1"

var Known = []KindInfo{
	{
		Kind:        KindAnthropic,
		Category:    "llm",
		DisplayName: "provider.anthropic.name",
		Fields: []Field{
			{Key: "api_key", Label: "provider.api_key", Type: "password", Placeholder: "sk-ant-…", Required: true},
			{
				Key:     "model",
				Label:   "provider.model",
				Type:    "select",
				Default: DefaultAnthropicModel,
				Hint:    "provider.anthropic.model_hint",
				Options: []FieldOption{
					{Value: "claude-haiku-4-5", Label: "provider.anthropic.haiku"},
					{Value: "claude-sonnet-4-6", Label: "provider.anthropic.sonnet"},
					{Value: "claude-opus-4-7", Label: "provider.anthropic.opus"},
				},
			},
		},
	},
	{
		Kind:        KindOpenAI,
		Category:    "llm",
		DisplayName: "provider.openai.name",
		Fields: []Field{
			{Key: "api_key", Label: "provider.api_key", Type: "password", Placeholder: "sk-…", Required: true},
			{
				Key:     "model",
				Label:   "provider.model",
				Type:    "select",
				Default: DefaultOpenAIModel,
				Hint:    "provider.openai.model_hint",
				Options: []FieldOption{
					{Value: "gpt-4.1-nano", Label: "GPT-4.1 Nano ($0.10/$0.40)"},
					{Value: "gpt-4.1-mini", Label: "GPT-4.1 Mini ($0.40/$1.60)"},
					{Value: "gpt-5.4-nano", Label: "GPT-5.4 Nano ($0.20/$1.25)"},
					{Value: "gpt-5.4-mini", Label: "GPT-5.4 Mini ($0.75/$4.50)"},
					{Value: "gpt-4.1", Label: "GPT-4.1 ($2/$8)"},
					{Value: "gpt-5.1", Label: "GPT-5.1 ($1.25/$10)"},
					{Value: "gpt-5.4", Label: "GPT-5.4 ($2.50/$15)"},
				},
			},
		},
	},
	{
		Kind:        KindOpenAICompat,
		Category:    "llm",
		DisplayName: "provider.openai_compat.name",
		Fields: []Field{
			{Key: "base_url", Label: "provider.openai_compat.base_url", Type: "text", Placeholder: "http://localhost:11434/v1", Required: true, Hint: "provider.openai_compat.base_url_hint"},
			{Key: "api_key", Label: "provider.api_key", Type: "password", Hint: "provider.openai_compat.api_key_hint"},
			{Key: "model", Label: "provider.model", Type: "text", Placeholder: "llama3.1:8b", Required: true, Hint: "provider.openai_compat.model_hint"},
		},
	},
	{
		Kind:        KindWillys,
		Category:    "shopping",
		DisplayName: "Willys",
		Fields: []Field{
			{Key: "username", Label: "provider.willys.username", Type: "text", Required: true},
			{Key: "password", Label: "provider.password", Type: "password", Required: true},
		},
	},
}

func KindInfoFor(kind Kind) (KindInfo, bool) {
	for _, k := range Known {
		if k.Kind == kind {
			return k, true
		}
	}
	return KindInfo{}, false
}

// Provider is the stored record.
type Provider struct {
	Kind      Kind           `json:"kind"`
	Enabled   bool           `json:"enabled"`
	Config    map[string]any `json:"config"`
	UpdatedAt string         `json:"updated_at"`
}

// Mask returns a copy of p with secret fields replaced by the Store's
// per-instance sentinel string. The sentinel is random per process so a
// user who picks it as their password (astronomically unlikely, but
// cheap to guarantee) cannot have it silently discarded on write.
func (s *Store) Mask(p Provider) Provider {
	cp := p
	cp.Config = make(map[string]any, len(p.Config))
	info, _ := KindInfoFor(p.Kind)
	for k, v := range p.Config {
		if isSecretField(info, k) {
			if str, ok := v.(string); ok && str != "" {
				cp.Config[k] = s.sentinel
				continue
			}
			cp.Config[k] = ""
			continue
		}
		cp.Config[k] = v
	}
	return cp
}

// Sentinel is the random string returned in place of secret fields on read.
// Exposed so callers (e.g. tests) can round-trip it back through Upsert.
func (s *Store) Sentinel() string { return s.sentinel }

func isSecretField(info KindInfo, key string) bool {
	for _, f := range info.Fields {
		if f.Key == key {
			return f.Type == "password"
		}
	}
	// Fallback: generic key-name heuristics so unknown kinds still mask.
	switch key {
	case "api_key", "password", "token", "secret":
		return true
	}
	return false
}

// Store is a thin CRUD layer over the providers table. When constructed
// with a master key, secret fields (kind metadata type "password") are
// transparently AES-GCM encrypted on write and decrypted on read.
type Store struct {
	Pool     *pgxpool.Pool
	crypt    *cryptor
	sentinel string // random, per-process; returned in place of secret values on read
}

// New constructs a Store. masterKey may be nil (plaintext mode) or a
// 32-byte key (AES-256-GCM). Returns an error only for malformed keys.
func New(pool *pgxpool.Pool, masterKey []byte) (*Store, error) {
	c, err := newCryptor(masterKey)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("sentinel: %w", err)
	}
	return &Store{
		Pool:     pool,
		crypt:    c,
		sentinel: "redacted:" + hex.EncodeToString(buf),
	}, nil
}

// EncryptString wraps plaintext in the standard enc:v1: envelope. The crypt
// field is non-nil in production (LoadOrGenerateMasterKey always returns a
// key), but the nil branch is preserved so legacy plaintext rows from older
// deployments still round-trip through the API without corruption.

func (s *Store) EncryptString(plain string) (string, error) {
	if s.crypt == nil || isEncrypted(plain) {
		return plain, nil
	}
	return s.crypt.encrypt(plain)
}

// DecryptString unwraps a value previously returned by EncryptString.
// Accepts plaintext (no enc:v1: prefix) and returns it unchanged so rows
// written before MASTER_KEY was configured keep working after rollout.
func (s *Store) DecryptString(v string) (string, error) {
	if s.crypt == nil || !isEncrypted(v) {
		return v, nil
	}
	return s.crypt.decrypt(v)
}

var ErrNotFound = errors.New("provider not found")

func (s *Store) List(ctx context.Context) ([]Provider, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT kind, enabled, config_json::text, updated_at::text
		FROM providers ORDER BY kind`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Provider{}
	for rows.Next() {
		var p Provider
		var cfg string
		if err := rows.Scan(&p.Kind, &p.Enabled, &cfg, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(cfg), &p.Config); err != nil {
			return nil, err
		}
		s.decryptSecrets(p.Kind, p.Config)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Get(ctx context.Context, kind Kind) (*Provider, error) {
	var p Provider
	var cfg string
	err := s.Pool.QueryRow(ctx, `
		SELECT kind, enabled, config_json::text, updated_at::text
		FROM providers WHERE kind = $1`, kind).
		Scan(&p.Kind, &p.Enabled, &cfg, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(cfg), &p.Config); err != nil {
		return nil, err
	}
	s.decryptSecrets(p.Kind, p.Config)
	return &p, nil
}

// decryptSecrets replaces any enc:v1: wrapped values with their plaintext.
// Unencrypted values pass through; lets plaintext rows coexist with
// encrypted ones during MASTER_KEY rollout.
func (s *Store) decryptSecrets(kind Kind, config map[string]any) {
	if s.crypt == nil {
		return
	}
	info, _ := KindInfoFor(kind)
	for _, f := range info.Fields {
		if f.Type != "password" {
			continue
		}
		v, ok := config[f.Key].(string)
		if !ok || !isEncrypted(v) {
			continue
		}
		if plain, err := s.crypt.decrypt(v); err == nil {
			config[f.Key] = plain
		}
		// On decrypt error we leave the wrapped value in place; the convenience
		// accessors will report the provider as misconfigured, which is the
		// right signal for "wrong MASTER_KEY".
	}
}

// encryptSecrets wraps plaintext secret values. Already-encrypted values
// pass through. No-ops when encryption is disabled.
func (s *Store) encryptSecrets(kind Kind, config map[string]any) error {
	if s.crypt == nil {
		return nil
	}
	info, _ := KindInfoFor(kind)
	for _, f := range info.Fields {
		if f.Type != "password" {
			continue
		}
		v, ok := config[f.Key].(string)
		if !ok || v == "" || isEncrypted(v) {
			continue
		}
		wrapped, err := s.crypt.encrypt(v)
		if err != nil {
			return err
		}
		config[f.Key] = wrapped
	}
	return nil
}

// UpsertPatch merges the given patch into the existing config. Keys whose
// value is the Store's sentinel are left as-is (UI sent the masked value
// back untouched). Empty-string values *delete* the key.
type UpsertPatch struct {
	Enabled *bool
	Config  map[string]any
}

func (s *Store) Upsert(ctx context.Context, kind Kind, patch UpsertPatch) (*Provider, error) {
	current, err := s.Get(ctx, kind)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	var config map[string]any
	enabled := false
	if current != nil {
		config = current.Config
		enabled = current.Enabled
	}
	if config == nil {
		config = make(map[string]any)
	}
	if patch.Enabled != nil {
		enabled = *patch.Enabled
	}
	for k, v := range patch.Config {
		if str, ok := v.(string); ok {
			if str == s.sentinel {
				continue // UI echoed the mask; keep existing value
			}
			if str == "" {
				delete(config, k)
				continue
			}
		}
		config[k] = v
	}
	if err := s.encryptSecrets(kind, config); err != nil {
		return nil, err
	}
	cfgJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO providers (kind, enabled, config_json)
		VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (kind) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			config_json = EXCLUDED.config_json`,
		kind, enabled, string(cfgJSON))
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, kind)
}

// ---------------------------------------------------------------------------
// Convenience accessors
// ---------------------------------------------------------------------------

// AnthropicAPIKey returns the Anthropic API key from the providers row, or
// "" if none is configured. Never returns an error; callers treat empty as
// "not configured yet, show a Settings pointer".
func (s *Store) AnthropicAPIKey(ctx context.Context) string {
	p, err := s.Get(ctx, KindAnthropic)
	if err != nil || !p.Enabled {
		return ""
	}
	if v, ok := p.Config["api_key"].(string); ok {
		return v
	}
	return ""
}

// AnthropicModel returns the model slug chosen in Settings, falling back to
// DefaultAnthropicModel if unset or the provider is disabled.
func (s *Store) AnthropicModel(ctx context.Context) string {
	p, err := s.Get(ctx, KindAnthropic)
	if err != nil {
		return DefaultAnthropicModel
	}
	if v, ok := p.Config["model"].(string); ok && v != "" {
		return v
	}
	return DefaultAnthropicModel
}

type OpenAIConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

func (s *Store) OpenAIConfig(ctx context.Context) (OpenAIConfig, bool) {
	p, err := s.Get(ctx, KindOpenAI)
	if err != nil || !p.Enabled {
		return OpenAIConfig{}, false
	}
	apiKey, _ := p.Config["api_key"].(string)
	if apiKey == "" {
		return OpenAIConfig{}, false
	}
	model, _ := p.Config["model"].(string)
	if model == "" {
		model = DefaultOpenAIModel
	}
	return OpenAIConfig{BaseURL: openAIBaseURL, APIKey: apiKey, Model: model}, true
}

type OpenAICompatConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

func (s *Store) OpenAICompatConfig(ctx context.Context) (OpenAICompatConfig, bool) {
	p, err := s.Get(ctx, KindOpenAICompat)
	if err != nil || !p.Enabled {
		return OpenAICompatConfig{}, false
	}
	baseURL, _ := p.Config["base_url"].(string)
	model, _ := p.Config["model"].(string)
	if baseURL == "" || model == "" {
		return OpenAICompatConfig{}, false
	}
	apiKey, _ := p.Config["api_key"].(string)
	return OpenAICompatConfig{BaseURL: baseURL, APIKey: apiKey, Model: model}, true
}

type WillysCreds struct {
	Username string
	Password string
}

func (s *Store) WillysCredentials(ctx context.Context) (WillysCreds, bool) {
	p, err := s.Get(ctx, KindWillys)
	if err != nil || !p.Enabled {
		return WillysCreds{}, false
	}
	u, _ := p.Config["username"].(string)
	pw, _ := p.Config["password"].(string)
	if u == "" || pw == "" {
		return WillysCreds{}, false
	}
	return WillysCreds{Username: u, Password: pw}, true
}
