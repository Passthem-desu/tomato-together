package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"tomatogether/backend/internal/service"
)

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	var req struct {
		Name         string `json:"name"`
		RoomPassword string `json:"room_password"`
		IsReadonly   bool   `json:"is_readonly"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Name == "" {
		s.errorResponse(w, http.StatusBadRequest, "name_required")
		return
	}

	roomService := s.roomService.(*service.RoomService)
	room, err := roomService.CreateRoom(req.Name, req.RoomPassword, req.IsReadonly, claims.UserID)
	if err != nil {
		if err == service.ErrRoomNameTaken {
			s.errorResponse(w, http.StatusConflict, "room_name_taken")
			return
		}
		s.errorResponse(w, http.StatusInternalServerError, "create_room_failed")
		return
	}

	s.jsonResponse(w, http.StatusCreated, formatSuccess(map[string]interface{}{
		"id":           room.ID,
		"name":         room.Name,
		"is_readonly":  room.IsReadonly,
		"has_password": room.HasPassword,
		"created_at":   room.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}))
}

func (s *Server) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	roomName := getRoomName(r)
	if roomName == "" {
		s.errorResponse(w, http.StatusBadRequest, "room_name_required")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Username == "" {
		s.errorResponse(w, http.StatusBadRequest, "username_required")
		return
	}

	userService := s.userService.(*service.UserService)
	roomService := s.roomService.(*service.RoomService)

	// Check if username is taken
	exists, _ := userService.UsernameExists(req.Username)
	if exists {
		s.errorResponse(w, http.StatusConflict, "username_taken")
		return
	}

	// Create or get user
	user, err := userService.CreateUser(req.Username, "")
	if err != nil {
		// User might already exist, try to get it
		user, err = userService.GetUserByUsername(req.Username)
		if err != nil || user == nil {
			s.errorResponse(w, http.StatusInternalServerError, "join_room_failed")
			return
		}
	}

	// Join room
	token, err := roomService.JoinRoom(roomName, req.Username, req.Password, user.ID)
	if err != nil {
		switch err {
		case service.ErrRoomNotFound:
			s.errorResponse(w, http.StatusNotFound, "room_not_found")
		case service.ErrInvalidPassword:
			s.errorResponse(w, http.StatusUnauthorized, "invalid_password")
		case service.ErrAlreadyInRoom:
			s.errorResponse(w, http.StatusBadRequest, "already_in_room")
		default:
			s.errorResponse(w, http.StatusInternalServerError, "join_room_failed")
		}
		return
	}

	room, _ := roomService.GetRoomByName(roomName)
	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"room": map[string]interface{}{
			"id":           room.ID,
			"name":         room.Name,
			"is_readonly":  room.IsReadonly,
			"has_password": room.HasPassword,
		},
		"user": map[string]string{
			"id":       user.ID,
			"username": user.Username,
		},
		"token": token.Token,
	}))
}

func (s *Server) handleLeaveRoom(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	roomName := getRoomName(r)
	roomService := s.roomService.(*service.RoomService)

	if err := roomService.LeaveRoom(roomName, userID); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "leave_room_failed")
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]string{
		"message": "已离开房间",
	}))
}

func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	roomName := getRoomName(r)
	roomService := s.roomService.(*service.RoomService)

	info, err := roomService.GetRoomInfo(roomName)
	if err != nil || info == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"id":           info.ID,
		"name":         info.Name,
		"is_readonly":  info.IsReadonly,
		"has_password": info.HasPassword,
		"member_count": info.MemberCount,
		"created_at":   info.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"owner":        info.Owner,
	}))
}

func (s *Server) handleGetRoomUsers(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	roomName := getRoomName(r)
	roomService := s.roomService.(*service.RoomService)

	room, err := roomService.GetRoomByName(roomName)
	if err != nil || room == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	members, _ := roomService.GetRoomMembers(room.ID)
	userService := s.userService.(*service.UserService)
	statusService := s.statusService.(*service.UserStatusService)
	pomodoroService := s.pomodoroService.(*service.PomodoroService)

	users := make([]map[string]interface{}, 0)
	for _, member := range members {
		user, _ := userService.GetUserByID(member.UserID)
		if user == nil {
			continue
		}

		status, _ := statusService.GetStatus(member.UserID, room.ID)
		session, _ := pomodoroService.GetActiveSession(member.UserID)

		userInfo := map[string]interface{}{
			"id":        user.ID,
			"username":  user.Username,
			"is_owner":  member.IsOwner,
			"is_online": s.hub.IsUserOnline(room.ID, user.ID),
		}

		if status != nil {
			userInfo["status"] = map[string]string{
				"emoji":   status.Emoji,
				"message": status.Message,
			}
		}

		if session != nil {
			pomodoro := map[string]interface{}{
				"is_active": true,
			}
			if session.IsFollowed && session.LeaderID != nil {
				leader, _ := userService.GetUserByID(*session.LeaderID)
				if leader != nil {
					pomodoro["is_following"] = true
					pomodoro["leader_username"] = leader.Username
				}
			}
			pomodoro["started_at"] = session.StartedAt.Format("2006-01-02T15:04:05Z")
			userInfo["pomodoro"] = pomodoro
		}

		users = append(users, userInfo)
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"users": users,
	}))
}

func (s *Server) handleUpdateRoomSettings(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	roomName := getRoomName(r)
	roomService := s.roomService.(*service.RoomService)

	room, err := roomService.GetRoomByName(roomName)
	if err != nil || room == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	// Check if user is owner
	member, err := roomService.GetRoomMember(room.ID, userID)
	if err != nil || member == nil || !member.IsOwner {
		s.errorResponse(w, http.StatusForbidden, "not_room_owner")
		return
	}

	var req struct {
		RoomPassword *string `json:"room_password"`
		IsReadonly   *bool   `json:"is_readonly"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := roomService.UpdateRoomSettings(roomName, req.RoomPassword, req.IsReadonly); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "update_settings_failed")
		return
	}

	// Broadcast settings change
	s.hub.Broadcast(room.ID, "room_settings_changed", map[string]interface{}{
		"is_readonly":  room.IsReadonly,
		"changed_by":   member.UserID,
	})

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]string{
		"message": "设置已更新",
	}))
}

func (s *Server) handleSetOwner(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	roomName := getRoomName(r)
	roomService := s.roomService.(*service.RoomService)

	room, err := roomService.GetRoomByName(roomName)
	if err != nil || room == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	// Check if current user is owner
	member, err := roomService.GetRoomMember(room.ID, userID)
	if err != nil || member == nil || !member.IsOwner {
		s.errorResponse(w, http.StatusForbidden, "not_room_owner")
		return
	}

	vars := mux.Vars(r)
	targetUserID := vars["user_id"]
	if targetUserID == userID {
		s.errorResponse(w, http.StatusBadRequest, "cannot_remove_self_owner")
		return
	}

	var req struct {
		IsOwner bool `json:"is_owner"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}

	// Check if target is persistent user
	userService := s.userService.(*service.UserService)
	targetUser, _ := userService.GetUserByID(targetUserID)
	if targetUser == nil || !targetUser.IsPersistent() {
		s.errorResponse(w, http.StatusForbidden, "must_be_persistent_user")
		return
	}

	if err := roomService.SetOwner(roomName, targetUserID, req.IsOwner); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "set_owner_failed")
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]string{
		"message": "房主设置已更新",
	}))
}

func (s *Server) handleGetRoomStats(w http.ResponseWriter, r *http.Request) {
	roomName := getRoomName(r)
	roomService := s.roomService.(*service.RoomService)

	room, err := roomService.GetRoomByName(roomName)
	if err != nil || room == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	pomodoroService := s.pomodoroService.(*service.PomodoroService)
	stats, err := pomodoroService.GetRoomStats(room.ID, "2006-01-02") // TODO: use actual date
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "get_stats_failed")
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"total_pomodoros": stats.TotalPomodoros,
		"total_duration":  stats.TotalDuration,
		"active_users":    stats.ActiveUsers,
	}))
}

func (s *Server) handleCreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	roomName := getRoomName(r)
	roomService := s.roomService.(*service.RoomService)
	announcementService := s.announcementService.(*service.AnnouncementService)

	room, err := roomService.GetRoomByName(roomName)
	if err != nil || room == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	// Check if user is owner
	isOwner, _ := announcementService.IsRoomOwner(room.ID, userID)
	if !isOwner {
		s.errorResponse(w, http.StatusForbidden, "not_room_owner")
		return
	}

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Title == "" {
		s.errorResponse(w, http.StatusBadRequest, "title_required")
		return
	}

	announcement, err := announcementService.CreateAnnouncement(room.ID, userID, req.Title, req.Body)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "create_announcement_failed")
		return
	}

	// Broadcast announcement
	userService := s.userService.(*service.UserService)
	user, _ := userService.GetUserByID(userID)
	s.hub.Broadcast(room.ID, "announcement", map[string]interface{}{
		"id":    announcement.ID,
		"title": announcement.Title,
		"body":  announcement.Body,
		"from":  user.Username,
	})

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"message":          "公告已发送",
		"announcement_id": announcement.ID,
	}))
}

func (s *Server) handleGetAnnouncements(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	roomName := getRoomName(r)
	roomService := s.roomService.(*service.RoomService)
	announcementService := s.announcementService.(*service.AnnouncementService)

	room, err := roomService.GetRoomByName(roomName)
	if err != nil || room == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		// Parse limit
	}

	announcements, err := announcementService.GetAnnouncementsByRoom(room.ID, limit)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "get_announcements_failed")
		return
	}

	userService := s.userService.(*service.UserService)
	result := make([]map[string]interface{}, 0)
	for _, a := range announcements {
		sender, _ := userService.GetUserByID(a.SenderID)
		item := map[string]interface{}{
			"id":         a.ID,
			"title":      a.Title,
			"body":       a.Body,
			"created_at": a.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if sender != nil {
			item["sender"] = map[string]string{
				"id":       sender.ID,
				"username": sender.Username,
			}
		}
		result = append(result, item)
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"announcements": result,
	}))
}

func getRoomName(r *http.Request) string {
	vars := mux.Vars(r)
	return vars["name"]
}

func init() {
	_ = json.Unmarshal
}