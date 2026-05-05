# Configuration

Env vars only cover what the binary needs before the UI loads. LLM provider, model selection, store credentials, household defaults, and backup schedule all live in Settings.

## LLM providers

Configured in **Settings > Integrations**. One LLM provider is active at a time. Tab buttons switch the active provider; each provider's fields (API keys, model selection) are saved independently.

### Anthropic

Requires an API key from [console.anthropic.com](https://console.anthropic.com/settings/keys). Model selection via dropdown: Haiku 4.5, Sonnet 4.6, Opus 4.7. Prompt caching is used automatically to reduce costs on multi-turn tool loops.

### OpenAI

Requires an API key from [platform.openai.com](https://platform.openai.com/api-keys). Model selection via dropdown: GPT-4.1 (nano/mini/full) and GPT-5 (mini/full).

### OpenAI-compatible (local/other)

For llama.cpp, Ollama, or any endpoint that serves the OpenAI chat completions API. Fields:

- **Base URL**: the `/v1` endpoint, e.g. `http://127.0.0.1:8082/v1`. Use `127.0.0.1` instead of `localhost` if the server only binds IPv4.
- **API key**: optional. Leave empty for local backends.
- **Model**: the model name the backend expects, e.g. `gemma-4-26B-A4B-APEX-I-Balanced.gguf`.

**Note on reasoning models**: models that use internal chain-of-thought (Gemma APEX, OpenAI o-series) consume tokens on reasoning before producing visible output. The UI will show a pause before text streams in. This is normal.

### Test connection

After saving provider config, use the **Test connection** button to verify the endpoint responds. It sends a minimal completion and shows the model's reply.

## Environment variables

| Var | Purpose |
|---|---|
| `MASTER_KEY` | 32-byte base64 AES key encrypting provider secrets in the DB. **Optional**: auto-generated and persisted on first boot. Set explicitly only to manage the key externally (KMS, sealed secrets). Generate with `openssl rand -base64 32`. |
| `DATABASE_URL` | Postgres DSN. Set automatically by compose. |
| `HTTP_ADDR` | Listen address. Defaults to `:8080`. |
| `HOST_PORT` | Host port mapped to the container's 8080. Defaults to 8080. |
| `BACKUP_DIR` | Where pre-migration `pg_dump` snapshots are written. Set by compose to `/var/lib/veckomenyn/backups`. Empty disables snapshots. |
| `PREMIGRATION_BACKUP_KEEP` | Number of pre-migration snapshots to retain. Defaults to 10. |
| `VECKOMENYN_SKIP_PREMIGRATION_BACKUP` | Set to `1` to allow migration even if the pre-migration `pg_dump` fails. Dev-only escape hatch. |
| `DISABLE_UPDATE_CHECK` | Set to `1` to opt out of polling GitHub releases for the in-app update banner. |
| `TS_AUTHKEY` | Only used by the Tailscale overlay (`docker-compose.tailscale.yml`). Tailscale auth key for joining the tailnet. |
