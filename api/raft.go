package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func (s *storeApiServer) joinHandler(w http.ResponseWriter, r *http.Request) {
	m := map[string]string{}
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		slog.Error("Failed to decode request body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	nodeAddress, ok := m["addr"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	nodeID, ok := m["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := s.store.Join(nodeID, nodeAddress); err != nil {
		slog.Error("Failed to join node", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
