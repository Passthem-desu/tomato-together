package models

import "time"

// Room represents a room
type Room struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	IsReadonly   bool      `json:"is_readonly"`
	CreatedAt    time.Time `json:"created_at"`
	HasPassword  bool      `json:"has_password"`
}

// RoomMember represents a member in a room (the core user entity)
type RoomMember struct {
	ID           string    `json:"id"`
	RoomID       string    `json:"room_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	IsOwner      bool      `json:"is_owner"`
	JoinedAt     time.Time `json:"joined_at"`
	IsPersistent bool      `json:"is_persistent"`
}

// RoomToken represents a room authentication token
type RoomToken struct {
	ID            string    `json:"id"`
	MemberID      string    `json:"member_id"`
	RoomID        string    `json:"room_id"`
	Token         string    `json:"token"`
	TokenHash     string    `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// Tag represents a tag (category label for tasks and pomodoros)
type Tag struct {
	ID        string    `json:"id"`
	MemberID  string    `json:"member_id"`
	RoomID    string    `json:"room_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Task represents a WIP task
type Task struct {
	ID          string     `json:"id"`
	ClientID    string     `json:"client_id"`
	MemberID    string     `json:"member_id"`
	RoomID      string     `json:"room_id"`
	TagID       string     `json:"tag_id"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// PomodoroSession represents a pomodoro session (also acts as state machine)
type PomodoroSession struct {
	ID                       string     `json:"id"`
	MemberID                 string     `json:"member_id"`
	RoomID                   string     `json:"room_id"`
	TagID                    string     `json:"tag_id"`
	TaskID                   string     `json:"task_id"`
	Duration                 int        `json:"duration"`
	PlannedDuration          int        `json:"planned_duration"`
	IsFollowed               bool       `json:"is_followed"`
	LeaderID                 string     `json:"leader_id"`
	StartedAt                time.Time  `json:"started_at"`
	EndedAt                  *time.Time `json:"ended_at"`
	PausedAt                 *time.Time `json:"paused_at,omitempty"`
	RestDuration             int        `json:"rest_duration"`
	IsLongBreak              bool       `json:"is_long_break"`
	PlannedRestDuration      int        `json:"-"`
	PlannedLongBreakDuration int        `json:"-"`
	SessionsBeforeLongBreak  int        `json:"-"`
	SessionIndex             int        `json:"-"` // 1-based per-batch index
	TotalSessions            int        `json:"-"`
}

// UserStatus represents a user's status
type UserStatus struct {
	ID        string    `json:"id"`
	MemberID  string    `json:"member_id"`
	RoomID    string    `json:"room_id"`
	Emoji     string    `json:"emoji"`
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Announcement represents an announcement
type Announcement struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	SenderID  string    `json:"sender_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
