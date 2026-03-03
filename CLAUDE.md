# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

US Military Tracker — a self-evolving OSINT system that tracks US military assets globally via a dynamically updating KML file served by GitHub Pages. Licensed under GPLv3.

## Build & Run

```bash
go build ./cmd/tracker        # Build the binary
go test ./...                  # Run all tests
./tracker                      # Run main pipeline (needs env vars)
./tracker --evaluate           # Run chairman evaluation
./tracker --refresh-static     # Run monthly static data refresh
./tracker --evolve             # Run weekly architecture evolution
```

## Required Environment Variables

AI providers (set whichever keys you have as GitHub Actions secrets):

| Secret Name | Provider | Model | Free Tier |
|---|---|---|---|
| `GEMINI_API_KEY` | Google Gemini | gemini-2.0-flash-lite | Yes |
| `GROQ_API_KEY` | Groq | llama-3.3-70b-versatile | Yes (daily token cap) |
| `MISTRAL_API_KEY` | Mistral | mistral-small-latest | Yes |
| `DEEPSEEK_API_KEY` | DeepSeek | deepseek-chat | Free credits |
| `OPENROUTER_API_KEY` | OpenRouter | llama-3.3-70b-instruct:free | Yes |
| `OPENAI_API_KEY` | OpenAI ChatGPT | gpt-4o-mini | No ($5 trial credits) |
| `ANTHROPIC_API_KEY` | Anthropic Claude | claude-haiku-4-5-20251001 | No (trial credits) |

Local model (always available, zero cost): **Ollama** running **Qwen 2.5 1.5B** on the GitHub runner's CPU. Installed automatically by the workflow.

Data sources:
- `AISSTREAM_API_KEY` — vessel tracking (free via GitHub auth at aisstream.io)
- `GNEWS_API_KEY` — news search (free tier at gnews.io)
- `ACLED_API_KEY` — conflict data (free at acleddata.com)

## Architecture

**Pipeline:** Collect (parallel) → AI Council → Chairman Synthesis (with fallback chain) → Generate KML → Publish

**Chairman fallback chain:** If the selected chairman fails (rate limit, etc.), the system tries other successful council members as chairman, then falls back to parsing individual council responses directly. This prevents a single provider outage from losing all enrichment.

**Four scheduled workflows:**
- `update-tracker.yml` — every hour, main pipeline
- `evaluate-chairman.yml` — after each tracker run, scores chairman quality
- `update-static-data.yml` — monthly, refreshes bases/registries
- `evolve-architecture.yml` — weekly, monitors AI providers + GitHub platform

**AI Council (LLM Council pattern):** Multiple AI providers analyze data in parallel. A dynamically selected chairman synthesizes the consensus. Offline evaluation scores chairman quality using the local Ollama model (zero API cost). Council exchange format is JSON — chosen over YAML/TOON/TSV for parsing reliability across models of all sizes (especially the 1.5B local model).

**Self-evolving:** Weekly workflow discovers new models, shadow-tests candidates for 3 weeks, and promotes them if they outperform current members. Also monitors GitHub runner specs and adapts.

## Key Packages

- `internal/collectors/` — aircraft (3 ADS-B APIs), vessels (AISStream WebSocket), events (GDELT/ACLED), news (GNews/RSS)
- `internal/enrichment/` — council orchestration, chairman selection/scoring, evaluation heuristics
- `internal/enrichment/providers/` — Completer interface, OpenAI-compatible client (Groq/Mistral/DeepSeek/OpenRouter/ChatGPT/Ollama), Gemini SDK, Anthropic Claude
- `internal/kml/` — KML/XML generation with encoding/xml
- `internal/platform/` — GitHub runner monitoring, model discovery/evolution
- `internal/models/` — shared data types

## Dependencies

Only 2 third-party deps: `github.com/coder/websocket` (AISStream) and `google.golang.org/genai` (Gemini). Everything else uses stdlib.

## Design Doc

Full design: `docs/plans/2026-03-02-military-tracker-design.md`
Implementation plan: `docs/plans/2026-03-02-military-tracker-implementation.md`
