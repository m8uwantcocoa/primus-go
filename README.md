<div align="center">
  <img src="https://go.dev/blog/gopher/header.jpg" alt="Go Gopher" width="300"/>

# primus-go

> *Primus* — Latin for "first". Fire multiple API endpoints at once, and the fastest one wins.

```
primus-go starting... 3... 2... 1... Go!
```

</div>

---

## Live Demo

**[→ Try it here](https://primus-web.vercel.app/)**

> **The demo is limited.** It runs a hosted version of the frontend connected to a shared backend.
>
> | Feature | Demo | Self-hosted |
> |---|---|---|
> | `/race` — fastest wins | ✅ Up to 5 endpoints | ✅ Unlimited |
> | `/race/all` — ranked results | ✅ Up to 5 endpoints | ✅ Unlimited |
> | `/race/benchmark` — performance stats | ❌ Not available | ✅ Full access |
> | Concurrency & load testing | ❌ Not available | ✅ Full access |
> | Custom timeout | ✅ | ✅ |
>
> To get the full experience, clone and run it yourself — see [Getting started](#getting-started).

---

## What is this?

`primus-go` is a small HTTP server written in Go that races multiple API endpoints (including LLMs) against each other and returns the response from whichever one replies first.

You send it a list of endpoints. It fires them all simultaneously using goroutines. The first one that responds wins — the others get cancelled via context. Simple as that.

This is useful when:
- You have redundant services or mirrors and want the fastest one
- You're comparing response times between different providers (e.g. two LLM APIs)
- You want automatic failover without complex load-balancing setup
- LLM providers are chained as fallbacks: Gemini → Groq (Llama) → Cohere. If one fails or hits a rate limit, the next is tried automatically
- Pretty much: if you need to guarantee your users always get a response from the fastest available source, this is the layer that makes that happen

---

## How it works

```
Client ──POST /race──► primus-go ──► [endpoint A]
                                  ──► [endpoint B]  ◄── first response wins
                                  ──► [endpoint C]
```

All endpoints are called concurrently. The moment the first successful response comes back, it's forwarded to the client and the rest are cancelled. If none respond within the timeout, the server returns a `502`.

---

## Getting started

**Prerequisites:** Go 1.21+

```bash
git clone https://github.com/m8uwantcocoa/primus-go
cd primus-go
go run .
```

The server starts on port `8080`.

---

## Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | Health check |
| `/race` | POST | Race endpoints, return the first winner |
| `/race/all` | POST | Race endpoints, return results from all of them |
| `/race/benchmark` | POST | Run repeated races and collect performance stats |

---

## `/health`

A simple liveness check. No body required.

```bash
curl http://localhost:8080/health
```

```json
{"status": "ok"}
```

---

## `/race`

Fires all endpoints simultaneously and returns the response body from whichever one replies first. The rest are cancelled.

```bash
curl -X POST http://localhost:8080/race \
  -H "Content-Type: application/json" \
  -d '{
    "endpoints": [
      {
        "name": "primary",
        "url": "https://api.example.com/data",
        "method": "GET"
      },
      {
        "name": "backup",
        "url": "https://backup.example.com/data",
        "method": "GET"
      }
    ]
  }'
```

**Response headers:**

| Header | Description |
|---|---|
| `X-Winner` | The `name` of the endpoint that responded first |
| `X-Duration` | How long that endpoint took to respond |

The response body is whatever the winning endpoint returned.

---

## `/race/all`

Same as `/race`, but instead of returning on the first winner, it waits for **all** endpoints to finish and returns a ranked list. Useful when you want to see how every endpoint performed, not just who won.

```bash
curl -X POST http://localhost:8080/race/all \
  -H "Content-Type: application/json" \
  -d '{
    "endpoints": [
      {"name": "fast-api",  "url": "https://api.example.com/ping",   "method": "GET"},
      {"name": "slow-api",  "url": "https://api.example2.com/ping",  "method": "GET"},
      {"name": "third-api", "url": "https://api.example3.com/ping",  "method": "GET"}
    ]
  }'
```

**Response:**

```json
[
  {"name": "fast-api",  "duration": "112ms", "winner": true},
  {"name": "third-api", "duration": "230ms", "winner": false},
  {"name": "slow-api",  "duration": "418ms", "winner": false, "error": ""}
]
```

Results are returned in the order they finished. The first entry (`"winner": true`) is the one that came back first.

---

## `/race/benchmark`

Runs repeated races across your endpoints and collects performance statistics — average latency, fastest/slowest response, standard deviation, consistency rating, success rate, and win count. Optionally includes a load test to measure degradation under concurrency.

```bash
curl -X POST http://localhost:8080/race/benchmark \
  -H "Content-Type: application/json" \
  -d '{
    "endpoints": [
      {"name": "provider-a", "url": "https://api.provider-a.com/ping", "method": "GET"},
      {"name": "provider-b", "url": "https://api.provider-b.com/ping", "method": "GET"}
    ],
    "benchmark": {
      "runs": 10,
      "concurrency": 5,
      "include_load": true
    }
  }'
```

**Benchmark fields:**

| Field | Type | Default | Description |
|---|---|---|---|
| `runs` | int | `5` | How many sequential race rounds to run |
| `concurrency` | int | `3` | Number of concurrent workers used during the load test |
| `include_load` | bool | `false` | Whether to run a load test in addition to the sequential runs |

**Response:**

```json
[
  {
    "name": "provider-a",
    "avg_ms": 143.2,
    "fastest_ms": 98.0,
    "slowest_ms": 201.0,
    "std_dev_ms": 22.4,
    "consistency": "good",
    "success_rate": 100,
    "wins": 7,
    "avg_ms_under_load": 178.5,
    "degradation": "none"
  },
  {
    "name": "provider-b",
    "avg_ms": 198.7,
    "fastest_ms": 150.0,
    "slowest_ms": 310.0,
    "std_dev_ms": 55.1,
    "consistency": "moderate",
    "success_rate": 90,
    "wins": 3,
    "avg_ms_under_load": 340.2,
    "degradation": "moderate"
  }
]
```

**Consistency ratings** (based on standard deviation):

| Rating | Std Dev |
|---|---|
| `excellent` | < 10ms |
| `good` | 10–30ms |
| `moderate` | 30–60ms |
| `unreliable` | > 60ms |

**Degradation ratings** (load avg vs normal avg):

| Rating | Slowdown |
|---|---|
| `none` | < 20% |
| `moderate` | 20–50% |
| `severe` | > 50% |

`avg_ms_under_load` and `degradation` are only present when `include_load: true`.

---

## Configurable timeout

Every endpoint accepts an optional `timeout_ms` field. If omitted, the default is **5000ms (5 seconds)**. If all endpoints fail or exceed the timeout, `/race` returns `502 Bad Gateway`.

```json
{
  "timeout_ms": 2000,
  "endpoints": [...]
}
```

This works on all three POST endpoints (`/race`, `/race/all`, `/race/benchmark`).

---

## Endpoint fields

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | A label for this endpoint (used in responses and headers) |
| `url` | string | yes | The full URL to call |
| `method` | string | no | HTTP method. Defaults to `GET` |
| `headers` | object | no | Key-value pairs added to the request |
| `body` | string | no | Request body (for POST/PUT requests) |

You can also pass headers and a request body per endpoint, which makes it work with POST-based APIs and ML model endpoints:

```json
{
  "endpoints": [
    {
      "name": "model-a",
      "url": "https://api.provider-a.com/generate",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer YOUR_TOKEN",
        "Content-Type": "application/json"
      },
      "body": "{\"prompt\": \"hello world\"}"
    },
    {
      "name": "model-b",
      "url": "https://api.provider-b.com/generate",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer YOUR_OTHER_TOKEN",
        "Content-Type": "application/json"
      },
      "body": "{\"prompt\": \"hello world\"}"
    }
  ]
}
```

---

## Using as a Go package

You can embed the server directly in your own Go app:

```bash
go get github.com/m8uwantcocoa/primus-go/cmd
```

```go
package main

import "github.com/m8uwantcocoa/primus-go/cmd"

func main() {
    cmd.StartServer() // starts all endpoints on :8080, blocking
}
```

Or call the core functions directly from `internal`:

```go
import "github.com/m8uwantcocoa/primus-go/internal"

// Race — returns the first winner
result := internal.Race(ctx, endpoints, 3000)

// RaceAll — returns all results
results := internal.RaceAll(ctx, endpoints, 3000)

// Benchmark — returns performance stats
config := internal.BenchmarkRequest{Runs: 10, Concurrency: 5, IncludeLoad: true}
stats := internal.Benchmark(ctx, endpoints, config, 3000)
```

---

## Project structure

```
primus-go/
├── main.go              # entry point, starts the server
├── cmd/
│   └── server.go        # HTTP handlers for all endpoints
└── internal/
    └── racer.go         # Race, RaceAll, Benchmark logic
```

---

## Built with

- Pure Go standard library — no external dependencies
- `net/http` for the server
- Goroutines + channels for concurrent requests
- `context.WithTimeout` for race timeout and cancellation
- `sync.WaitGroup` for load test concurrency

---

<div align="center"><i>Made with Go</i></div>
