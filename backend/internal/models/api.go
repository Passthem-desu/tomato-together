package models

// APIResponse is the standard API response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// RoomResponse is the response for room operations
type RoomResponse struct {
	Room         *RoomInfo   `json:"room"`
	Member       *MemberInfo `json:"member"`
	Token        string      `json:"token"`
	AccessToken  string      `json:"access_token,omitempty"`
	RefreshToken string      `json:"refresh_token,omitempty"`
	ExpiresIn    int64       `json:"expires_in,omitempty"`
}

// RoomInfo contains room information for responses
type RoomInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsReadonly  bool   `json:"is_readonly"`
	HasPassword bool   `json:"has_password"`
}

// UserCheckResponse is the response for checking if a username exists
type UserCheckResponse struct {
	Exists       bool `json:"exists"`
	IsPersistent bool `json:"is_persistent"`
}

// MemberInfo contains member information for responses
type MemberInfo struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	IsOwner      bool   `json:"is_owner"`
	IsPersistent bool   `json:"is_persistent"`
	JoinedAt     string `json:"joined_at,omitempty"`
}

// UserInfo contains detailed user information for responses
type UserInfo struct {
	ID             string        `json:"id"`
	Username       string        `json:"username"`
	IsOwner        bool          `json:"is_owner"`
	IsPersistent   bool          `json:"is_persistent"`
	Status         *StatusInfo   `json:"status,omitempty"`
	Pomodoro       *PomodoroInfo `json:"pomodoro,omitempty"`
	IsOnline       bool          `json:"is_online"`
	TotalPomodoros int           `json:"total_pomodoros"`
	TotalDuration  int           `json:"total_duration"`
}

// StatusInfo contains user status information
type StatusInfo struct {
	Emoji   string `json:"emoji"`
	Message string `json:"message"`
}

// PomodoroInfo contains user pomodoro information (for user list)
type PomodoroInfo struct {
	Phase            string `json:"phase"`
	IsFollowing      bool   `json:"is_following"`
	LeaderUsername   string `json:"leader_username,omitempty"`
	StartedAt        string `json:"started_at,omitempty"`
	RemainingSeconds int    `json:"remaining_seconds,omitempty"`
}

// PomodoroStatusResponse is the response for pomodoro status
// Phase replaces the old is_active + status fields
type PomodoroStatusResponse struct {
	Phase                   string `json:"phase"`
	SessionID               string `json:"session_id,omitempty"`
	StartedAt               string `json:"started_at,omitempty"`
	PausedAt                string `json:"paused_at,omitempty"`
	RemainingSeconds        int    `json:"remaining_seconds,omitempty"`
	LeaderID                string `json:"leader_id,omitempty"`
	LeaderUsername          string `json:"leader_username,omitempty"`
	PlannedDuration         int    `json:"planned_duration,omitempty"`
	RestDuration            int    `json:"rest_duration,omitempty"`
	LongBreakDuration       int    `json:"long_break_duration,omitempty"`
	SessionsBeforeLongBreak int    `json:"sessions_before_long_break,omitempty"`
	Duration                int    `json:"duration,omitempty"`
	IsFollowed              bool   `json:"is_followed,omitempty"`
	IsLongBreak             bool   `json:"is_long_break,omitempty"`
	ShouldTakeLongBreak     bool   `json:"should_take_long_break,omitempty"`
	SessionsCompleted       int    `json:"sessions_completed,omitempty"`
	TotalSessions           int    `json:"total_sessions,omitempty"`
}

// CreateRoomRequest is the request for creating a room
type CreateRoomRequest struct {
	RoomName     string `json:"room_name"`
	RoomPassword string `json:"room_password,omitempty"`
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`
	IsReadonly   bool   `json:"is_readonly"`
}

// JoinRoomRequest is the request for joining a room
type JoinRoomRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`
	RoomPassword string `json:"room_password,omitempty"`
}

// LoginRequest is the request for persistent user login
type LoginRequest struct {
	RoomName     string `json:"room_name"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	RoomPassword string `json:"room_password,omitempty"`
}

// UpgradeRequest is the request for upgrading to persistent user
type UpgradeRequest struct {
	Password string `json:"password"`
}

// ChangePasswordRequest is the request for changing own password
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// StartPomodoroRequest is the request for starting a pomodoro
type StartPomodoroRequest struct {
	RoomName                string `json:"room_name"`
	TagID                   string `json:"tag_id,omitempty"`
	TaskID                  string `json:"task_id,omitempty"`
	PlannedDuration         int    `json:"planned_duration"`
	RestDuration            int    `json:"rest_duration"`
	LongBreakDuration       int    `json:"long_break_duration"`
	SessionsBeforeLongBreak int    `json:"sessions_before_long_break"`
	SessionIndex            int    `json:"session_index,omitempty"` // 1-based batch position
	TotalSessions           int    `json:"total_sessions,omitempty"`
}

// FollowPomodoroRequest is the request for following a pomodoro
type FollowPomodoroRequest struct {
	RoomName string `json:"room_name"`
	LeaderID string `json:"leader_id"`
}

// FollowRoomRequest is the request for unfollow/cancel operations
type FollowRoomRequest struct {
	RoomName string `json:"room_name"`
}

// EndPomodoroRequest is the request for ending a pomodoro
type EndPomodoroRequest struct {
	RoomName     string `json:"room_name"`
	Aborted      bool   `json:"aborted"`
	SessionIndex int    `json:"session_index,omitempty"` // 1-based batch position, used for long-break calc
}

// UpdateStatusRequest is the request for updating user status
type UpdateStatusRequest struct {
	RoomName string `json:"room_name"`
	Emoji    string `json:"emoji"`
	Message  string `json:"message"`
}

// UpdateRoomSettingsRequest is the request for updating room settings
type UpdateRoomSettingsRequest struct {
	RoomPassword string `json:"room_password,omitempty"`
	IsReadonly   *bool  `json:"is_readonly,omitempty"`
}

// SetOwnerRequest is the request for setting room owner
type SetOwnerRequest struct {
	IsOwner bool `json:"is_owner"`
}

// CreateTagRequest is the request for creating a tag
type CreateTagRequest struct {
	RoomName string `json:"room_name"`
	Name     string `json:"name"`
}

// UpdateTagRequest is the request for updating a tag
type UpdateTagRequest struct {
	Name string `json:"name"`
}

// CreateTaskRequest is the request for creating a task
type CreateTaskRequest struct {
	RoomName string `json:"room_name"`
	ClientID string `json:"client_id"`
	Title    string `json:"title"`
	TagID    string `json:"tag_id,omitempty"`
}

// UpdateTaskRequest is the request for updating a task
type UpdateTaskRequest struct {
	Title  string `json:"title,omitempty"`
	Status string `json:"status,omitempty"`
	TagID  string `json:"tag_id,omitempty"`
}

// CreateAnnouncementRequest is the request for creating an announcement
type CreateAnnouncementRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// SyncTasksRequest is the request for syncing tasks
type SyncTasksRequest struct {
	RoomName string         `json:"room_name"`
	Tasks    []SyncTaskItem `json:"tasks"`
}

// SyncTaskItem represents a task item for syncing
type SyncTaskItem struct {
	ClientID  string `json:"client_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	TagID     string `json:"tag_id,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at,omitempty"`
	SortOrder int    `json:"sort_order,omitempty"`
}

// SyncTasksResponse is the response for syncing tasks
type SyncTasksResponse struct {
	Synced    int              `json:"synced"`
	Tasks     []SyncTaskResult `json:"tasks"`
	Conflicts []ConflictItem   `json:"conflicts,omitempty"`
}

// SyncTaskResult represents the result of syncing a task
type SyncTaskResult struct {
	ClientID string `json:"client_id"`
	ServerID string `json:"server_id"`
}

// ConflictItem represents a sync conflict where server data is newer
type ConflictItem struct {
	ClientID        string `json:"client_id"`
	ServerTitle     string `json:"server_title"`
	ServerStatus    string `json:"server_status"`
	ServerUpdatedAt string `json:"server_updated_at"`
	ClientUpdatedAt string `json:"client_updated_at"`
}

// StatsResponse is the response for stats
type StatsResponse struct {
	Period         string    `json:"period"`
	TotalPomodoros int       `json:"total_pomodoros"`
	TotalDuration  int       `json:"total_duration"`
	ByTag          []TagStat `json:"by_tag,omitempty"`
	ByDay          []DayStat `json:"by_day,omitempty"`
}

// TagStat represents stats by tag
type TagStat struct {
	TagID         string `json:"tag_id"`
	TagName       string `json:"tag_name"`
	PomodoroCount int    `json:"pomodoro_count"`
	TotalDuration int    `json:"total_duration"`
}

// DayStat represents stats by day
type DayStat struct {
	Date          string `json:"date"`
	PomodoroCount int    `json:"pomodoro_count"`
	TotalDuration int    `json:"total_duration"`
}
