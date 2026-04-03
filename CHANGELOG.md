# Changelog

## v0.3.0

- WebSocket proxy support: transparent tunnel for `Upgrade: websocket` requests
- Enables Codex CLI (Responses API) through toktap without code changes
- Bidirectional `io.Copy` tunnel, no framing inspection, zero new dependencies
- TLS connections force HTTP/1.1 ALPN to prevent h2 negotiation breaking WebSocket upgrades

## v0.2.0

- Conversation recording: capture full request/response bodies as daily JSONL files via `RECORDER_PATH`
- SSE streaming responses captured through `RecordingTapReader` with full event buffering
- Non-streaming responses stored as complete JSON bodies
- AGENTS.md as canonical agent instructions (CLAUDE.md symlinks to it)
- Site and llms.txt updated with recording documentation

## v0.1.0

Initial release.

- Transparent reverse proxy for LLM API providers
- SSE stream tapping for token usage extraction (Anthropic, OpenAI formats)
- Non-streaming response extraction
- Data-driven route configuration via YAML
- InfluxDB metrics writing (async)
- YAML-based pricing table with hot reload
- Structured JSON request logging via slog
- Chrome TLS fingerprint spoofing for Cloudflare Bot Management
- Cloudflare Tunnel header stripping (RFC 8586 loop prevention)
- Graceful shutdown with 5-minute drain timeout
- Device, harness, and auth type metadata extraction
