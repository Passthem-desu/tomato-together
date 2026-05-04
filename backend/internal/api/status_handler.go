package api

import (
	"net/http"
	"time"

	"tomatogether/backend/internal/service"
)

func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	var req struct {
		RoomName string `json:"room_name"`
		Emoji    string `json:"emoji"`
		Message  string `json:"message"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if len(req.Message) > 50 {
		s.errorResponse(w, http.StatusBadRequest, "message_too_long")
		return
	}

	roomService := s.roomService.(*service.RoomService)
	room, err := roomService.GetRoomByName(req.RoomName)
	if err != nil || room == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	statusService := s.statusService.(*service.UserStatusService)
	if err := statusService.UpdateStatus(userID, room.ID, req.Emoji, req.Message); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "update_status_failed")
		return
	}

	userService := s.userService.(*service.UserService)
	user, _ := userService.GetUserByID(userID)

	s.hub.Broadcast(room.ID, "status_updated", map[string]interface{}{
		"user_id":  userID,
		"username": user.Username,
		"emoji":    req.Emoji,
		"message":  req.Message,
	})

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"emoji":      req.Emoji,
		"message":   req.Message,
		"updated_at": time.Now().Format(time.RFC3339),
	}))
}

func (s *Server) handleDeleteStatus(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	var req struct {
		RoomName string `json:"room_name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}

	roomService := s.roomService.(*service.RoomService)
	room, err := roomService.GetRoomByName(req.RoomName)
	if err != nil || room == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	statusService := s.statusService.(*service.UserStatusService)
	if err := statusService.ClearStatus(userID, room.ID); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "clear_status_failed")
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]string{
		"message": "状态已清除",
	}))
}

func (s *Server) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	// Notifications are not persisted, return empty list for now
	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"notifications": []interface{}{},
	}))
}