package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/m8uwantcocoa/primus-go/internal"
)

type raceRequest struct {
	Endpoints []internal.ApiEndpoint `json:"endpoints"`
	TimeoutMs int                    `json:"timeout_ms"`
}

// StartServer initializes and starts the HTTP server that listens for incoming requests to the /race endpoint. It sets up
// the necessary route and handles incoming requests using the handleRace function. The server runs on port 8080 and
// will print a message to the console when it starts successfully. If there are any issues with starting the server,
// it will log the error accordingly.
func StartServer() {
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/race", handleRace)
	fmt.Println("Server is running on port 8080...")
	http.ListenAndServe(":8080", nil)
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
