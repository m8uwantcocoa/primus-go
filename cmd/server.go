package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/m8uwantcocoa/primus-go/internal"
)

type raceRequest struct {
	Endpoints []internal.ApiEndpoint `json:"endpoints"`
}

func StartServer() {
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

	result := internal.Race(r.Context(), req.Endpoints)

	if result.Error != nil {
		http.Error(w, "sadly, all endpoints failed", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Winner", result.Name)
	w.Header().Set("X-Duration", result.Duration.String())
	w.Write(result.Body)
}
