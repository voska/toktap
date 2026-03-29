# toktap

LLM API usage proxy. Intercepts traffic between AI clients and provider APIs, extracts token usage from SSE streams and JSON responses, writes metrics to InfluxDB.

## Architecture

```
Client (Claude Code, Codex, SDK) → toktap → Provider API (Anthropic, OpenAI, etc.)
                                       ↓
                                  InfluxDB + Loki
```

Requests arrive at `/<route>/...`, the route prefix is stripped, and the request is forwarded to the configured upstream. On the way back, SSE streams are tapped to extract token usage without modifying the response.

## Code Layout

- `cmd/toktap/` — entry point, graceful shutdown
- `internal/proxy/` — reverse proxy, metadata extraction, SSE tapping, request logging
- `internal/sse/` — SSE scanner, Anthropic and OpenAI usage extractors
- `internal/influx/` — async InfluxDB writer
- `internal/pricing/` — YAML-based pricing table with hot reload
- `internal/config/` — env-based configuration
- `deploy/config/` — routes.yaml, pricing.yaml

## Route Config

Routes are data-driven via `deploy/config/routes.yaml`. Each route declares:
- `upstream` — target URL
- `provider` — canonical provider name for metrics (e.g., multiple routes can map to `openai`)
- `inject_stream_options` — whether to inject `stream_options.include_usage` for OpenAI-compatible APIs
- `chrome_transport` — whether to use Chrome TLS fingerprint (for Cloudflare Bot Management)

Adding a new route requires zero Go code — just a YAML entry.

## Key Technical Details

- **SSE tapping:** TapReader wraps the response body, feeds bytes to the SSE scanner, extracts usage via provider-specific extractors, fires a callback on Close()
- **Header stripping:** Cloudflare Tunnel headers (`Cdn-Loop`, `Cf-*`) are stripped to prevent RFC 8586 loop detection. `Accept-Encoding` is stripped so SSE arrives uncompressed.
- **Chrome TLS:** Uses utls `HelloChrome_Auto` for providers behind Cloudflare Bot Management. Go's default JA3 fingerprint gets blocked.
- **Metadata:** Device from `X-Device` header, harness from User-Agent, auth type from API key format
- **Graceful shutdown:** SIGTERM drains in-flight connections (5 min timeout)

## Recorder

Optional conversation recording for training data collection. When `RECORDER_PATH` is set, toktap captures full request/response bodies as JSONL files (one per day).

- **Non-streaming:** Full request and response JSON bodies are stored.
- **SSE streaming:** All SSE events are captured via `RecordingTapReader`, which extends `TapReader` to buffer events alongside the existing zero-copy usage extraction.
- **Storage:** Daily JSONL files at `$RECORDER_PATH/YYYY-MM-DD.jsonl`. Each line is a complete `recorder.Record` with request body, response body or SSE events, metadata, and token counts.
- **Config:** Set `RECORDER_PATH=/data/recordings` to enable. Empty string (default) disables recording.
