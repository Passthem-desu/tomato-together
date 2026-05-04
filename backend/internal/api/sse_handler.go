package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/google/uuid"

	"tomatogether/backend/internal/service"
	"tomatogether/backend/internal/sse"
)

func (s *Server) handleRoomSSE(w http.ResponseWriter, r *http.Request) {
	roomName := mux.Vars(r)["name"]
	if roomName == "" {
		http.Error(w, "room name required", http.StatusBadRequest)
		return
	}

	roomService := s.roomService.(*service.RoomService)
	room, err := roomService.GetRoomByName(roomName)
	if err != nil || room == nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	token := r.URL.Query().Get("token")
	authService := s.authService.(*service.AuthService)
	userID := ""
	level := 1

	if token != "" {
		roomToken, err := authService.ValidateRoomToken(token)
		if err == nil && roomToken != nil {
			userID = roomToken.UserID
			level = 2
		}
	}

	client := &sse.Client{
		ID:     uuid.New().String(),
		UserID: userID,
		RoomID: room.ID,
		Token:  token,
		Level:  level,
		Chan:   make(chan []byte, 256),
		Hub:    s.hub,
	}

	s.hub.Register(client)
	defer s.hub.Unregister(client)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	client.Chan <- formatSSEEvent("connected", map[string]interface{}{
		"room_id": room.ID,
		"level":   level,
	})

	done := r.Context().Done()

	for {
		select {
		case <-done:
			return
		case msg, ok := <-client.Chan:
			if !ok {
				return
			}
			w.Write(msg)
			flusher.Flush()

			if level == 2 && token != "" {
				authService.UpdateHeartbeat(token)
			}
		}
	}
}

func formatSSEEvent(event string, data interface{}) []byte {
	jsonData, _ := json.Marshal(data)
	return []byte("event: " + event + "\ndata: " + string(jsonData) + "\n\n")
}

var _ = time.Now