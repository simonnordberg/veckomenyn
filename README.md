<p align="center">
  <img src="docs/logo.svg" alt="Veckomenyn" width="420" />
</p>

<p align="center"><em>Familjens veckomeny, planerad och handlad.</em></p>

---

An AI agent plans your family's week of dinners and builds the grocery cart. Self-hosted. It learns what's in the fridge, which kid won't eat cilantro, which store brands you trust.

LLM backends are pluggable: Anthropic (Claude), OpenAI (GPT), or any OpenAI-compatible endpoint (llama.cpp, Ollama, etc.). Shopping backends are pluggable too. Willys.se ships today.

## The loop

1. Set household constraints. Dinners per week, servings, allergies, what's usually in the pantry.
2. Ask the agent to plan a week. Swap dishes, regenerate, nudge until it looks right.
3. Let the agent build the grocery cart. It aggregates ingredients across all dinners, picks one product per ingredient, verifies.
4. Place the order in the store's own UI. Veckomenyn stops at cart-ready. Delivery and payment stay where they belong.
5. After the week, record a retrospective. That feedback shapes next week.

![Weekly menu](docs/screenshots/week-light.png)

## Get started

- **[Quickstart (LAN)](docs/quickstart.md)**. `podman compose up -d` on a trusted home network.
- **[Deploy on a remote VM (Tailscale)](docs/deploy-tailscale.md)**. One curl-piped command, ~5 minutes, no public ports.
- **[Upgrading](docs/upgrading.md)**. Channels, automatic pre-migration snapshots, the in-app update banner.
- **[Backups & restore](docs/backups.md)**. What's automatic, what's optional, how to restore.
- **[Configuration reference](docs/configuration.md)**. Every env var.

## Screenshots

| Week view (light) | Week view (dark) |
|---|---|
| ![](docs/screenshots/week-light.png) | ![](docs/screenshots/week-dark.png) |
| **Chat drawer.** The agent narrates what it's doing. | **Chat** (dark). |
| ![](docs/screenshots/chat-open.png) | ![](docs/screenshots/chat-dark.png) |
| **Settings.** Household defaults and integrations. | **Settings** (dark). |
| ![](docs/screenshots/settings.png) | ![](docs/screenshots/settings-dark.png) |
| **Preferences.** Free-form markdown per category. | **Print preview.** Paper stays light whatever the theme. |
| ![](docs/screenshots/preferences.png) | ![](docs/screenshots/print-preview.png) |

## Layout

```
cmd/
  veckomenyn/       HTTP server + embedded SPA. Main binary.
internal/
  agent/            Agent loop. System prompt, tools, streaming. Provider-agnostic.
  llm/              LLM provider interface. Anthropic, OpenAI, OpenAI-compatible backends.
  willys/           Willys.se HTTP client.
  shopping/         Store-agnostic Provider interface. Willys adapter.
  providers/        Registry for LLM and shopping backends. AES-GCM at rest.
  server/           chi router, SSE chat, handlers.
  store/            pgxpool + goose migration runner.
  migrations/       Embedded SQL migrations.
  backup/           pg_dump-based snapshotter + scheduler.
  updates/          GitHub-release polling for the update banner.
  seed/             Template preferences embedded for first-run seeding.
web/                React 19, TypeScript, Tailwind v4, Biome, Vite.
```

The Go binary embeds the Vite bundle via `//go:embed`.

## Threat model

There is no built-in authentication. The network boundary is the trust boundary. Run on a trusted LAN, on a tailnet ([Tailscale guide](docs/deploy-tailscale.md)), or behind any equivalent VPN. Never on the public internet. Full picture in [SECURITY.md](SECURITY.md).

## Contributing

[CONTRIBUTING.md](CONTRIBUTING.md) for PRs and the local dev loop. [SECURITY.md](SECURITY.md) for vulnerabilities.

## License

MIT.
