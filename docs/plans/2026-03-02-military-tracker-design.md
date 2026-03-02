# US Military Tracker — System Design

## Overview

A self-evolving OSINT (Open Source Intelligence) system that tracks US military assets globally and publishes a dynamically updating KML file served via GitHub Pages. Users load a Network Link KML into Google Earth once, and it auto-refreshes every 15 minutes with live aircraft positions, vessel locations, military base data, conflict zones, news events, and AI-generated intelligence summaries.

The system runs entirely on free-tier infrastructure: GitHub Actions for compute, GitHub Pages for hosting, free AI APIs for analysis, and a local LLM on the runner as a zero-cost safety net.

## Architecture

### Core Pipeline (every 15 minutes)

```
Collect (parallel) → AI Council → Chairman Synthesis → Generate KML → Publish
                                                                        │
                                                          GitHub Pages serves KML
                                                                        │
                                                          Google Earth auto-refreshes
```

**Approach:** Monolithic single Go binary, structured internally as separate packages per data source so it can be refactored into distributed Actions later if needed.

### Four Scheduled Workflows

| Workflow | File | Schedule | Purpose |
|---|---|---|---|
| Main tracker | `update-tracker.yml` | Every 15 min | Collect → Enrich → Generate KML → Publish |
| Chairman evaluation | `evaluate-chairman.yml` | After each tracker run | Score chairman quality, update rankings |
| Static data refresh | `update-static-data.yml` | Monthly (1st) | Refresh bases, registries, squadrons |
| Architecture evolution | `evolve-architecture.yml` | Weekly (Sunday) | Monitor AI providers + GitHub platform, discover new models |

### Time Budget Per 15-Minute Cycle

| Step | Estimated time |
|---|---|
| Collect data (parallel API calls) | ~20 sec |
| Restore Ollama cache + start server | ~15 sec |
| Stage 1: Council analysis (parallel API + local) | ~40 sec |
| Stage 2: Chairman synthesis | ~20 sec |
| Generate KML | ~10 sec |
| Git commit + push | ~15 sec |
| Evaluate previous cycle (local Ollama) | ~45 sec |
| **Total** | **~3 min of 15 min window** |

## Project Structure

```
us-military-tracker/
├── cmd/
│   └── tracker/
│       └── main.go                  # Entry point — orchestrates the pipeline
├── internal/
│   ├── collectors/
│   │   ├── aircraft.go              # Airplanes.live, ADSB.one, ADSB.lol
│   │   ├── vessels.go               # AISStream.io WebSocket
│   │   ├── events.go                # GDELT, ACLED
│   │   └── news.go                  # GNews, RSS feeds (DVIDS, DOD, Defense News)
│   ├── enrichment/
│   │   ├── council.go               # Council orchestration — parallel dispatch, collect responses
│   │   ├── chairman.go              # Chairman selection + synthesis
│   │   ├── evaluator.go             # Offline evaluation of chairman output
│   │   ├── providers/
│   │   │   ├── provider.go          # Common interface for all AI providers
│   │   │   ├── gemini.go            # Google Gemini client
│   │   │   ├── groq.go              # Groq client
│   │   │   ├── mistral.go           # Mistral client
│   │   │   ├── deepseek.go          # DeepSeek client
│   │   │   ├── openrouter.go        # OpenRouter fallback
│   │   │   └── ollama.go            # Local Ollama client
│   │   └── shadow/
│   │       └── shadow.go            # Shadow testing for candidate models
│   ├── kml/
│   │   └── generator.go             # KML/XML generation
│   ├── platform/
│   │   ├── github.go                # GitHub platform monitoring (runner specs, limits)
│   │   └── evolution.go             # Model discovery, benchmarking, promotion/demotion
│   └── models/
│       └── types.go                 # Shared data types (Aircraft, Vessel, Event, etc.)
├── config/
│   ├── providers.json               # Self-managed AI provider config (council, candidates, scores)
│   └── platform.json                # GitHub platform limits and adaptation rules
├── data/
│   ├── static/
│   │   ├── bases.json               # US military bases worldwide (~750)
│   │   ├── aircraft_types.json      # ICAO hex → aircraft type mappings
│   │   ├── vessel_registry.json     # Hull numbers, ship classes
│   │   └── squadrons.json           # Unit/squadron designations
│   ├── aircraft.json                # Latest collected aircraft positions
│   ├── vessels.json                 # Latest collected vessel positions
│   ├── events.json                  # Latest GDELT/ACLED events
│   ├── news.json                    # Latest news articles
│   └── evolution/
│       └── changelog.md             # Auto-generated evolution history
├── output/
│   └── tracker.kml                  # The generated KML served by GitHub Pages
├── network-link.kml                 # Static file users load into Google Earth once
├── .github/
│   └── workflows/
│       ├── update-tracker.yml       # Every 15 min — main pipeline
│       ├── evaluate-chairman.yml    # After each tracker run
│       ├── update-static-data.yml   # Monthly — refresh static datasets
│       └── evolve-architecture.yml  # Weekly — AI + platform monitoring
├── go.mod
├── go.sum
├── CLAUDE.md
└── docs/
    └── plans/
        └── 2026-03-02-military-tracker-design.md   # This file
```

## Data Sources

### Real-Time (polled every 15 minutes)

| Source | Data | Auth | Rate limit | Notes |
|---|---|---|---|---|
| Airplanes.live `/v2/mil/` | Military aircraft positions | None | 1 req/sec | Unfiltered, dedicated military endpoint |
| ADSB.one `/v2/mil/` | Military aircraft positions | None | 1 req/sec | Independent feeder network, redundant source |
| ADSB.lol | Military aircraft positions | None | None (for now) | Open source, third redundant source |
| AISStream.io | Vessel positions via WebSocket | API key (free, GitHub auth) | 1 sub update/sec | Military ships often disable AIS |
| GDELT | Global military events | Free registration | 30 days rolling | Updated every 15 min, georeferenced |
| ACLED | Conflict zone data | Free registration | Free tier | Real-time conflict events |
| GNews | Military news | API key (free) | 100 req/day | 80,000+ sources, keyword search |
| RSS feeds | DoD announcements | None | None | DVIDS, DOD, DSCA, Defense News, Military.com |

### Static (refreshed monthly)

| Dataset | Content | Source |
|---|---|---|
| Military bases | ~750 US bases worldwide with coordinates | Public databases + AI-verified monthly |
| Aircraft types | ICAO hex code → aircraft type/model mapping | ICAO database + news |
| Vessel registry | Hull numbers, ship classes, commissioning dates | Navy press releases |
| Squadrons | Unit designations, home bases | DoD announcements |
| Deployments | Known forward deployment locations | GDELT trends + news |

### Deduplication

Aircraft from Airplanes.live, ADSB.one, and ADSB.lol are deduplicated by ICAO hex code, keeping the record with the most recent timestamp.

## AI Council (LLM Council Pattern)

Inspired by [Karpathy's LLM Council](https://github.com/karpathy/llm-council), adapted for OSINT with an offline evaluation feedback loop.

### Stage 1: Independent Analysis (parallel)

Each council member receives the same raw data and prompt. They independently classify aircraft types, identify vessel classes, cross-reference positions with news, and flag unusual patterns. Responses are anonymized.

| Provider | Model | Type | Daily capacity | Role |
|---|---|---|---|---|
| Groq | Llama 3.3 70B | API | 14,400 req/day | Council member |
| Gemini | 2.5 Flash-Lite | API | ~1,000 req/day | Council member |
| Ollama | Qwen 2.5 1.5B | Local on runner | Unlimited | Council member + all evaluations |
| Mistral | Experiment tier | API | 1B tokens/month | Council member (backup) |
| DeepSeek | V3 | API | Unlimited (30-day free credits) | Council member |
| OpenRouter | Best free model | API | Varies | Fallback |

### Stage 2: Chairman Synthesis

A single model synthesizes all council analyses into the final enrichment. The chairman is selected dynamically based on quality scores from the offline evaluation pipeline.

| Provider | Model | Type | Daily capacity | Chairman role |
|---|---|---|---|---|
| Gemini | 2.5 Flash | API | ~250/day | Primary chairman |
| Groq | Llama 3.3 70B | API | 14,400/day | Backup chairman |
| Ollama | Qwen 2.5 1.5B | Local | Unlimited | Last-resort chairman |

### Special uses

- **Gemini 2.5 Pro** (~25/day): Used approximately once per hour for higher-quality synthesis. Usage tracked via counter file.
- **Minimum quorum**: 2 council members for consensus, 1 if only one responds, raw data if all fail.

### Offline Evaluation (after each cycle)

Runs on the **local Ollama model only** (zero API cost):

**Automated heuristics:**
- Data fidelity — did the chairman preserve all positions from raw data?
- Completeness — did it classify all assets?
- Format correctness — valid structured output?
- Hallucination detection — mentions of assets not in raw data?

**AI-evaluated quality:**
- Classification accuracy (callsign patterns match classifications)
- Cross-reference quality (positions meaningfully connected to news)
- Consensus respect (chairman honored majority agreement)

**Scoring:**
```json
{
  "gemini-flash": { "avg_score": 0.82, "runs": 47, "last_5": [0.85, 0.79, 0.88, 0.81, 0.77] },
  "groq-llama":   { "avg_score": 0.78, "runs": 31, "last_5": [0.80, 0.75, 0.82, 0.76, 0.79] },
  "local-qwen":   { "avg_score": 0.65, "runs": 18, "last_5": [0.68, 0.62, 0.70, 0.63, 0.61] }
}
```

Chairman with highest rolling average is selected. New models get an exploration phase.

## Self-Evolving Architecture (Weekly)

The system monitors and adapts to changes in both AI providers and the GitHub platform.

### AI Provider Evolution

**Health check:** Hit each API with a minimal test request. Read rate limit headers (`X-RateLimit-Limit`, `X-RateLimit-Remaining`). Detect free tier changes. Flag providers requiring payment.

**Model discovery:**
- Fetch OpenRouter `/api/v1/models` for new free models
- Fetch Ollama library page for new small models (<4 GB)
- Fetch provider pricing/docs pages for tier changes
- Check HuggingFace trending GGUF models

**Candidate lifecycle:**
```
DISCOVERED → TESTING (shadow, 3 weeks) → PROMOTED or REJECTED
```

Shadow members receive the same prompts as real council members and get scored by the evaluator, but their output is NOT used in the KML. After 3 weeks, if they outperform the weakest current member, they're promoted.

**Member removal:**
```
ACTIVE → DEGRADED (1 failed health check) → REMOVED (3 consecutive failures)
```

### GitHub Platform Monitoring

**Hardware detection (every 15-min run):**
Record runner CPU (`runtime.NumCPU()`), RAM (`/proc/meminfo`), and disk at pipeline start. Compare against stored values. Immediate detection of spec changes.

**Policy detection (weekly):**
Fetch GitHub docs/changelog. Local AI extracts current limits for: free minutes/month, cache size, Pages bandwidth, runner specs, new runner types.

**Monitored values and adaptation rules:**

| Feature | Current value | Adaptation |
|---|---|---|
| Runner RAM | 16 GB | <8 GB → smaller local model. >24 GB → try 7B model |
| Runner CPUs | 4 vCPU | Affects local inference speed estimates |
| Runner disk | 14 GB | Affects model caching |
| Free minutes/month | 2,000 | <1,500 → increase interval to 20 min. More → consider 10 min |
| Actions cache | 10 GB | <5 GB → cache only primary local model |
| Pages bandwidth | 100 GB/month | Monitor usage, warn if approaching |
| Repo size | 1 GB recommended | >800 MB → prune old data commits, keep 7 days |
| GPU runners | Not free (yet) | If free GPU appears → switch to GPU-accelerated local model |

### Repo Size Management

Every 15 minutes we commit updated JSON + KML. Growth estimate:
- Month 1: ~50 MB
- Month 6: ~300 MB
- Year 1: ~600 MB

When repo exceeds 800 MB: squash old data commits (keep last 7 days) or migrate historical data to GitHub Releases as artifacts.

## KML Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
<Document>
  <name>US Military Tracker</name>
  <description>Last updated: {timestamp} | Chairman: {model} (score: {score})</description>

  <Folder>  <!-- Real-time aircraft -->
    <name>Aircraft ({count} tracked)</name>
    <Folder><name>US Air Force</name><!-- Placemarks --></Folder>
    <Folder><name>US Navy</name></Folder>
    <Folder><name>US Marines</name></Folder>
    <Folder><name>US Army</name></Folder>
    <Folder><name>Unidentified Military</name></Folder>
  </Folder>

  <Folder>  <!-- Vessels -->
    <name>Naval Vessels ({count} tracked)</name>
    <Folder><name>Aircraft Carriers &amp; Amphibious</name></Folder>
    <Folder><name>Destroyers &amp; Cruisers</name></Folder>
    <Folder><name>Submarines (surfaced/in port)</name></Folder>
    <Folder><name>Support &amp; Logistics</name></Folder>
  </Folder>

  <Folder>  <!-- Static bases -->
    <name>Military Bases (~750 worldwide)</name>
    <Folder><name>Air Force Bases</name></Folder>
    <Folder><name>Naval Bases</name></Folder>
    <Folder><name>Army Bases</name></Folder>
    <Folder><name>Marine Corps Bases</name></Folder>
    <Folder><name>Joint / Coalition Bases</name></Folder>
  </Folder>

  <Folder>  <!-- News-derived ground forces -->
    <name>Ground Forces &amp; Equipment (news-derived)</name>
    <Folder><name>Reported Deployments</name></Folder>
    <Folder><name>Tanks &amp; Armor</name></Folder>
    <Folder><name>Missile / Air Defense</name></Folder>
  </Folder>

  <Folder><name>Military Exercises</name></Folder>
  <Folder><name>Conflict Zones (ACLED/GDELT)</name></Folder>
  <Folder><name>News (last 24h)</name></Folder>
  <Folder><name>AI Intelligence Summary</name></Folder>
</Document>
</kml>
```

Each placemark includes AI-enriched descriptions:
```
C-17 Globemaster III (EVAC01)
Branch: US Air Force
Mission: Aeromedical Evacuation
Altitude: 28,000 ft | Speed: 450 kts | Heading: 270°
AI Assessment: Medical evacuation flight from Ramstein AFB,
likely CONUS-bound. Consistent with routine medevac operations.
Confidence: HIGH (3/3 council members agree)
```

## Network Link KML (user-facing)

The file users load once into Google Earth:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
  <NetworkLink>
    <name>US Military Tracker</name>
    <Link>
      <href>https://ko5tas.github.io/us-military-tracker/tracker.kml</href>
      <refreshMode>onInterval</refreshMode>
      <refreshInterval>900</refreshInterval>
    </Link>
  </NetworkLink>
</kml>
```

## Technology Stack

- **Language:** Go
- **Compute:** GitHub Actions (ubuntu-latest, 4 vCPU, 16 GB RAM)
- **Hosting:** GitHub Pages
- **Local AI:** Ollama with Qwen 2.5 1.5B (cached between runs)
- **AI APIs:** Gemini (free), Groq (free), Mistral (free), DeepSeek (free credits), OpenRouter (free)
- **KML generation:** Go `encoding/xml` (native, no external library)
- **Budget:** $0/month — strictly free tier everything

## Key Design Decisions

1. **Go over Rust/Python** — Fast compilation (critical for CI), native XML/JSON, excellent concurrency for parallel API calls. Rust's compile time would eat into the 15-min budget. Python was ruled out by user preference.

2. **Monolith now, distributed later** — Single binary with clean package separation. Can be split into separate Actions/binaries when needed, without rewriting.

3. **Council over single-LLM** — Multiple perspectives produce better classifications. Consensus reduces hallucination risk. Fallback chain ensures the pipeline never fails completely.

4. **Fixed chairman → adaptive chairman** — Offline evaluation with the local model (zero cost) continuously measures chairman quality. Best-performing model is automatically selected.

5. **Shadow testing for new models** — 3-week evaluation period before promotion. Users never see output from untested models.

6. **Platform self-monitoring** — System detects changes to GitHub runner specs, free tier limits, and AI provider policies. Adapts automatically or falls back gracefully.

7. **15-minute intervals** — Balances freshness with free tier budget. Aircraft positions are the most time-sensitive data; 15 minutes is acceptable for the other layers.

8. **Monthly static refresh** — Prevents data degradation over years. Bases, registries, and squadron data stay current even if the project runs unattended.
