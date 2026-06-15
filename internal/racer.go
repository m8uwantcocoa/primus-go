package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ApiEndpoint struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Result struct {
	Name     string
	Body     []byte
	Duration time.Duration
	Error    error
}

func runner(ctx context.Context, endpoint ApiEndpoint, ch chan<- Result) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.URL, nil)

	if err != nil {
		ch <- Result{Name: endpoint.Name, Error: err}
		return
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		ch <- Result{Name: endpoint.Name, Error: err}
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	fmt.Printf("runner %s got %d bytes in %s\n", endpoint.Name, len(body), time.Since(start))

	ch <- Result{Name: endpoint.Name, Body: body, Duration: time.Since(start), Error: err}
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
