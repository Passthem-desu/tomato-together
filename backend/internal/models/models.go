package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// IsPersistent checks if the user is a persistent user (has password)
func (u *User) IsPersistent() bool {
	return u.PasswordHash != ""
}

// Room represents a chat room
type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Password  string    `json:"-"`
	IsReadonly bool     `json:"is_readonly"`
	CreatedAt time.Time `json:"created_at"`
	HasPassword bool    `json:"has_password"`
}

// RoomMember represents a user's membership in a room
type RoomMember struct {
	ID       string    `json:"id"`
	RoomID   string    `json:"room_id"`
	UserID   string    `json:"user_id"`
	IsOwner  bool      `json:"is_owner"`
	JoinedAt time.Time `json:"joined_at"`
}

// RoomToken represents a room-level authentication token
type RoomToken struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	RoomID        string    `json:"room_id"`
	Token         string    `json:"token"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// IsExpired checks if the token has expired
func (rt *RoomToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// Project represents a user's project
type Project struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Task represents a work-in-progress item
type Task struct {
	ID          string     `json:"id"`
	ClientID    string     `json:"client_id,omitempty"`
	UserID      string     `json:"user_id"`
	ProjectID   *string    `json:"project_id,omitempty"`
	Title       string     `json:"title"`
	Status      string     `json:"status"` // TODO, WIP, DONE
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// TaskStatus constants
const (
	TaskStatusTODO = "TODO"
	TaskStatusWIP  = "WIP"
	TaskStatusDONE = "DONE"
)

// PomodoroSession represents a pomodoro timer session
type PomodoroSession struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	RoomID           string     `json:"room_id"`
	ProjectID        *string    `json:"project_id,omitempty"`
	TaskID           *string    `json:"task_id,omitempty"`
	Duration         int        `json:"duration"`
	PlannedDuration  int        `json:"planned_duration"`
	IsFollowed       bool       `json:"is_followed"`
	LeaderID         *string    `json:"leader_id,omitempty"`
	StartedAt        time.Time  `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at,omitempty"`
}

// UserStatus represents a user's status in a room
type UserStatus struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	RoomID   string    `json:"room_id"`
	Emoji    string    `json:"emoji"`
	Message  string    `json:"message"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Announcement represents a room announcement
type Announcement struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	SenderID  string    `json:"sender_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// Pagination represents pagination info
type Pagination struct {
	Page    int `json:"page"`
	Limit   int `json:"limit"`
	Total   int `json:"total"`
}

// ---- API Request/Response types ----

// UserInfo represents public user info
type UserInfo struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token string    `json:"token"`
	User  *UserInfo `json:"user"`
}

// RoomInfo represents room info in API responses
type RoomInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IsReadonly  bool      `json:"is_readonly"`
	HasPassword bool      `json:"has_password"`
	MemberCount int       `json:"member_count,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	Owner       *UserInfo `json:"owner,omitempty"`
}

// RoomMemberInfo represents room member info in API responses
type RoomMemberInfo struct {
	ID       string    `json:"id"`
	Username string    `json:"username"`
	IsOwner  bool      `json:"is_owner"`
	Status   *StatusInfo `json:"status,omitempty"`
	Pomodoro *PomodoroInfo `json:"pomodoro,omitempty"`
	IsOnline bool      `json:"is_online"`
}

// StatusInfo represents user status info
type StatusInfo struct {
	Emoji   string `json:"emoji"`
	Message string `json:"message"`
}

// PomodoroInfo represents pomodoro status info
type PomodoroInfo struct {
	IsActive       bool   `json:"is_active"`
	IsFollowing    bool   `json:"is_following"`
	LeaderUsername string `json:"leader_username,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	RemainingSeconds int `json:"remaining_seconds,omitempty"`
}

// PomodoroSessionResponse represents pomodoro session in API responses
type PomodoroSessionResponse struct {
	SessionID              string `json:"session_id"`
	StartedAt              string `json:"started_at"`
	PlannedDuration        int    `json:"planned_duration"`
	RestDuration           int    `json:"rest_duration,omitempty"`
	LongBreakDuration      int    `json:"long_break_duration,omitempty"`
	SessionsBeforeLongBreak int   `json:"sessions_before_long_break,omitempty"`
	Status                 string `json:"status"` // focusing, following, rest
	SessionsToday          int    `json:"sessions_today,omitempty"`
	Duration               int    `json:"duration,omitempty"`
	IsFollowed             bool   `json:"is_followed,omitempty"`
	ShouldTakeLongBreak    bool   `json:"should_take_long_break,omitempty"`
	SessionsCompleted      int    `json:"sessions_completed,omitempty"`
	LeaderID               string `json:"leader_id,omitempty"`
	LeaderUsername         string `json:"leader_username,omitempty"`
	RemainingSeconds       int    `json:"remaining_seconds,omitempty"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ErrorResponse creates an error response
func ErrorResponse(message string) *APIResponse {
	return &APIResponse{
		Success: false,
		Error:   message,
	}
}

// SuccessResponse creates a success response
func SuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Success: true,
		Data:    data,
	}
}