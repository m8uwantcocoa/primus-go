<div align="center">
  <img src="https://go.dev/blog/gopher/header.jpg" alt="Go Gopher" width="300"/>

# primus-go

> *Primus* — Latin for "first". Fire multiple API endpoints at once, and the fastest one wins.

```
primus-go starting... 3... 2... 1... Go!
```

</div>

---

## What is this?

`primus-go` is a small HTTP server written in Go that races multiple API endpoints against each other and returns the response from whichever one replies first.

You send it a list of endpoints. It fires them all simultaneously using goroutines. The first one that responds wins — the others get cancelled via context. Simple as that.

This is useful when:
- You have redundant services or mirrors and want the fastest one
- You're comparing response times between different providers (e.g. two LLM APIs)
- You want automatic failover without complex load-balancing setup
- Pretty much : if you need to guarantee your users always get a response from the fastest available source, this is the layer that makes that happen

---

## How it works

```
Client ──POST /race──► primus-go ──► [endpoint A]
                                  ──► [endpoint B]  ◄── first response wins
                                  ──► [endpoint C]
```

All endpoints are called concurrently. The moment the first successful response comes back, it's forwarded to the client and the rest are cancelled. If none respond within **5 seconds**, the server returns a `502`.

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

## Usage

### As an external server

Run `primus-go` as a standalone service and point your app at it. Send a `POST` request to `/race` with a JSON body listing the endpoints you want to race:

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

Or from Go code:

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

func main() {
    payload := map[string]any{
        "endpoints": []map[string]any{
            {"name": "primary", "url": "https://api.example.com/data", "method": "GET"},
            {"name": "backup",  "url": "https://backup.example.com/data", "method": "GET"},
        },
    }

    body, _ := json.Marshal(payload)
    resp, _ := http.Post("http://localhost:8080/race", "application/json", bytes.NewReader(body))
    defer resp.Body.Close()

    result, _ := io.ReadAll(resp.Body)
    fmt.Println("Winner:", resp.Header.Get("X-Winner"))
    fmt.Println("Took:  ", resp.Header.Get("X-Duration"))
    fmt.Println("Body:  ", string(result))
}
```

### Imported into your own Go project

You can embed the server directly in your own Go app by importing the `cmd` package:

```bash
go get github.com/m8uwantcocoa/primus-go/cmd
```

```go
package main

import "github.com/m8uwantcocoa/primus-go/cmd"

func main() {
    // starts the /race endpoint on :8080, blocking
    cmd.StartServer()
}
```

This is useful if you want `primus-go` to be one handler among others in a larger service, or if you want to control startup yourself without running it as a separate process.

---

You can also pass headers and a request body per endpoint, which makes it work with POST-based APIs and ML model endpoints too:

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

### Response

The response body is whatever the winning endpoint returned. Two extra headers tell you who won:

| Header | Description |
|---|---|
| `X-Winner` | The `name` of the endpoint that responded first |
| `X-Duration` | How long that endpoint took to respond |

---

## Endpoint fields

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | A label for this endpoint (returned in `X-Winner`) |
| `url` | string | yes | The full URL to call |
| `method` | string | no | HTTP method. Defaults to `GET` |
| `headers` | object | no | Key-value pairs added to the request |
| `body` | string | no | Request body (for POST/PUT requests) |

---

## Timeout

All races have a hard **5-second timeout**. If every endpoint fails or takes longer than that, the server responds with `502 Bad Gateway`.

You can change the timeout in [internal/racer.go](internal/racer.go#L83):

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second) // change this
```

---

## Project structure

```
primus-go/
├── main.go              # entry point, starts the server
├── cmd/
│   └── server.go        # HTTP server and /race handler
└── internal/
    └── racer.go         # core racing logic (goroutines + context)
```

---

## Built with

- Pure Go standard library — no external dependencies
- `net/http` for the server
- Goroutines + channels for concurrent requests
- `context.WithTimeout` for the race timeout and cancellation

---

<div align="center"><i>Made with Go</i></div>
