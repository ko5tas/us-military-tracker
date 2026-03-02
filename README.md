# US Military Tracker

A self-evolving OSINT system that tracks US military assets globally and publishes a live-updating KML file via GitHub Pages. Load it into Google Earth once and it auto-refreshes every 15 minutes with aircraft positions, vessel locations, military bases, conflict zones, news events, and AI-generated intelligence summaries.

Runs entirely on free-tier infrastructure — zero cost.

## Quick Start

1. Download [`network-link.kml`](https://ko5tas.github.io/us-military-tracker/network-link.kml)
2. Open it in [Google Earth Pro](https://earth.google.com/intl/en/earth/download/gep/agree.html) (File > Open)
3. Done — the map auto-refreshes every 15 minutes

That's it. No accounts, no API keys, no installation required.

## What It Tracks

| Layer | Source | Update Frequency |
|---|---|---|
| Military aircraft | ADS-B (Airplanes.live, ADSB.one, ADSB.lol) | Every 15 min |
| Naval vessels | AISStream.io WebSocket | Every 15 min |
| Military bases | ~750 US installations worldwide | Monthly refresh |
| Conflict zones | GDELT, ACLED | Every 15 min |
| Military news | GNews API, RSS feeds (DVIDS, DoD, Defense News) | Every 15 min |
| AI intelligence summary | Multi-LLM council analysis | Every 15 min |

## How It Works

```
Collect (parallel) --> AI Council --> Chairman Synthesis --> Generate KML --> Publish
                                                                             |
                                                               GitHub Pages serves KML
                                                                             |
                                                               Google Earth auto-refreshes
```

**Data collection:** Four collectors run in parallel every 15 minutes via GitHub Actions, pulling from free public APIs.

**AI enrichment:** An [LLM Council](https://github.com/karpathy/llm-council) (Gemini, Groq, Mistral, DeepSeek, local Ollama) independently analyzes the raw data. A chairman model synthesizes their analyses into the final intelligence layer. The chairman is selected automatically based on quality scores from an offline evaluation pipeline.

**Self-evolution:** The system monitors AI provider changes (rate limits, new models, outages) and GitHub platform changes (runner specs, free tier limits) weekly, adapting automatically.

## Architecture

```
us-military-tracker/
├── cmd/tracker/          # Main pipeline entry point
├── internal/
│   ├── collectors/       # Aircraft, vessels, events, news collectors
│   ├── enrichment/       # AI council, chairman, evaluator
│   │   └── providers/    # Gemini, Groq, Mistral, DeepSeek, OpenRouter, Ollama
│   ├── kml/              # KML/XML generation
│   ├── models/           # Shared data types
│   └── platform/         # GitHub monitoring, model evolution
├── config/               # AI provider config, platform limits, chairman scores
├── data/static/          # Military bases, aircraft types (monthly refresh)
├── output/               # Generated tracker.kml (served by GitHub Pages)
└── .github/workflows/    # 4 scheduled workflows + Pages deployment
```

## Workflows

| Workflow | Schedule | Purpose |
|---|---|---|
| `update-tracker.yml` | Every 15 min | Collect > Enrich > Generate KML > Publish |
| `evaluate-chairman.yml` | After each tracker run | Score chairman quality, update rankings |
| `update-static-data.yml` | Monthly (1st) | Refresh bases, registries, squadrons |
| `evolve-architecture.yml` | Weekly (Sunday) | Monitor AI providers + GitHub platform |
| `deploy-pages.yml` | On push | Deploy KML to GitHub Pages |

## Tech Stack

- **Language:** Go
- **Compute:** GitHub Actions (ubuntu-latest, 4 vCPU, 16 GB RAM)
- **Hosting:** GitHub Pages
- **AI:** Gemini, Groq, Mistral, DeepSeek, OpenRouter (free tiers) + local Ollama (Qwen 2.5 1.5B)
- **Budget:** $0/month

## Documentation

- [System Design](docs/plans/2026-03-02-military-tracker-design.md) — Full architecture, data sources, AI council design, self-evolving system, KML structure, and key design decisions
- [Implementation Plan](docs/plans/2026-03-02-military-tracker-implementation.md) — 24 TDD tasks across 8 phases with complete code and test specifications

## Building from Source

```bash
go build -o tracker ./cmd/tracker
./tracker
```

Optional environment variables for AI enrichment and additional data sources:

```
GEMINI_API_KEY       # Google AI Studio (free)
GROQ_API_KEY         # Groq console (free)
MISTRAL_API_KEY      # Mistral platform (free)
DEEPSEEK_API_KEY     # DeepSeek platform (free credits)
OPENROUTER_API_KEY   # OpenRouter (free)
GNEWS_API_KEY        # GNews (free)
AISSTREAM_API_KEY    # AISStream.io (free, GitHub auth)
```

The tracker works without any keys — aircraft positions and news are collected from unauthenticated public APIs.

## License

GPL-3.0 — see [LICENSE](LICENSE)
