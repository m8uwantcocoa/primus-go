package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ApiEndpoint represents the structure of an API endpoint to be tested. It includes the name of the endpoint,
// the URL, the HTTP method, optional headers, and an optional body for POST/PUT requests. Edited to be more generic so
// that it can support not only APIs but also ML models or any other services that can be called via HTTP. You can further
// enhance it by adding more fields as needed, such as query parameters, authentication details, etc.
type ApiEndpoint struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// Result represents the outcome of calling an API endpoint. It includes the name of the endpoint, the response body,
// the duration of the call, and any error that occurred. This structure allows us to capture all relevant information
// about the API call, which can be useful for debugging and performance analysis. You can also add more fields here if
// needed, such as the HTTP status code, response headers, etc.
type Result struct {
	Name     string
	Body     []byte
	Duration time.Duration
	Error    error
}

// BenchmarkRequest represents the structure of a benchmark request, which includes the number of runs,
// the level of concurrency, and whether to include load testing. This structure can be used to configure the parameters
// of a benchmark test, allowing for flexible and customizable benchmarking of API endpoints or other services. You can
// further enhance this structure by adding more fields as needed, such as specific metrics to collect, thresholds for
// performance, etc.
type BenchmarkRequest struct {
	Runs        int  `json:"runs"`
	Concurrency int  `json:"concurrency"`
	IncludeLoad bool `json:"include_load"`
}

// BenchmarkResult represents the outcome of a benchmark test, including the name of the endpoint, average response time,
// fastest and slowest response times, standard deviation, consistency, success rate, number of wins, and optional fields
// for average response time under load and degradation. This structure allows us to capture comprehensive information about
// the performance of an API endpoint or service under benchmark testing, which can be useful for performance analysis and optimization.
// You can also add more fields here if needed, such as specific percentiles, error rates, etc.
type BenchmarkResult struct {
	Name           string  `json:"name"`
	AvgMs          float64 `json:"avg_ms"`
	FastestMs      float64 `json:"fastest_ms"`
	SlowestMs      float64 `json:"slowest_ms"`
	StdDevMs       float64 `json:"std_dev_ms"`
	Consistency    string  `json:"consistency"`
	SuccessRate    float64 `json:"success_rate"`
	Wins           int     `json:"wins"`
	AvgMsUnderLoad float64 `json:"avg_ms_under_load,omitempty"`
	Degradation    string  `json:"degradation,omitempty"`
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

// Race compares multiple API endpoints and returns the result of the first one that responds successfully.
// It uses a context with a timeout to ensure that it doesn't wait indefinitely for any endpoint, you can change it as needed.
// If all endpoints fail or time out, it returns an error indicating that all endpoints timed out.
func Race(ctx context.Context, endpoints []ApiEndpoint, timeoutMs int) Result {
	if timeoutMs == 0 {
		timeoutMs = 5000
	}

	ch := make(chan Result, len(endpoints))
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)

	defer cancel()

	for _, endpoint := range endpoints {
		go runner(ctx, endpoint, ch)
	}

	select {
	case result := <-ch:
		return result
	case <-ctx.Done():
		return Result{Error: fmt.Errorf("all endpoints timed out after %d ms", timeoutMs)}
	}
}

// RaceAll compares multiple API endpoints and returns the results of all endpoints that respond successfully
// within the specified timeout. It uses a context with a timeout to ensure that it doesn't wait indefinitely for any
// endpoint. If an endpoint fails or times out, it captures the error in the Result struct for that endpoint. This
// function allows you to get a comprehensive view of how all endpoints performed, rather than just the first successful
func RaceAll(ctx context.Context, endpoints []ApiEndpoint, timeoutMs int) []Result {
	if timeoutMs == 0 {
		timeoutMs = 5000
	}
	ch := make(chan Result, len(endpoints))
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)

	defer cancel()

	for _, endpoint := range endpoints {
		go runner(ctx, endpoint, ch)
	}

	results := make([]Result, 0, len(endpoints))

	for range endpoints {
		results = append(results, <-ch)
	}

	return results
}
