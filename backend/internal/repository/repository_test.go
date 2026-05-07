package repository

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"tomatogether/backend/internal/models"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Enable WAL
	db.Exec("PRAGMA journal_mode=WAL")

	// Create schema
	schema := `
	CREATE TABLE IF NOT EXISTS rooms (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		password_hash TEXT DEFAULT '',
		is_readonly INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL
	);
	CREATE TABLE IF NOT EXISTS room_members (
		id TEXT PRIMARY KEY,
		room_id TEXT NOT NULL,
		username TEXT NOT NULL,
		password_hash TEXT DEFAULT '',
		is_owner INTEGER DEFAULT 0,
		joined_at DATETIME NOT NULL,
		UNIQUE(room_id, username),
		FOREIGN KEY (room_id) REFERENCES rooms(id)
	);
	CREATE TABLE IF NOT EXISTS room_tokens (
		id TEXT PRIMARY KEY,
		member_id TEXT NOT NULL,
		room_id TEXT NOT NULL,
		token TEXT NOT NULL,
		token_hash TEXT DEFAULT '',
		created_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL,
		last_heartbeat DATETIME NOT NULL,
		UNIQUE(member_id, room_id),
		FOREIGN KEY (member_id) REFERENCES room_members(id),
		FOREIGN KEY (room_id) REFERENCES rooms(id)
	);
	CREATE TABLE IF NOT EXISTS pomodoro_sessions (
		id TEXT PRIMARY KEY,
		member_id TEXT NOT NULL,
		room_id TEXT NOT NULL,
		tag_id TEXT DEFAULT '',
		task_id TEXT DEFAULT '',
		duration INTEGER DEFAULT 0,
		planned_duration INTEGER DEFAULT 1500,
		is_followed INTEGER DEFAULT 0,
		leader_id TEXT DEFAULT '',
		started_at DATETIME NOT NULL,
		ended_at DATETIME,
		paused_at DATETIME,
		rest_duration INTEGER DEFAULT 0,
		is_long_break INTEGER DEFAULT 0,
		planned_rest_duration INTEGER DEFAULT 300,
		planned_long_break_duration INTEGER DEFAULT 900,
		sessions_before_long_break INTEGER DEFAULT 4,
		session_index INTEGER DEFAULT 0,
		total_sessions INTEGER DEFAULT 0,
		FOREIGN KEY (member_id) REFERENCES room_members(id),
		FOREIGN KEY (room_id) REFERENCES rooms(id)
	);
	CREATE TABLE IF NOT EXISTS token_revocations (
		id TEXT PRIMARY KEY,
		token_hash TEXT NOT NULL,
		reason TEXT DEFAULT '',
		revoked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func createTestRoom(repo *Repository) (*models.Room, error) {
	room := &models.Room{
		ID:           "room-1",
		Name:         "test-room",
		PasswordHash: "",
		IsReadonly:   false,
		CreatedAt:    time.Now(),
	}
	err := repo.CreateRoom(room)
	return room, err
}

func createTestMember(repo *Repository, room *models.Room) (*models.RoomMember, error) {
	member := &models.RoomMember{
		ID:           "member-1",
		RoomID:       room.ID,
		Username:     "testuser",
		PasswordHash: "",
		IsOwner:      true,
		JoinedAt:     time.Now(),
	}
	err := repo.CreateMember(member)
	return member, err
}

func createTestToken(repo *Repository, member *models.RoomMember, room *models.Room) (*models.RoomToken, error) {
	tokenVal := "test-token-value-1234"
	tokenHash := hashToken(tokenVal)
	token := &models.RoomToken{
		ID:            "token-1",
		MemberID:      member.ID,
		RoomID:        room.ID,
		Token:         tokenVal,
		TokenHash:     tokenHash,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		LastHeartbeat: time.Now(),
	}
	err := repo.CreateToken(token)
	return token, err
}

func TestCreateAndGetRoom(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := New(db)

	room, err := createTestRoom(repo)
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}

	got, err := repo.GetRoomByName(room.Name)
	if err != nil {
		t.Fatalf("GetRoomByName failed: %v", err)
	}
	if got.Name != room.Name {
		t.Errorf("Expected room name %q, got %q", room.Name, got.Name)
	}
}

func TestRoomNameUnique(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := New(db)

	createTestRoom(repo)

	// Try to create duplicate
	dup := &models.Room{
		ID:        "room-2",
		Name:      "test-room",
		CreatedAt: time.Now(),
	}
	err := repo.CreateRoom(dup)
	if err == nil {
		t.Error("Expected error for duplicate room name, got nil")
	}
}

func TestCreateAndGetMember(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := New(db)

	room, _ := createTestRoom(repo)
	member, err := createTestMember(repo, room)
	if err != nil {
		t.Fatalf("CreateMember failed: %v", err)
	}

	got, err := repo.GetMemberByID(member.ID)
	if err != nil {
		t.Fatalf("GetMemberByID failed: %v", err)
	}
	if got.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %q", got.Username)
	}
	if !got.IsOwner {
		t.Error("Expected member to be owner")
	}
}

func TestTokenHashStorage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := New(db)

	room, _ := createTestRoom(repo)
	member, _ := createTestMember(repo, room)
	token, err := createTestToken(repo, member, room)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	// Token should have hash set
	if token.TokenHash == "" {
		t.Error("Expected TokenHash to be set")
	}

	// Lookup by token value should work
	got, err := repo.GetTokenByValue(token.Token)
	if err != nil {
		t.Fatalf("GetTokenByValue failed: %v", err)
	}
	if got.MemberID != member.ID {
		t.Errorf("Expected member_id %q, got %q", member.ID, got.MemberID)
	}
}

func TestTokenRevocation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := New(db)

	room, _ := createTestRoom(repo)
	member, _ := createTestMember(repo, room)
	token, _ := createTestToken(repo, member, room)

	// Revoke token
	if err := repo.RevokeToken(token.Token); err != nil {
		t.Fatalf("RevokeToken failed: %v", err)
	}

	// Check revocation
	revoked, err := repo.IsTokenRevoked(token.Token)
	if err != nil {
		t.Fatalf("IsTokenRevoked failed: %v", err)
	}
	if !revoked {
		t.Error("Expected token to be revoked")
	}
}

func TestPomodoroSessionShortNotCounted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := New(db)

	room, _ := createTestRoom(repo)
	member, _ := createTestMember(repo, room)

	// Create a session that started 30 seconds ago
	now := time.Now()
	startedAt := now.Add(-30 * time.Second)
	session := &models.PomodoroSession{
		ID:              "session-1",
		MemberID:        member.ID,
		RoomID:          room.ID,
		PlannedDuration: 1500,
		IsFollowed:      false,
		StartedAt:       startedAt,
	}
	if err := repo.CreatePomodoroSession(session); err != nil {
		t.Fatalf("CreatePomodoroSession failed: %v", err)
	}

	// End the session - duration should be set to 0 since < 60s
	if err := repo.EndActiveSessionByMemberID(member.ID); err != nil {
		t.Fatalf("EndActiveSessionByMemberID failed: %v", err)
	}

	// Verify duration is 0
	_, totalDuration, err := repo.GetMemberStats(member.ID)
	if err != nil {
		t.Fatalf("GetMemberStats failed: %v", err)
	}
	if totalDuration != 0 {
		t.Errorf("Expected 0 duration for short session, got %d", totalDuration)
	}
}

func TestPomodoroSessionNormalCounted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := New(db)

	room, _ := createTestRoom(repo)
	member, _ := createTestMember(repo, room)

	// Create a session that started 90 seconds ago
	now := time.Now()
	startedAt := now.Add(-90 * time.Second)
	session := &models.PomodoroSession{
		ID:              "session-2",
		MemberID:        member.ID,
		RoomID:          room.ID,
		PlannedDuration: 1500,
		IsFollowed:      false,
		StartedAt:       startedAt,
	}
	if err := repo.CreatePomodoroSession(session); err != nil {
		t.Fatalf("CreatePomodoroSession failed: %v", err)
	}

	// End via EndActiveSession - duration >= 60, should be counted
	if err := repo.EndActiveSessionByMemberID(member.ID); err != nil {
		t.Fatalf("EndActiveSessionByMemberID failed: %v", err)
	}

	totalPomodoros, totalDuration, err := repo.GetMemberStats(member.ID)
	if err != nil {
		t.Fatalf("GetMemberStats failed: %v", err)
	}
	if totalDuration <= 0 {
		t.Errorf("Expected >0 duration for normal session, got %d", totalDuration)
	}
	if totalPomodoros != 1 {
		t.Errorf("Expected 1 pomodoro, got %d", totalPomodoros)
	}
}

func TestPauseSessionOnEndedSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := New(db)

	room, _ := createTestRoom(repo)
	member, _ := createTestMember(repo, room)

	now := time.Now()
	session := &models.PomodoroSession{
		ID:              "session-1",
		MemberID:        member.ID,
		RoomID:          room.ID,
		PlannedDuration: 1500,
		IsFollowed:      false,
		StartedAt:       now.Add(-120 * time.Second),
	}
	if err := repo.CreatePomodoroSession(session); err != nil {
		t.Fatalf("CreatePomodoroSession failed: %v", err)
	}

	if err := repo.EndActiveSessionByMemberID(member.ID); err != nil {
		t.Fatalf("EndActiveSessionByMemberID failed: %v", err)
	}

	err := repo.PauseSession(session.ID)
	if err != ErrSessionNotActive {
		t.Errorf("Expected ErrSessionNotActive when pausing ended session, got %v", err)
	}
}

func TestResumeSessionOnUnpausedSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := New(db)

	room, _ := createTestRoom(repo)
	member, _ := createTestMember(repo, room)

	now := time.Now()
	session := &models.PomodoroSession{
		ID:              "session-1",
		MemberID:        member.ID,
		RoomID:          room.ID,
		PlannedDuration: 1500,
		IsFollowed:      false,
		StartedAt:       now,
	}
	if err := repo.CreatePomodoroSession(session); err != nil {
		t.Fatalf("CreatePomodoroSession failed: %v", err)
	}

	err := repo.ResumeSession(session.ID, 0)
	if err != ErrSessionNotActive {
		t.Errorf("Expected ErrSessionNotActive when resuming unpaused session, got %v", err)
	}
}
