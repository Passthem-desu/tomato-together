package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"tomatogether/backend/internal/middleware"
	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/service"
	"tomatogether/backend/internal/sse"
)

type contextKey string

const tokenContextKey contextKey = "token"

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// SSE stream endpoint (no auth middleware, handles own auth via query param)
	r.HandleFunc("/rooms/{name}/sse", h.HandleSSE).Methods(http.MethodGet)

	// L1 Routes with auth rate limiting (general 60/min)
	r.HandleFunc("/rooms/{name}", h.GetRoomInfo).Methods(http.MethodGet)
	r.HandleFunc("/rooms/{name}/stats", h.GetRoomStats).Methods(http.MethodGet)

	// Sensitive L1 routes: stricter rate limit (10/min per IP)
	sensitiveRouter := r.NewRoute().Subrouter()
	sensitiveRouter.Use(middleware.AuthRateLimit())
	sensitiveRouter.HandleFunc("/rooms/{name}/join", h.JoinRoom).Methods(http.MethodPost)
	sensitiveRouter.HandleFunc("/rooms/{name}/check-user", h.CheckUser).Methods(http.MethodPost)
	sensitiveRouter.HandleFunc("/rooms/{name}/check-password", h.CheckRoomPassword).Methods(http.MethodPost)
	sensitiveRouter.HandleFunc("/auth/login", h.Login).Methods(http.MethodPost)
	sensitiveRouter.HandleFunc("/auth/refresh", h.RefreshAccessToken).Methods(http.MethodPost)
	sensitiveRouter.HandleFunc("/rooms", h.CreateRoom).Methods(http.MethodPost)

	// L2 Routes (require token, with general rate limiting)
	authRouter := r.PathPrefix("").Subrouter()
	authRouter.Use(h.authMiddleware)
	authRouter.Use(middleware.GeneralRateLimit())

	authRouter.HandleFunc("/rooms/{name}/leave", h.LeaveRoom).Methods(http.MethodPost)
	authRouter.HandleFunc("/rooms/{name}/users", h.GetRoomUsers).Methods(http.MethodGet)
	authRouter.HandleFunc("/rooms/{name}/settings", h.UpdateRoomSettings).Methods(http.MethodPut)
	authRouter.HandleFunc("/rooms/{name}/owners/{member_id}", h.SetRoomOwner).Methods(http.MethodPut)
	authRouter.HandleFunc("/rooms/{name}/members/{member_id}", h.KickMember).Methods(http.MethodDelete)
	authRouter.HandleFunc("/rooms/{name}/announcements", h.CreateAnnouncement).Methods(http.MethodPost)
	authRouter.HandleFunc("/rooms/{name}/announcements", h.GetAnnouncements).Methods(http.MethodGet)
	authRouter.HandleFunc("/rooms/{name}/announcements/{id}", h.DeleteAnnouncement).Methods(http.MethodDelete)
	authRouter.HandleFunc("/auth/me", h.GetMe).Methods(http.MethodGet)
	authRouter.HandleFunc("/auth/upgrade", h.UpgradeToPersistent).Methods(http.MethodPost)
	authRouter.HandleFunc("/auth/password", h.ChangePassword).Methods(http.MethodPut)
	authRouter.HandleFunc("/auth/logout", h.Logout).Methods(http.MethodPost)
	authRouter.HandleFunc("/auth/tokens", h.RevokeAllTokens).Methods(http.MethodDelete)
	authRouter.HandleFunc("/pomodoro/start", h.StartPomodoro).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/follow", h.FollowPomodoro).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/unfollow", h.UnfollowPomodoro).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/end", h.EndPomodoro).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/pause", h.PausePomodoro).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/resume", h.ResumePomodoro).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/skip-rest", h.SkipRest).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/status", h.GetPomodoroStatus).Methods(http.MethodGet)
	authRouter.HandleFunc("/status", h.UpdateStatus).Methods(http.MethodPut)
	authRouter.HandleFunc("/status", h.DeleteStatus).Methods(http.MethodDelete)
	authRouter.HandleFunc("/tags", h.GetTags).Methods(http.MethodGet)
	authRouter.HandleFunc("/tags", h.CreateTag).Methods(http.MethodPost)
	authRouter.HandleFunc("/tags/{id}", h.UpdateTag).Methods(http.MethodPut)
	authRouter.HandleFunc("/tags/{id}", h.DeleteTag).Methods(http.MethodDelete)
	authRouter.HandleFunc("/tasks", h.GetTasks).Methods(http.MethodGet)
	authRouter.HandleFunc("/tasks", h.CreateTask).Methods(http.MethodPost)
	authRouter.HandleFunc("/tasks/{id}", h.UpdateTask).Methods(http.MethodPut)
	authRouter.HandleFunc("/tasks/{id}", h.DeleteTask).Methods(http.MethodDelete)
	authRouter.HandleFunc("/tasks/sync", h.SyncTasks).Methods(http.MethodPost)
	authRouter.HandleFunc("/stats/me", h.GetMyStats).Methods(http.MethodGet)
	authRouter.HandleFunc("/stats/me", h.ResetMyStats).Methods(http.MethodDelete)
}

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.writeError(w, http.StatusUnauthorized, "token_missing")
			return
		}

		tokenValue := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenValue == authHeader {
			h.writeError(w, http.StatusUnauthorized, "token_invalid")
			return
		}

		tokenInfo, err := h.svc.ValidateToken(tokenValue)
		if err != nil {
			h.writeError(w, http.StatusUnauthorized, err.Error())
			return
		}

		ctx := SetTokenContext(r.Context(), tokenInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Context helpers

func SetTokenContext(ctx context.Context, token *models.TokenInfo) context.Context {
	return context.WithValue(ctx, tokenContextKey, token)
}

func GetTokenFromContext(ctx context.Context) *models.TokenInfo {
	if token := ctx.Value(tokenContextKey); token != nil {
		return token.(*models.TokenInfo)
	}
	return nil
}

// Helper functions

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err})
}

func (h *Handler) parseJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// writeSSEEvent writes a single SSE event to the response writer
func (h *Handler) writeSSEEvent(w http.ResponseWriter, event string, data interface{}) {
	dataJSON, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(dataJSON))
	w.(http.Flusher).Flush()
}

// L1 Handlers

func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRoomRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	// Must provide password to create room (become persistent user)
	if req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "password_required")
		return
	}

	resp, err := h.svc.CreateRoom(&req)
	if err != nil {
		log.Printf("CreateRoom error: %v", err)
		if err == service.ErrRoomNameTaken {
			h.writeError(w, http.StatusConflict, err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomName := vars["name"]

	var req models.JoinRoomRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	resp, err := h.svc.JoinRoom(roomName, &req)
	if err != nil {
		switch err {
		case service.ErrRoomNotFound:
			h.writeError(w, http.StatusNotFound, err.Error())
		case service.ErrInvalidRoomPassword:
			h.writeError(w, http.StatusUnauthorized, err.Error())
		case service.ErrInvalidPassword:
			h.writeError(w, http.StatusUnauthorized, err.Error())
		case service.ErrUsernameTaken:
			h.writeError(w, http.StatusConflict, err.Error())
		case service.ErrRoomIsReadonly:
			h.writeError(w, http.StatusForbidden, err.Error())
		default:
			h.writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) GetRoomInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomName := vars["name"]

	room, err := h.svc.GetRoomInfo(roomName)
	if err != nil {
		if err == service.ErrRoomNotFound {
			h.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	owner, _ := h.svc.GetRoomOwner(room.ID)
	memberCount, _ := h.svc.GetRoomMemberCount(room.ID)

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":           room.ID,
			"name":         room.Name,
			"is_readonly":  room.IsReadonly,
			"has_password": room.HasPassword,
			"member_count": memberCount,
			"created_at":   room.CreatedAt.Format(time.RFC3339),
			"owner": map[string]interface{}{
				"username": owner.Username,
			},
		},
	})
}

func (h *Handler) GetRoomStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomName := vars["name"]

	room, err := h.svc.GetRoomInfo(roomName)
	if err != nil {
		h.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	totalPomodoros, totalDuration, activeUsers, _ := h.svc.GetRoomStats(room.ID, time.Now())

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"date":            time.Now().Format("2006-01-02"),
			"total_pomodoros": totalPomodoros,
			"total_duration":  totalDuration,
			"active_users":    activeUsers,
		},
	})
}

func (h *Handler) CheckUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomName := vars["name"]

	var req struct {
		Username string `json:"username"`
	}
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	userStatus, err := h.svc.CheckUsername(roomName, req.Username)
	if err != nil {
		h.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    userStatus,
	})
}

func (h *Handler) CheckRoomPassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomName := vars["name"]

	var req struct {
		RoomPassword string `json:"room_password"`
	}
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	valid, err := h.svc.CheckRoomPassword(roomName, req.RoomPassword)
	if err != nil {
		h.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    map[string]interface{}{"valid": valid},
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	resp, err := h.svc.Login(&req)
	if err != nil {
		switch err {
		case service.ErrRoomNotFound:
			h.writeError(w, http.StatusNotFound, err.Error())
		case service.ErrInvalidPassword, service.ErrInvalidRoomPassword:
			h.writeError(w, http.StatusUnauthorized, err.Error())
		default:
			h.writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

// L2 Handlers

func (h *Handler) KickMember(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	memberID := vars["member_id"]

	if err := h.svc.KickMember(token.Token, memberID); err != nil {
		switch err {
		case service.ErrMustBeOwner:
			h.writeError(w, http.StatusForbidden, err.Error())
		case service.ErrMemberNotFound:
			h.writeError(w, http.StatusNotFound, err.Error())
		default:
			h.writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"message": "成员已被移出房间",
		},
	})
}

func (h *Handler) LeaveRoom(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	if err := h.svc.LeaveRoom(token.Token); err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"message": "已离开房间",
		},
	})
}

func (h *Handler) GetRoomUsers(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	users, err := h.svc.GetRoomUsers(token.Token)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"users": users,
		},
	})
}

func (h *Handler) UpdateRoomSettings(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.UpdateRoomSettingsRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	err := h.svc.UpdateRoomSettings(token.Token, &req)
	if err != nil {
		if err == service.ErrMustBeOwner {
			h.writeError(w, http.StatusForbidden, err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"message": "设置已更新",
		},
	})
}

func (h *Handler) SetRoomOwner(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	memberID := vars["member_id"]

	var req models.SetOwnerRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	err := h.svc.SetRoomOwner(token.Token, memberID, &req)
	if err != nil {
		switch err {
		case service.ErrMustBeOwner:
			h.writeError(w, http.StatusForbidden, err.Error())
		case service.ErrMustBePersistent:
			h.writeError(w, http.StatusForbidden, err.Error())
		default:
			h.writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"message": "房主设置已更新",
		},
	})
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	member, err := h.svc.GetMe(token.Token)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":            member.ID,
			"username":      member.Username,
			"is_owner":      member.IsOwner,
			"is_persistent": member.IsPersistent,
			"joined_at":     member.JoinedAt.Format(time.RFC3339),
		},
	})
}

func (h *Handler) UpgradeToPersistent(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.UpgradeRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	err := h.svc.UpgradeToPersistent(token.Token, &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	member, _ := h.svc.GetMe(token.Token)

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"message": "已升级为持久化用户",
			"member": map[string]interface{}{
				"id":            member.ID,
				"username":      member.Username,
				"is_persistent": true,
			},
		},
	})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.ChangePasswordRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := h.svc.ChangePassword(token.Token, &req); err != nil {
		switch err {
		case service.ErrInvalidPassword:
			h.writeError(w, http.StatusUnauthorized, err.Error())
		case service.ErrMustBePersistent:
			h.writeError(w, http.StatusBadRequest, err.Error())
		case service.ErrPasswordTooShort:
			h.writeError(w, http.StatusBadRequest, err.Error())
		case service.ErrFieldTooLong:
			h.writeError(w, http.StatusBadRequest, err.Error())
		default:
			h.writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"message": "密码已更新",
		},
	})
}

func (h *Handler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := h.parseJSON(r, &req); err != nil || req.RefreshToken == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	accessToken, refreshToken, expiresIn, err := h.svc.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		switch err {
		case service.ErrInvalidRefreshToken:
			h.writeError(w, http.StatusUnauthorized, "invalid_refresh_token")
		case service.ErrRefreshTokenRevoked:
			h.writeError(w, http.StatusUnauthorized, "refresh_token_revoked")
		case service.ErrRefreshTokenExpired:
			h.writeError(w, http.StatusUnauthorized, "refresh_token_expired")
		default:
			h.writeError(w, http.StatusUnauthorized, "invalid_refresh_token")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"expires_in":    expiresIn,
		},
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := h.parseJSON(r, &req); err == nil && req.RefreshToken != "" {
		h.svc.RevokeRefreshTokenByValue(req.RefreshToken)
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    map[string]string{"message": "logged_out"},
	})
}

func (h *Handler) RevokeAllTokens(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	count, err := h.svc.RevokeAllRefreshTokens(token.MemberID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    map[string]interface{}{"revoked_count": count},
	})
}

// Pomodoro handlers

func (h *Handler) StartPomodoro(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.StartPomodoroRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	resp, err := h.svc.StartPomodoro(token.Token, &req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) FollowPomodoro(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.FollowPomodoroRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	resp, err := h.svc.FollowPomodoro(token.Token, &req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) UnfollowPomodoro(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.FollowRoomRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	status, err := h.svc.UnfollowPomodoro(token.Token, &req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"message": "已取消跟随",
			"status":  status,
		},
	})
}

func (h *Handler) EndPomodoro(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.EndPomodoroRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	resp, err := h.svc.EndPomodoro(token.Token, &req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) GetPomodoroStatus(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	resp, err := h.svc.GetPomodoroStatus(token.Token)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

// Status handlers

// Pause / Resume / SkipRest handlers

func (h *Handler) PausePomodoro(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	resp, err := h.svc.PausePomodoro(token.Token)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) ResumePomodoro(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	resp, err := h.svc.ResumePomodoro(token.Token)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) SkipRest(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	err := h.svc.SkipRest(token.Token)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"phase": "idle",
		},
	})
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.UpdateStatusRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	status, err := h.svc.UpdateStatus(token.Token, &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"emoji":      status.Emoji,
			"message":    status.Message,
			"updated_at": status.UpdatedAt.Format(time.RFC3339),
		},
	})
}

func (h *Handler) DeleteStatus(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	if err := h.svc.DeleteStatus(token.Token); err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"message": "状态已清除",
		},
	})
}

// Tag handlers

func (h *Handler) GetTags(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	tags, err := h.svc.GetTags(token.MemberID, token.RoomID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"tags": tags,
		},
	})
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.CreateTagRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	tag, err := h.svc.CreateTag(token.MemberID, token.RoomID, req.Name)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    tag,
	})
}

func (h *Handler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	tagID := vars["id"]

	var req models.UpdateTagRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := h.svc.UpdateTag(tagID, token.MemberID, req.Name); err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":   tagID,
			"name": req.Name,
		},
	})
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	tagID := vars["id"]

	if err := h.svc.DeleteTag(tagID, token.MemberID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"message": "标签已删除",
		},
	})
}

// Task handlers

func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	tasks, err := h.svc.GetTasks(token.MemberID, token.RoomID, r.URL.Query().Get("status"))
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"tasks": tasks,
		},
	})
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.CreateTaskRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	task, err := h.svc.CreateTask(token.MemberID, token.RoomID, req.ClientID, req.Title, req.TagID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    task,
	})
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	taskID := vars["id"]

	var req models.UpdateTaskRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := h.svc.UpdateTask(taskID, token.MemberID, req.Title, req.Status, req.TagID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id": taskID,
		},
	})
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	taskID := vars["id"]

	if err := h.svc.DeleteTask(taskID, token.MemberID); err != nil {
		if err.Error() == "task_not_found" {
			h.writeError(w, http.StatusNotFound, "task_not_found")
		} else {
			h.writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"message": "WIP 已删除",
		},
	})
}

func (h *Handler) SyncTasks(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.SyncTasksRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	resp, err := h.svc.SyncTasks(token.MemberID, token.RoomID, req.Tasks)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) GetMyStats(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	totalPomodoros, totalDuration, err := h.svc.GetMemberStats(token.MemberID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"total_pomodoros": totalPomodoros,
			"total_duration":  totalDuration,
		},
	})
}

func (h *Handler) ResetMyStats(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}
	if err := h.svc.ResetMemberStats(token.MemberID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    map[string]string{"message": "统计数据已重置"},
	})
}

// Announcement handlers

func (h *Handler) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	roomName := vars["name"]

	var req models.CreateAnnouncementRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	announcement, err := h.svc.CreateAnnouncement(roomName, token.MemberID, req.Title, req.Body)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"message":         "公告已发送",
			"announcement_id": announcement.ID,
		},
	})
}

func (h *Handler) GetAnnouncements(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	roomName := vars["name"]

	announcements, err := h.svc.GetAnnouncements(roomName, 20)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"announcements": announcements,
		},
	})
}

func (h *Handler) DeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	announcementID := vars["id"]

	if err := h.svc.DeleteAnnouncement(announcementID, token.MemberID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"message": "公告已删除",
		},
	})
}

// SSE Handler for real-time events

func (h *Handler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomName := vars["name"]

	// Get room info to validate room exists
	room, err := h.svc.GetRoomInfo(roomName)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "room_not_found")
		return
	}

	// Check for token in query param or cookie (Sec #11 fix)
	tokenValue := r.URL.Query().Get("token")
	if tokenValue == "" {
		if cookie, err := r.Cookie("sse_token"); err == nil {
			tokenValue = cookie.Value
		}
	}
	var memberID, username string
	var isOwner, isAuth bool

	if tokenValue != "" {
		// Validate token (supports JWT + RoomToken)
		tokenInfo, err := h.svc.ValidateToken(tokenValue)
		if err == nil && tokenInfo.RoomID == room.ID {
			memberID = tokenInfo.MemberID
			username = tokenInfo.Username
			isOwner = tokenInfo.IsOwner
			isAuth = true

			// Update heartbeat
			h.svc.RefreshToken(tokenValue)
		} else if err != nil && tokenValue != "" {
			// Token is invalid or expired - send token_expired and close
			h.writeSSEEvent(w, "token_expired", map[string]interface{}{
				"message": "Token 已过期或无效，请重新加入房间",
			})
			return
		}
	}

	// Create SSE client
	hub := sse.GetHub()
	if hub == nil {
		h.writeError(w, http.StatusInternalServerError, "sse_unavailable")
		return
	}

	// Check SSE connection limits (Sec #6 fix)
	clientIP := r.RemoteAddr
	if !hub.CanConnect(r, roomName, clientIP, isAuth) {
		h.writeError(w, http.StatusServiceUnavailable, "too_many_connections")
		return
	}

	client := hub.NewClient(roomName, memberID, username, isOwner, isAuth, clientIP)
	client.Register()
	defer client.Unregister()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// If not authenticated (L1旁观者), send connected message
	if !isAuth {
		initialData, _ := json.Marshal(map[string]interface{}{
			"message": "旁观模式 - 仅可查看",
		})
		fmt.Fprintf(w, "event: connected\ndata: %s\n\n", string(initialData))
		w.(http.Flusher).Flush()
	}

	// Create heartbeat ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-client.Notify():
			if !ok {
				return
			}
			w.Write(msg)
			w.(http.Flusher).Flush()

		case <-ticker.C:
			// Check if token is still valid (for authenticated clients)
			if isAuth && tokenValue != "" {
				token, err := h.svc.ValidateToken(tokenValue)
				if err != nil {
					// Token expired - notify and close connection
					h.writeSSEEvent(w, "token_expired", map[string]interface{}{
						"message": "Token 已过期，请重新加入房间",
					})
					return
				}

				// Update heartbeat and refresh token
				h.svc.RefreshToken(tokenValue)
				client.Ping()
				_ = token
			}

			// Send ping to keep connection alive
			hub.BroadcastEvent(roomName, "ping", map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
			})

		case <-r.Context().Done():
			return
		}
	}
}
