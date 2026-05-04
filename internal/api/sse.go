package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/util"
)

const maxSSEClientsPerProject = 100

func (s *Server) handleProjectSSE(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing project id"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientID := util.GenerateID()
	ch := make(chan []byte, 10)

	s.mu.Lock()
	if len(s.sseClients[projectID]) >= maxSSEClientsPerProject {
		s.mu.Unlock()
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "too many SSE connections"))
		return
	}
	if s.sseClients[projectID] == nil {
		s.sseClients[projectID] = make(map[string]chan []byte)
	}
	s.sseClients[projectID][clientID] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sseClients[projectID], clientID)
		if len(s.sseClients[projectID]) == 0 {
			delete(s.sseClients, projectID)
		}
		s.mu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New(errors.ErrInternal, "SSE not supported"))
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", `{"event":"connected"}`)
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, "data: %s\n\n", `{"event":"ping"}`)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) broadcastProjectSSE(projectID string, data map[string]interface{}) {
	b, _ := json.Marshal(data)
	s.mu.Lock()
	clients, ok := s.sseClients[projectID]
	s.mu.Unlock()
	if !ok {
		return
	}
	for _, ch := range clients {
		select {
		case ch <- b:
		default:
		}
	}
}
