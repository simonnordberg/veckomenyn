# Security

## Reporting

Open a [private security advisory](https://github.com/simonnordberg/veckomenyn/security/advisories/new). Don't file a public issue. Include:

- What the issue is and what it lets an attacker do.
- Steps to reproduce.
- Any fix you already have in mind.

Expect an initial reply within a few days.

## Threat model

Veckomenyn runs on a home LAN, for one household. No user accounts, no built-in auth. The network boundary (Tailscale, home VPN, firewall) is what keeps strangers out. Exposing the service to the public internet without auth in front of it is outside scope.

Inside that model:

- Secrets encrypt at rest. With `MASTER_KEY` set, API keys and store credentials live AES-256-GCM wrapped in the `providers` table.
- Secrets don't leave the server. The REST layer replaces password fields with a random per-process sentinel. The UI echoes it back verbatim to mean "leave this alone."
- `/api/chat` is rate-limited per IP. A misbehaving process can't run up the Anthropic bill.
- Path parameters are bounds-checked. Request bodies are capped at 1 MiB.
- Internal errors log server-side and return a generic 500. No stack traces or SQL strings on the wire.

## Not in scope

- Multi-tenant isolation. There's no concept of tenants.
- Auth. See above.
- Upstream store APIs. If Willys ships a broken session cookie, there's only so much we can do.

## Non-security bugs

Regular issues go to [github.com/simonnordberg/veckomenyn/issues](https://github.com/simonnordberg/veckomenyn/issues).
