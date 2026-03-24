# Changelog

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
