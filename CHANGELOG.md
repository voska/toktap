# Changelog

## v0.5.3

- Bundled default routes and pricing so standalone Docker and Homebrew installs start without config files
- Config lookup order: explicit `ROUTES_CONFIG` / `PRICING_CONFIG` paths, working-directory `routes.yaml` / `pricing.yaml` files, then built-in defaults
- Without a pricing file, `cost_usd` now uses the bundled pricing table for listed models instead of staying at zero; set `PRICING_CONFIG` to use your own prices
- Standalone Docker instructions now include the InfluxDB 2 token, organization, and bucket settings
- Both runtime images now use Alpine 3.24

## v0.5.2

- Patched `golang.org/x/crypto` and `golang.org/x/net` dependency vulnerabilities (#5)
- Patched Go toolchain and dependency vulnerabilities (#6)

## v0.5.1

- Reverted to a Homebrew formula. Casks are macOS-only, so v0.5.0 silently dropped `brew install` support on Linux

## v0.5.0

- Distributed as a Homebrew **cask** instead of a formula; `brew update` migrates existing installs automatically
- goreleaser deprecated `brews` in v2.10 (hard since v2.16), so casks are the supported path

## v0.4.0

- Publish the Homebrew formula into `Formula/` in `voska/homebrew-tap` instead of the tap root, so `brew install voska/tap/toktap` resolves again

## v0.3.1

- Fix high-cardinality bug: `request_id` moved from InfluxDB tag to string field
- Resolves Grafana query timeouts caused by ~100k unique series on `llm_usage` measurement
- Existing series require manual cleanup (drop measurement or wait for retention)

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
