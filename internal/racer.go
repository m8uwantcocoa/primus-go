package internal

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

type ApiEndpoint struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type Result struct {
	Name     string
	Body     []byte
	Duration time.Duration
	Error    error
}

func runner(ctx context.Context, endpoint ApiEndpoint, ch chan<- Result) {
	start := time.Now()

	var bodyReader io.Reader

	if endpoint.Body != "" {
		bodyReader = strings.NewReader(endpoint.Body)
	}

	if endpoint.Method == "" {
		endpoint.Method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, endpoint.Method, endpoint.URL, bodyReader)

	if err != nil {
		ch <- Result{Name: endpoint.Name, Error: err}
		return
	}

	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		ch <- Result{Name: endpoint.Name, Error: err}
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		ch <- Result{Name: endpoint.Name, Error: err}
		return
	}

	ch <- Result{Name: endpoint.Name, Body: body, Duration: time.Since(start)}
}

func Race(ctx context.Context, endpoints []ApiEndpoint) Result {
	ch := make(chan Result, len(endpoints))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, endpoint := range endpoints {
		go runner(ctx, endpoint, ch)
	}

	return <-ch
}
