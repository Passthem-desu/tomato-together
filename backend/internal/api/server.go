package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"tomatogether/backend/internal/sse"
)

// Server represents the HTTP server
type Server struct {
	port string
	db   *sql.DB
	hub  *sse.Hub

	// Services
	userService       interface{}
	roomService       interface{}
	authService       interface{}
	pomodoroService   interface{}
	projectService    interface{}
	taskService       interface{}
	statusService     interface{}
	announcementService interface{}

	// Repositories
	userRepo        interface{}
	roomRepo        interface{}
	roomMemberRepo  interface{}
	roomTokenRepo   interface{}
	projectRepo     interface{}
	taskRepo        interface{}
	pomodoroRepo    interface{}
	statusRepo      interface{}
	announcementRepo interface{}
}

// NewServer creates a new server
func NewServer(port string, db *sql.DB, hub *sse.Hub) *Server {
	return &Server{
		port: port,
		db:   db,
		hub:  hub,
	}
}

// RegisterServices registers all services
func (s *Server) RegisterServices(user, room, auth, pomodoro, project, task, status, announcement interface{}) {
	s.userService = user
	s.roomService = room
	s.authService = auth
	s.pomodoroService = pomodoro
	s.projectService = project
	s.taskService = task
	s.statusService = status
	s.announcementService = announcement
}

// RegisterRepositories registers all repositories
func (s *Server) RegisterRepositories(user, room, member, token, project, task, pomodoro, status, announcement interface{}) {
	s.userRepo = user
	s.roomRepo = room
	s.roomMemberRepo = member
	s.roomTokenRepo = token
	s.projectRepo = project
	s.taskRepo = task
	s.pomodoroRepo = pomodoro
	s.statusRepo = status
	s.announcementRepo = announcement
}

// Run starts the server
func (s *Server) Run() error {
	// Load .env file
	godotenv.Load()

	router := mux.NewRouter()

	// CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origins := os.Getenv("CORS_ORIGINS")
			if origins == "" || origins == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				for _, origin := range strings.Split(origins, ",") {
					if r.Header.Get("Origin") == strings.TrimSpace(origin) {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Health check
	router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// API routes
	api := router.PathPrefix("/api").Subrouter()

	// Auth routes (L3)
	api.HandleFunc("/auth/register", s.handleRegister).Methods("POST")
	api.HandleFunc("/auth/login", s.handleLogin).Methods("POST")
	api.HandleFunc("/auth/me", s.handleMe).Methods("GET")
	api.HandleFunc("/auth/refresh", s.handleRefresh).Methods("POST")
	api.HandleFunc("/auth/upgrade", s.handleUpgrade).Methods("POST")

	// Room routes
	api.HandleFunc("/rooms", s.handleCreateRoom).Methods("POST")
	api.HandleFunc("/rooms/{name}/join", s.handleJoinRoom).Methods("POST")
	api.HandleFunc("/rooms/{name}/leave", s.handleLeaveRoom).Methods("POST")
	api.HandleFunc("/rooms/{name}", s.handleGetRoom).Methods("GET")
	api.HandleFunc("/rooms/{name}/users", s.handleGetRoomUsers).Methods("GET")
	api.HandleFunc("/rooms/{name}/settings", s.handleUpdateRoomSettings).Methods("PUT")
	api.HandleFunc("/rooms/{name}/owners/{user_id}", s.handleSetOwner).Methods("PUT")
	api.HandleFunc("/rooms/{name}/stats", s.handleGetRoomStats).Methods("GET")
	api.HandleFunc("/rooms/{name}/announcement", s.handleCreateAnnouncement).Methods("POST")
	api.HandleFunc("/rooms/{name}/announcements", s.handleGetAnnouncements).Methods("GET")
	api.HandleFunc("/rooms/{name}/sse", s.handleRoomSSE).Methods("GET")

	// Pomodoro routes (L2)
	api.HandleFunc("/pomodoro/start", s.handleStartPomodoro).Methods("POST")
	api.HandleFunc("/pomodoro/follow", s.handleFollowPomodoro).Methods("POST")
	api.HandleFunc("/pomodoro/unfollow", s.handleUnfollowPomodoro).Methods("POST")
	api.HandleFunc("/pomodoro/end", s.handleEndPomodoro).Methods("POST")
	api.HandleFunc("/pomodoro/status", s.handleGetPomodoroStatus).Methods("GET")

	// Status routes (L2)
	api.HandleFunc("/status", s.handleUpdateStatus).Methods("PUT")
	api.HandleFunc("/status", s.handleDeleteStatus).Methods("DELETE")

	// Stats routes
	api.HandleFunc("/stats", s.handleGetStats).Methods("GET")

	// Project routes (L3)
	api.HandleFunc("/projects", s.handleGetProjects).Methods("GET")
	api.HandleFunc("/projects", s.handleCreateProject).Methods("POST")
	api.HandleFunc("/projects/{id}", s.handleUpdateProject).Methods("PUT")
	api.HandleFunc("/projects/{id}", s.handleDeleteProject).Methods("DELETE")

	// Task routes (L3)
	api.HandleFunc("/tasks", s.handleGetTasks).Methods("GET")
	api.HandleFunc("/tasks", s.handleCreateTask).Methods("POST")
	api.HandleFunc("/tasks/sync", s.handleSyncTasks).Methods("POST")
	api.HandleFunc("/tasks/{id}", s.handleUpdateTask).Methods("PUT")
	api.HandleFunc("/tasks/{id}", s.handleDeleteTask).Methods("DELETE")

	// Notifications (L2)
	api.HandleFunc("/notifications", s.handleGetNotifications).Methods("GET")

	return http.ListenAndServe(":"+s.port, router)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Helper functions

func (s *Server) getUserFromToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", nil
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	// Token validation would go here
	return token, nil
}

func (s *Server) getRoomTokenFromRequest(r *http.Request) string {
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if auth != "" && !strings.HasPrefix(auth, "eyJ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	// Check query param
	return r.URL.Query().Get("token")
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return defaultValue
}

func formatSuccess(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"data":    data,
	}
}

func formatError(message string) map[string]string {
	return map[string]string{"error": message}
}