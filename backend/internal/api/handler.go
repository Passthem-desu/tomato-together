package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/service"
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
	// L1 Routes
	r.HandleFunc("/rooms/{name}", h.GetRoomInfo).Methods(http.MethodGet)
	r.HandleFunc("/rooms/{name}/join", h.JoinRoom).Methods(http.MethodPost)
	r.HandleFunc("/rooms/{name}/stats", h.GetRoomStats).Methods(http.MethodGet)
	r.HandleFunc("/rooms/{name}/check-user", h.CheckUser).Methods(http.MethodPost)
	r.HandleFunc("/auth/login", h.Login).Methods(http.MethodPost)

	// L2 Routes (require token)
	authRouter := r.PathPrefix("").Subrouter()
	authRouter.Use(h.authMiddleware)

	authRouter.HandleFunc("/rooms/{name}/leave", h.LeaveRoom).Methods(http.MethodPost)
	authRouter.HandleFunc("/rooms/{name}/users", h.GetRoomUsers).Methods(http.MethodGet)
	authRouter.HandleFunc("/rooms/{name}/settings", h.UpdateRoomSettings).Methods(http.MethodPut)
	authRouter.HandleFunc("/rooms/{name}/owners/{member_id}", h.SetRoomOwner).Methods(http.MethodPut)
	authRouter.HandleFunc("/rooms/{name}/announcements", h.CreateAnnouncement).Methods(http.MethodPost)
	authRouter.HandleFunc("/rooms/{name}/announcements", h.GetAnnouncements).Methods(http.MethodGet)
	authRouter.HandleFunc("/auth/me", h.GetMe).Methods(http.MethodGet)
	authRouter.HandleFunc("/auth/upgrade", h.UpgradeToPersistent).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/start", h.StartPomodoro).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/follow", h.FollowPomodoro).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/unfollow", h.UnfollowPomodoro).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/end", h.EndPomodoro).Methods(http.MethodPost)
	authRouter.HandleFunc("/pomodoro/status", h.GetPomodoroStatus).Methods(http.MethodGet)
	authRouter.HandleFunc("/status", h.UpdateStatus).Methods(http.MethodPut)
	authRouter.HandleFunc("/status", h.DeleteStatus).Methods(http.MethodDelete)
	authRouter.HandleFunc("/projects", h.GetProjects).Methods(http.MethodGet)
	authRouter.HandleFunc("/projects", h.CreateProject).Methods(http.MethodPost)
	authRouter.HandleFunc("/projects/{id}", h.UpdateProject).Methods(http.MethodPut)
	authRouter.HandleFunc("/projects/{id}", h.DeleteProject).Methods(http.MethodDelete)
	authRouter.HandleFunc("/tasks", h.GetTasks).Methods(http.MethodGet)
	authRouter.HandleFunc("/tasks", h.CreateTask).Methods(http.MethodPost)
	authRouter.HandleFunc("/tasks/{id}", h.UpdateTask).Methods(http.MethodPut)
	authRouter.HandleFunc("/tasks/{id}", h.DeleteTask).Methods(http.MethodDelete)
	authRouter.HandleFunc("/tasks/sync", h.SyncTasks).Methods(http.MethodPost)

	// Room creation (requires persistent user - password must be provided)
	r.HandleFunc("/rooms", h.CreateRoom).Methods(http.MethodPost)
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

		token, err := h.svc.ValidateToken(tokenValue)
		if err != nil {
			h.writeError(w, http.StatusUnauthorized, err.Error())
			return
		}

		// Store token in context for later use
		ctx := SetTokenContext(r.Context(), token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Context helpers

func SetTokenContext(ctx context.Context, token *models.RoomToken) context.Context {
	return context.WithValue(ctx, tokenContextKey, token)
}

func GetTokenFromContext(ctx context.Context) *models.RoomToken {
	if token := ctx.Value(tokenContextKey); token != nil {
		return token.(*models.RoomToken)
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
		h.writeError(w, http.StatusInternalServerError, err.Error())
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

// Project handlers

func (h *Handler) GetProjects(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	projects, err := h.svc.GetProjects(token.MemberID, token.RoomID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"projects": projects,
		},
	})
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req models.CreateProjectRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	project, err := h.svc.CreateProject(token.MemberID, token.RoomID, req.Name)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    project,
	})
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	projectID := vars["id"]

	var req models.UpdateProjectRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := h.svc.UpdateProject(projectID, token.MemberID, req.Name); err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":   projectID,
			"name": req.Name,
		},
	})
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == nil {
		h.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	vars := mux.Vars(r)
	projectID := vars["id"]

	if err := h.svc.DeleteProject(projectID, token.MemberID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"message": "项目已删除",
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

	task, err := h.svc.CreateTask(token.MemberID, token.RoomID, req.ClientID, req.Title, req.ProjectID)
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

	if err := h.svc.UpdateTask(taskID, token.MemberID, req.Title, req.Status, req.ProjectID); err != nil {
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
		h.writeError(w, http.StatusInternalServerError, "internal_error")
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
			"message":          "公告已发送",
			"announcement_id":  announcement.ID,
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
