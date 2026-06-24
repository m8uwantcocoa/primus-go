package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/m8uwantcocoa/primus-go/internal"
)

type raceRequest struct {
	Endpoints []internal.ApiEndpoint    `json:"endpoints"`
	TimeoutMs int                       `json:"timeout_ms"`
	Benchmark internal.BenchmarkRequest `json:"benchmark"`
}

func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "X-Winner, X-Duration")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

// StartServer initializes and starts the HTTP server that listens for incoming requests to the /race endpoint. It sets up
// the necessary route and handles incoming requests using the handleRace function. The server runs on port 8080 and
// will print a message to the console when it starts successfully. If there are any issues with starting the server,
// it will log the error accordingly.
func StartServer() {
	http.HandleFunc("/health", withCORS(handleHealth))
	http.HandleFunc("/race", withCORS(handleRace))
	http.HandleFunc("/race/all", withCORS(handleRaceAll))
	http.HandleFunc("/race/benchmark", withCORS(handleBenchmark))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server is running on port %s...\n", port)
	http.ListenAndServe(":"+port, nil)
}

func handleRace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST method is allowed for Race endpoint", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed/error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req raceRequest

	err = json.Unmarshal(body, &req)
	fmt.Printf("received %d endpoints\n", len(req.Endpoints))

	if err != nil {
		http.Error(w, "invalid JSON format", http.StatusBadRequest)
		return
	}

	result := internal.Race(context.Background(), req.Endpoints, req.TimeoutMs)

	if result.Error != nil {
		http.Error(w, "sadly, all endpoints failed", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Winner", result.Name)
	w.Header().Set("X-Duration", result.Duration.String())
	w.Write(result.Body)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}

func handleRaceAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req raceRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "invalid JSON format", http.StatusBadRequest)
		return
	}

	results := internal.RaceAll(context.Background(), req.Endpoints, req.TimeoutMs)

	type resultResponse struct {
		Name     string `json:"name"`
		Duration string `json:"duration"`
		Error    string `json:"error,omitempty"`
		Winner   bool   `json:"winner"`
	}

	var responses []resultResponse
	for i, result := range results {
		r := resultResponse{
			Name:     result.Name,
			Duration: result.Duration.String(),
			Winner:   i == 0,
		}
		if result.Error != nil {
			r.Error = result.Error.Error()
		}
		responses = append(responses, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

func handleBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req raceRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "invalid JSON format", http.StatusBadRequest)
		return
	}

	results := internal.Benchmark(context.Background(), req.Endpoints, req.Benchmark, req.TimeoutMs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
