package service

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

func setupTestService(t *testing.T) (*Service, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	db.Exec("PRAGMA journal_mode=WAL")

	schema := `
	CREATE TABLE rooms (id TEXT PRIMARY KEY, name TEXT UNIQUE NOT NULL, password_hash TEXT DEFAULT '', is_readonly INTEGER DEFAULT 0, created_at DATETIME NOT NULL);
	CREATE TABLE room_members (id TEXT PRIMARY KEY, room_id TEXT NOT NULL, username TEXT NOT NULL, password_hash TEXT DEFAULT '', is_owner INTEGER DEFAULT 0, joined_at DATETIME NOT NULL, UNIQUE(room_id, username), FOREIGN KEY (room_id) REFERENCES rooms(id));
	CREATE TABLE room_tokens (id TEXT PRIMARY KEY, member_id TEXT NOT NULL, room_id TEXT NOT NULL, token TEXT NOT NULL, token_hash TEXT DEFAULT '', created_at DATETIME NOT NULL, expires_at DATETIME NOT NULL, last_heartbeat DATETIME NOT NULL, UNIQUE(member_id, room_id));
	CREATE TABLE pomodoro_sessions (id TEXT PRIMARY KEY, member_id TEXT NOT NULL, room_id TEXT NOT NULL, tag_id TEXT DEFAULT '', task_id TEXT DEFAULT '', duration INTEGER DEFAULT 0, planned_duration INTEGER DEFAULT 1500, is_followed INTEGER DEFAULT 0, leader_id TEXT DEFAULT '', started_at DATETIME NOT NULL, ended_at DATETIME, paused_at DATETIME, rest_duration INTEGER DEFAULT 0, is_long_break INTEGER DEFAULT 0, planned_rest_duration INTEGER DEFAULT 300, planned_long_break_duration INTEGER DEFAULT 900, sessions_before_long_break INTEGER DEFAULT 4, session_index INTEGER DEFAULT 0);
	CREATE TABLE token_revocations (id TEXT PRIMARY KEY, token_hash TEXT NOT NULL, reason TEXT DEFAULT '', revoked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
	CREATE TABLE announcements (id TEXT PRIMARY KEY, room_id TEXT NOT NULL, sender_id TEXT NOT NULL, title TEXT NOT NULL, body TEXT DEFAULT '', created_at DATETIME NOT NULL);
	CREATE TABLE user_statuses (id TEXT PRIMARY KEY, member_id TEXT NOT NULL, room_id TEXT NOT NULL, emoji TEXT DEFAULT '', message TEXT DEFAULT '', updated_at DATETIME NOT NULL, UNIQUE(member_id, room_id));
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	repo := repository.New(db)
	svc := New(repo)

	cleanup := func() {
		db.Close()
	}

	return svc, cleanup
}

func TestCreateRoom(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	resp, err := svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "test-room",
		Username: "owner",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}
	if resp.Token == "" {
		t.Error("Expected non-empty token")
	}
	if !resp.Member.IsOwner {
		t.Error("Expected creator to be owner")
	}
	if !resp.Member.IsPersistent {
		t.Error("Expected persistent user (password set)")
	}
}

func TestCreateRoomDuplicateName(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "dup-room",
		Username: "u1",
		Password: "pass1234",
	})

	_, err := svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "dup-room",
		Username: "u2",
		Password: "pass1234",
	})
	if err != ErrRoomNameTaken {
		t.Errorf("Expected ErrRoomNameTaken, got %v", err)
	}
}

func TestCreateRoomPasswordTooShort(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	_, err := svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "room",
		Username: "u",
		Password: "12345", // < 6
	})
	if err != ErrPasswordTooShort {
		t.Errorf("Expected ErrPasswordTooShort, got %v", err)
	}
}

func TestJoinRoom(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	// Create room
	svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "testroom",
		Username: "owner",
		Password: "owner123",
	})

	// Join as new user
	resp, err := svc.JoinRoom("testroom", &models.JoinRoomRequest{
		Username: "joiner",
		Password: "join4567",
	})
	if err != nil {
		t.Fatalf("JoinRoom failed: %v", err)
	}
	if resp.Token == "" {
		t.Error("Expected non-empty token after join")
	}
}

func TestJoinRoomInvalidPassword(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "testroom",
		Username: "owner",
		Password: "owner123",
	})

	// Try to join as existing persistent user with wrong password
	_, err := svc.JoinRoom("testroom", &models.JoinRoomRequest{
		Username: "owner",
		Password: "wrongpass",
	})
	if err != ErrInvalidPassword {
		t.Errorf("Expected ErrInvalidPassword, got %v", err)
	}
}

func TestValidateToken(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	resp, _ := svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "testroom",
		Username: "owner",
		Password: "owner123",
	})

	token, err := svc.ValidateToken(resp.Token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if token.MemberID == "" {
		t.Error("Expected member_id in token")
	}
}

func TestValidateTokenInvalid(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	_, err := svc.ValidateToken("nonexistent-token")
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got %v", err)
	}
}

func TestValidateTokenRevoked(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	resp, _ := svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "testroom",
		Username: "owner",
		Password: "owner123",
	})

	// Revoke the token
	svc.repo.RevokeToken(resp.Token)

	_, err := svc.ValidateToken(resp.Token)
	if err != ErrTokenRevoked {
		t.Errorf("Expected ErrTokenRevoked, got %v", err)
	}
}

func TestPomodoroStartAndEnd(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	resp, _ := svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "testroom",
		Username: "owner",
		Password: "owner123",
	})

	// Start pomodoro
	status, err := svc.StartPomodoro(resp.Token, &models.StartPomodoroRequest{
		PlannedDuration: 1500,
		RestDuration:    300,
	})
	if err != nil {
		t.Fatalf("StartPomodoro failed: %v", err)
	}
	if status.Phase != "focusing" {
		t.Errorf("Expected phase 'focusing', got %q", status.Phase)
	}

	// End pomodoro (normal end, not aborted)
	endResp, err := svc.EndPomodoro(resp.Token, &models.EndPomodoroRequest{
		Aborted: false,
	})
	if err != nil {
		t.Fatalf("EndPomodoro failed: %v", err)
	}
	if endResp.Phase != "rest" {
		t.Errorf("Expected phase 'rest' after normal end, got %q", endResp.Phase)
	}
}

func TestPomodoroShortNotCounted(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	resp, _ := svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "testroom",
		Username: "owner",
		Password: "owner123",
	})

	// Start pomodoro
	svc.StartPomodoro(resp.Token, &models.StartPomodoroRequest{
		PlannedDuration: 1500,
	})

	// Immediately abort
	endResp, err := svc.EndPomodoro(resp.Token, &models.EndPomodoroRequest{
		Aborted: true,
	})
	if err != nil {
		t.Fatalf("EndPomodoro failed: %v", err)
	}

	// Duration should be 0 (started < 60s ago)
	if endResp.Duration != 0 {
		t.Errorf("Expected Duration=0 for short pomodoro, got %d", endResp.Duration)
	}

	// Stats should not count this
	count, dur, _ := svc.GetMemberStats(resp.Member.ID)
	if count != 0 {
		t.Errorf("Expected 0 pomodoros counted, got %d", count)
	}
	if dur != 0 {
		t.Errorf("Expected 0 total duration, got %d", dur)
	}
}

func TestCreateAnnouncementRequiresOwner(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "testroom",
		Username: "owner",
		Password: "owner123",
	})

	// Join as non-owner
	joinResp, _ := svc.JoinRoom("testroom", &models.JoinRoomRequest{
		Username: "regular",
	})

	// Non-owner should not be able to create announcement
	_, err := svc.CreateAnnouncement("testroom", joinResp.Member.ID, "Title", "Body")
	if err != ErrMustBeOwner {
		t.Errorf("Expected ErrMustBeOwner, got %v", err)
	}
}

func TestFieldLengthValidation(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	longName := ""
	for i := 0; i < MaxRoomNameLen+1; i++ {
		longName += "a"
	}

	_, err := svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: longName,
		Username: "u",
		Password: "123456",
	})
	if err != ErrFieldTooLong {
		t.Errorf("Expected ErrFieldTooLong for long room name, got %v", err)
	}
}

func TestKickMember(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	ownerResp, _ := svc.CreateRoom(&models.CreateRoomRequest{
		RoomName: "testroom",
		Username: "owner",
		Password: "owner123",
	})

	joinResp, _ := svc.JoinRoom("testroom", &models.JoinRoomRequest{
		Username: "regular",
	})

	err := svc.KickMember(ownerResp.Token, joinResp.Member.ID)
	if err != nil {
		t.Fatalf("KickMember failed: %v", err)
	}

	// Regular user's token should be revoked
	revoked, _ := svc.repo.IsTokenRevoked(joinResp.Token)
	if !revoked {
		t.Error("Expected kicked user's token to be revoked")
	}
}
