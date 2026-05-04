package api

import (
	"net/http"
	"time"

	"tomatogether/backend/internal/service"
)

func (s *Server) handleStartPomodoro(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	var req struct {
		RoomName              string  `json:"room_name"`
		ProjectID             *string `json:"project_id"`
		TaskID                *string `json:"task_id"`
		PlannedDuration       int     `json:"planned_duration"`
		RestDuration          int     `json:"rest_duration"`
		LongBreakDuration     int     `json:"long_break_duration"`
		SessionsBeforeLongBreak int    `json:"sessions_before_long_break"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}

	// Set defaults
	if req.PlannedDuration == 0 {
		req.PlannedDuration = 1500
	}
	if req.RestDuration == 0 {
		req.RestDuration = 300
	}
	if req.LongBreakDuration == 0 {
		req.LongBreakDuration = 900
	}
	if req.SessionsBeforeLongBreak == 0 {
		req.SessionsBeforeLongBreak = 4
	}

	roomService := s.roomService.(*service.RoomService)
	room, err := roomService.GetRoomByName(req.RoomName)
	if err != nil || room == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	pomodoroService := s.pomodoroService.(*service.PomodoroService)
	session, err := pomodoroService.StartPomodoro(userID, room.ID, req.ProjectID, req.TaskID, req.PlannedDuration, req.RestDuration, req.LongBreakDuration, req.SessionsBeforeLongBreak)
	if err != nil {
		if err == service.ErrAlreadyFollowing {
			s.errorResponse(w, http.StatusBadRequest, "already_in_session")
		} else {
			s.errorResponse(w, http.StatusInternalServerError, "start_pomodoro_failed")
		}
		return
	}

	userService := s.userService.(*service.UserService)
	user, _ := userService.GetUserByID(userID)
	sessionsToday, _ := pomodoroService.GetTodayCount(userID)

	// Broadcast pomodoro started
	s.hub.Broadcast(room.ID, "pomodoro_started", map[string]interface{}{
		"user_id":     userID,
		"username":    user.Username,
		"session_id":  session.ID,
		"started_at":  session.StartedAt.Format(time.RFC3339),
	})

	s.jsonResponse(w, http.StatusCreated, formatSuccess(map[string]interface{}{
		"session_id":               session.ID,
		"started_at":               session.StartedAt.Format(time.RFC3339),
		"planned_duration":         req.PlannedDuration,
		"rest_duration":           req.RestDuration,
		"long_break_duration":      req.LongBreakDuration,
		"sessions_before_long_break": req.SessionsBeforeLongBreak,
		"status":                  "focusing",
		"sessions_today":          sessionsToday,
	}))
}

func (s *Server) handleFollowPomodoro(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	var req struct {
		RoomName  string `json:"room_name"`
		LeaderID  string `json:"leader_id"`
	}
	if err := decodeJSON(r, &req); err != nil || req.LeaderID == "" {
		s.errorResponse(w, http.StatusBadRequest, "leader_id_required")
		return
	}

	roomService := s.roomService.(*service.RoomService)
	room, err := roomService.GetRoomByName(req.RoomName)
	if err != nil || room == nil {
		s.errorResponse(w, http.StatusNotFound, "room_not_found")
		return
	}

	pomodoroService := s.pomodoroService.(*service.PomodoroService)
	session, err := pomodoroService.FollowPomodoro(userID, room.ID, req.LeaderID)
	if err != nil {
		switch err {
		case service.ErrNoActiveSession:
			s.errorResponse(w, http.StatusBadRequest, "no_active_session")
		case service.ErrAlreadyFollowing:
			s.errorResponse(w, http.StatusBadRequest, "already_following")
		default:
			s.errorResponse(w, http.StatusInternalServerError, "follow_pomodoro_failed")
		}
		return
	}

	// Get leader session info
	leaderSession, _ := pomodoroService.GetActiveSession(req.LeaderID)
	userService := s.userService.(*service.UserService)
	user, _ := userService.GetUserByID(userID)
	leader, _ := userService.GetUserByID(req.LeaderID)

	// Broadcast
	s.hub.Broadcast(room.ID, "pomodoro_followed", map[string]interface{}{
		"user_id":         userID,
		"username":        user.Username,
		"leader_id":       req.LeaderID,
		"leader_username": leader.Username,
	})

	if leaderSession != nil {
		remaining := int(time.Since(leaderSession.StartedAt).Seconds())
		if remaining < 0 {
			remaining = leaderSession.PlannedDuration
		} else {
			remaining = leaderSession.PlannedDuration - remaining
		}
		s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
			"session_id":               session.ID,
			"leader_id":                req.LeaderID,
			"leader_username":          leader.Username,
			"started_at":               leaderSession.StartedAt.Format(time.RFC3339),
			"remaining_seconds":        remaining,
			"planned_duration":        leaderSession.PlannedDuration,
			"status":                  "following",
		}))
	} else {
		s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
			"session_id":      session.ID,
			"leader_id":       req.LeaderID,
			"leader_username": leader.Username,
			"status":          "following",
		}))
	}
}

func (s *Server) handleUnfollowPomodoro(w http.ResponseWriter, r *http.Request) {
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
	room, _ := roomService.GetRoomByName(req.RoomName)

	pomodoroService := s.pomodoroService.(*service.PomodoroService)
	if err := pomodoroService.UnfollowPomodoro(userID); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "not_following")
		return
	}

	if room != nil {
		userService := s.userService.(*service.UserService)
		user, _ := userService.GetUserByID(userID)
		s.hub.Broadcast(room.ID, "pomodoro_unfollowed", map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
		})
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"message": "已取消跟随",
		"status": "idle",
	}))
}

func (s *Server) handleEndPomodoro(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	var req struct {
		RoomName string `json:"room_name"`
		Aborted  bool   `json:"aborted"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}

	pomodoroService := s.pomodoroService.(*service.PomodoroService)
	session, err := pomodoroService.EndPomodoro(userID, req.Aborted)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "no_active_session")
		return
	}

	roomService := s.roomService.(*service.RoomService)
	room, _ := roomService.GetRoomByName(req.RoomName)
	userService := s.userService.(*service.UserService)
	user, _ := userService.GetUserByID(userID)

	if room != nil {
		s.hub.Broadcast(room.ID, "pomodoro_ended", map[string]interface{}{
			"user_id":    userID,
			"username":   user.Username,
			"session_id": session.ID,
			"duration":   session.Duration,
			"status":     "rest",
		})

		// If aborted, notify followers
		if req.Aborted {
			s.hub.Broadcast(room.ID, "leader_aborted", map[string]interface{}{
				"leader_id":    userID,
				"leader_username": user.Username,
				"message":      "主导者已提前结束番茄",
			})
		}
	}

	sessionsToday, _ := pomodoroService.GetTodayCount(userID)
	shouldLongBreak := sessionsToday > 0 && sessionsToday%4 == 0

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"session_id":             session.ID,
		"duration":               session.Duration,
		"planned_duration":       session.PlannedDuration,
		"is_followed":           session.IsFollowed,
		"status":                 "rest",
		"rest_duration":          300,  // Default rest duration
		"long_break_duration":    900, // Default long break duration
		"should_take_long_break": shouldLongBreak,
		"sessions_completed":     sessionsToday,
	}))
}

func (s *Server) handleGetPomodoroStatus(w http.ResponseWriter, r *http.Request) {
	userID := s.validateL2Token(r)
	if userID == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	pomodoroService := s.pomodoroService.(*service.PomodoroService)
	session, err := pomodoroService.GetActiveSession(userID)
	if err != nil || session == nil {
		s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
			"is_active": false,
			"status":    "idle",
		}))
		return
	}

	remaining := int(time.Since(session.StartedAt).Seconds())
	if remaining < 0 {
		remaining = session.PlannedDuration
	} else {
		remaining = session.PlannedDuration - remaining
	}

	response := map[string]interface{}{
		"is_active":          true,
		"status":            "focusing",
		"session_id":        session.ID,
		"started_at":        session.StartedAt.Format(time.RFC3339),
		"remaining_seconds": remaining,
	}

	if session.IsFollowed && session.LeaderID != nil {
		userService := s.userService.(*service.UserService)
		leader, _ := userService.GetUserByID(*session.LeaderID)
		if leader != nil {
			response["status"] = "following"
			response["leader_id"] = *session.LeaderID
			response["leader_username"] = leader.Username
		}
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(response))
}