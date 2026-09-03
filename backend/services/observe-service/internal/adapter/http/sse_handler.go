package http

import (
	"encoding/json"
	"fmt"

	"net/http"
	"time"

	"github.com/vnp-memory/services/observe-service/internal/observe"
)

type SSEHandler struct {
	stream *observe.StreamBroker
}

func NewSSEHandler(stream *observe.StreamBroker) *SSEHandler {
	return &SSEHandler{stream: stream}
}

// ServeSSE handles GET /v1/stream?session_id=<id>
func (h *SSEHandler) ServeSSE(w http.ResponseWriter, r *http.Request) {
	sessionFilter := r.URL.Query().Get("session_id")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch, cancel := h.stream.Subscribe(sessionFilter)
	defer cancel()

	// Send heartbeat every 30s
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()

		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}
