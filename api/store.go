package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/mrtc0-sandbox/hraft/store"
)

const (
	joinPath  = "/internal/cluster/join"
	storePath = "/kv/"
)

type storeApiServer struct {
	addr  string
	store *store.Store
	mux   *http.ServeMux
}

func NewStoreApiServer(addr string, store *store.Store) *storeApiServer {
	return &storeApiServer{
		addr:  addr,
		store: store,
		mux:   http.NewServeMux(),
	}
}

func (s *storeApiServer) Start() error {
	s.mux.HandleFunc(joinPath, s.joinHandler)
	s.mux.HandleFunc(storePath, s.storeHandler)

	return http.ListenAndServe(s.addr, s.mux)
}

func (s *storeApiServer) storeHandler(w http.ResponseWriter, r *http.Request) {
	// buggy. not support /kv/foo/bar case.
	key := filepath.Base(r.URL.Path)

	switch r.Method {
	case http.MethodGet:
		value, err := s.store.Get(key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		b, err := json.Marshal(map[string]string{key: value})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	case http.MethodPut:
		value, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := s.store.Set(key, string(value)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error("Failed to set key", "error", err)
			return
		}

		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		if err := s.store.Delete(key); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error("Failed to delete key", "error", err)
			return
		}

		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
