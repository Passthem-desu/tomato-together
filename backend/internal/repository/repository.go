package repository

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"time"

	"tomatogether/backend/internal/models"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Room operations

func (r *Repository) CreateRoom(room *models.Room) error {
	query := `INSERT INTO rooms (id, name, password_hash, is_readonly, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, room.ID, room.Name, room.PasswordHash, boolToInt(room.IsReadonly), room.CreatedAt)
	return err
}

func (r *Repository) GetRoomByName(name string) (*models.Room, error) {
	query := `SELECT id, name, password_hash, is_readonly, created_at FROM rooms WHERE name = ?`
	row := r.db.QueryRow(query, name)
	room := &models.Room{}
	var isReadonly int
	err := row.Scan(&room.ID, &room.Name, &room.PasswordHash, &isReadonly, &room.CreatedAt)
	if err != nil {
		return nil, err
	}
	room.IsReadonly = isReadonly == 1
	room.HasPassword = room.PasswordHash != ""
	return room, nil
}

func (r *Repository) GetRoomByID(id string) (*models.Room, error) {
	query := `SELECT id, name, password_hash, is_readonly, created_at FROM rooms WHERE id = ?`
	row := r.db.QueryRow(query, id)
	room := &models.Room{}
	var isReadonly int
	err := row.Scan(&room.ID, &room.Name, &room.PasswordHash, &isReadonly, &room.CreatedAt)
	if err != nil {
		return nil, err
	}
	room.IsReadonly = isReadonly == 1
	room.HasPassword = room.PasswordHash != ""
	return room, nil
}

func (r *Repository) UpdateRoomSettings(roomID string, passwordHash string, isReadonly *bool) error {
	if passwordHash != "" && isReadonly != nil {
		_, err := r.db.Exec(`UPDATE rooms SET password_hash = ?, is_readonly = ? WHERE id = ?`,
			passwordHash, boolToInt(*isReadonly), roomID)
		return err
	}
	if passwordHash != "" {
		_, err := r.db.Exec(`UPDATE rooms SET password_hash = ? WHERE id = ?`, passwordHash, roomID)
		return err
	}
	if isReadonly != nil {
		_, err := r.db.Exec(`UPDATE rooms SET is_readonly = ? WHERE id = ?`, boolToInt(*isReadonly), roomID)
		return err
	}
	return nil
}

// RoomMember operations

func (r *Repository) CreateMember(member *models.RoomMember) error {
	query := `INSERT INTO room_members (id, room_id, username, password_hash, is_owner, joined_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, member.ID, member.RoomID, member.Username, member.PasswordHash, boolToInt(member.IsOwner), member.JoinedAt)
	return err
}

func (r *Repository) GetMemberByID(id string) (*models.RoomMember, error) {
	query := `SELECT id, room_id, username, password_hash, is_owner, joined_at FROM room_members WHERE id = ?`
	row := r.db.QueryRow(query, id)
	return r.scanMember(row)
}

func (r *Repository) GetMemberByUsername(roomID, username string) (*models.RoomMember, error) {
	query := `SELECT id, room_id, username, password_hash, is_owner, joined_at FROM room_members WHERE room_id = ? AND username = ?`
	row := r.db.QueryRow(query, roomID, username)
	return r.scanMember(row)
}

func (r *Repository) GetMembersByRoomID(roomID string) ([]*models.RoomMember, error) {
	query := `SELECT id, room_id, username, password_hash, is_owner, joined_at FROM room_members WHERE room_id = ?`
	rows, err := r.db.Query(query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*models.RoomMember
	for rows.Next() {
		member := &models.RoomMember{}
		var isOwner int
		err := rows.Scan(&member.ID, &member.RoomID, &member.Username, &member.PasswordHash, &isOwner, &member.JoinedAt)
		if err != nil {
			return nil, err
		}
		member.IsOwner = isOwner == 1
		member.IsPersistent = member.PasswordHash != ""
		members = append(members, member)
	}
	return members, nil
}

func (r *Repository) UpdateMemberPassword(memberID, passwordHash string) error {
	query := `UPDATE room_members SET password_hash = ? WHERE id = ?`
	_, err := r.db.Exec(query, passwordHash, memberID)
	return err
}

func (r *Repository) UpdateMemberPasswordInTx(tx *sql.Tx, memberID, passwordHash string) error {
	_, err := tx.Exec(`UPDATE room_members SET password_hash = ? WHERE id = ?`, passwordHash, memberID)
	return err
}

func (r *Repository) SetMemberOwner(memberID string, isOwner bool) error {
	query := `UPDATE room_members SET is_owner = ? WHERE id = ?`
	_, err := r.db.Exec(query, boolToInt(isOwner), memberID)
	return err
}

func (r *Repository) DeleteMember(memberID string) error {
	query := `DELETE FROM room_members WHERE id = ?`
	_, err := r.db.Exec(query, memberID)
	return err
}

func (r *Repository) HasOwnerInRoom(roomID string) (bool, error) {
	query := `SELECT COUNT(*) FROM room_members WHERE room_id = ? AND is_owner = 1`
	var count int
	err := r.db.QueryRow(query, roomID).Scan(&count)
	return count > 0, err
}

func (r *Repository) scanMember(row *sql.Row) (*models.RoomMember, error) {
	member := &models.RoomMember{}
	var isOwner int
	err := row.Scan(&member.ID, &member.RoomID, &member.Username, &member.PasswordHash, &isOwner, &member.JoinedAt)
	if err != nil {
		return nil, err
	}
	member.IsOwner = isOwner == 1
	member.IsPersistent = member.PasswordHash != ""
	return member, nil
}

// RoomToken operations

func (r *Repository) CreateToken(token *models.RoomToken) error {
	tokenHash := hashToken(token.Token)
	token.TokenHash = tokenHash
	query := `INSERT INTO room_tokens (id, member_id, room_id, token, token_hash, created_at, expires_at, last_heartbeat) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, token.ID, token.MemberID, token.RoomID, token.Token, token.TokenHash, token.CreatedAt, token.ExpiresAt, token.LastHeartbeat)
	return err
}

func (r *Repository) GetTokenByValue(tokenValue string) (*models.RoomToken, error) {
	tokenHash := hashToken(tokenValue)
	query := `SELECT id, member_id, room_id, token, token_hash, created_at, expires_at, last_heartbeat FROM room_tokens WHERE token_hash = ?`
	row := r.db.QueryRow(query, tokenHash)
	token, err := r.scanToken(row)
	if err != nil {
		// Fallback: try lookup by plain token for legacy (pre-migration) tokens
		query = `SELECT id, member_id, room_id, token, token_hash, created_at, expires_at, last_heartbeat FROM room_tokens WHERE token = ?`
		row = r.db.QueryRow(query, tokenValue)
		token, err = r.scanToken(row)
		if err != nil {
			return nil, err
		}
		// Upgrade legacy token to hash-based
		newHash := hashToken(tokenValue)
		r.db.Exec(`UPDATE room_tokens SET token_hash = ? WHERE id = ?`, newHash, token.ID)
		token.TokenHash = newHash
	}
	return token, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (r *Repository) UpdateTokenHeartbeat(tokenID string) error {
	query := `UPDATE room_tokens SET last_heartbeat = ? WHERE id = ?`
	_, err := r.db.Exec(query, time.Now(), tokenID)
	return err
}

func (r *Repository) DeleteToken(tokenID string) error {
	query := `DELETE FROM room_tokens WHERE id = ?`
	_, err := r.db.Exec(query, tokenID)
	return err
}

func (r *Repository) DeleteTokenByMemberID(memberID string) error {
	query := `DELETE FROM room_tokens WHERE member_id = ?`
	_, err := r.db.Exec(query, memberID)
	return err
}

func (r *Repository) DeleteTokenByMemberAndRoom(memberID, roomID string) error {
	query := `DELETE FROM room_tokens WHERE member_id = ? AND room_id = ?`
	_, err := r.db.Exec(query, memberID, roomID)
	return err
}

func (r *Repository) GetTokenByMemberAndRoom(memberID, roomID string) (*models.RoomToken, error) {
	query := `SELECT id, member_id, room_id, token, token_hash, created_at, expires_at, last_heartbeat FROM room_tokens WHERE member_id = ? AND room_id = ?`
	row := r.db.QueryRow(query, memberID, roomID)
	return r.scanToken(row)
}

func (r *Repository) scanToken(row *sql.Row) (*models.RoomToken, error) {
	token := &models.RoomToken{}
	err := row.Scan(&token.ID, &token.MemberID, &token.RoomID, &token.Token, &token.TokenHash, &token.CreatedAt, &token.ExpiresAt, &token.LastHeartbeat)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// RevokeToken revokes a token and adds it to the blacklist
func (r *Repository) RevokeToken(tokenValue string) error {
	tokenHash := hashToken(tokenValue)
	// Add to revocation list
	_, err := r.db.Exec(`INSERT OR IGNORE INTO token_revocations (id, token_hash, revoked_at) VALUES (?, ?, ?)`,
		tokenHash+"_revoked", tokenHash, time.Now())
	return err
}

// IsTokenRevoked checks if a token hash is in the blacklist
func (r *Repository) IsTokenRevoked(tokenValue string) (bool, error) {
	tokenHash := hashToken(tokenValue)
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM token_revocations WHERE token_hash = ?`, tokenHash).Scan(&count)
	return count > 0, err
}

// DeleteTokensByMemberID deletes all tokens for a member (kicks them from all sessions)
func (r *Repository) DeleteTokensByMemberID(memberID string) error {
	_, err := r.db.Exec(`DELETE FROM room_tokens WHERE member_id = ?`, memberID)
	return err
}

// RunInTx executes a function within a database transaction
func (r *Repository) RunInTx(fn func(tx *sql.Tx) error) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}

// CreateTokenInTx creates a token within an existing transaction
func (r *Repository) CreateTokenInTx(tx *sql.Tx, token *models.RoomToken) error {
	tokenHash := hashToken(token.Token)
	token.TokenHash = tokenHash
	query := `INSERT INTO room_tokens (id, member_id, room_id, token, token_hash, created_at, expires_at, last_heartbeat) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := tx.Exec(query, token.ID, token.MemberID, token.RoomID, token.Token, token.TokenHash, token.CreatedAt, token.ExpiresAt, token.LastHeartbeat)
	return err
}

// DeleteTokenByMemberAndRoomInTx deletes a token within an existing transaction
func (r *Repository) DeleteTokenByMemberAndRoomInTx(tx *sql.Tx, memberID, roomID string) error {
	_, err := tx.Exec(`DELETE FROM room_tokens WHERE member_id = ? AND room_id = ?`, memberID, roomID)
	return err
}

// CleanupExpiredRevocations removes revocation records older than 24 hours
func (r *Repository) CleanupExpiredRevocations() {
	_, err := r.db.Exec(`DELETE FROM token_revocations WHERE revoked_at < datetime('now', '-24 hours')`)
	if err != nil {
		log.Printf("Warning: failed to cleanup expired revocations: %v", err)
	}
}

// Tag operations

func (r *Repository) CreateTag(tag *models.Tag) error {
	query := `INSERT INTO tags (id, member_id, room_id, name, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, tag.ID, tag.MemberID, tag.RoomID, tag.Name, tag.CreatedAt)
	return err
}

func (r *Repository) GetTagByID(id string) (*models.Tag, error) {
	query := `SELECT id, member_id, room_id, name, created_at FROM tags WHERE id = ?`
	row := r.db.QueryRow(query, id)
	tag := &models.Tag{}
	err := row.Scan(&tag.ID, &tag.MemberID, &tag.RoomID, &tag.Name, &tag.CreatedAt)
	if err != nil {
		return nil, err
	}
	return tag, nil
}

func (r *Repository) GetTagsByMemberID(memberID string) ([]*models.Tag, error) {
	query := `SELECT id, member_id, room_id, name, created_at FROM tags WHERE member_id = ?`
	return r.scanTags(query, memberID)
}

func (r *Repository) GetTagsByMemberAndRoom(memberID, roomID string) ([]*models.Tag, error) {
	query := `SELECT id, member_id, room_id, name, created_at FROM tags WHERE member_id = ? AND room_id = ?`
	return r.scanTags(query, memberID, roomID)
}

func (r *Repository) UpdateTag(tagID, name string) error {
	query := `UPDATE tags SET name = ? WHERE id = ?`
	_, err := r.db.Exec(query, name, tagID)
	return err
}

func (r *Repository) DeleteTag(tagID string) error {
	query := `DELETE FROM tags WHERE id = ?`
	_, err := r.db.Exec(query, tagID)
	return err
}

func (r *Repository) scanTags(query string, args ...interface{}) ([]*models.Tag, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		tag := &models.Tag{}
		err := rows.Scan(&tag.ID, &tag.MemberID, &tag.RoomID, &tag.Name, &tag.CreatedAt)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// Task operations

func (r *Repository) CreateTask(task *models.Task) error {
	query := `INSERT INTO tasks (id, client_id, member_id, room_id, tag_id, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, task.ID, task.ClientID, task.MemberID, task.RoomID, task.TagID, task.Title, task.Status, task.CreatedAt, task.UpdatedAt)
	return err
}

func (r *Repository) GetTaskByID(id string) (*models.Task, error) {
	query := `SELECT id, client_id, member_id, room_id, tag_id, title, status, created_at, updated_at, completed_at FROM tasks WHERE id = ?`
	return r.scanTask(query, id)
}

func (r *Repository) GetTasksByMemberAndRoom(memberID, roomID string, status string) ([]*models.Task, error) {
	query := `SELECT id, client_id, member_id, room_id, tag_id, title, status, created_at, updated_at, completed_at FROM tasks WHERE member_id = ? AND room_id = ?`
	args := []interface{}{memberID, roomID}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	return r.scanTasks(query, args...)
}

func (r *Repository) UpdateTask(taskID string, title, status, tagID string, completedAt *time.Time) error {
	query := `UPDATE tasks SET title = COALESCE(NULLIF(?, ''), title), status = COALESCE(NULLIF(?, ''), status), tag_id = ?, updated_at = ?`
	args := []interface{}{title, status, tagID, time.Now()}
	if completedAt != nil {
		query += `, completed_at = ?`
		args = append(args, *completedAt)
	}
	query += ` WHERE id = ?`
	args = append(args, taskID)
	_, err := r.db.Exec(query, args...)
	return err
}

func (r *Repository) DeleteTask(taskID string) error {
	query := `DELETE FROM tasks WHERE id = ?`
	_, err := r.db.Exec(query, taskID)
	return err
}

func (r *Repository) scanTask(query string, args ...interface{}) (*models.Task, error) {
	row := r.db.QueryRow(query, args...)
	task := &models.Task{}
	var tagID sql.NullString
	var completedAt sql.NullTime
	err := row.Scan(&task.ID, &task.ClientID, &task.MemberID, &task.RoomID, &tagID, &task.Title, &task.Status, &task.CreatedAt, &task.UpdatedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	if tagID.Valid {
		task.TagID = tagID.String
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	return task, nil
}

func (r *Repository) scanTasks(query string, args ...interface{}) ([]*models.Task, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		var tagID sql.NullString
		var completedAt sql.NullTime
		err := rows.Scan(&task.ID, &task.ClientID, &task.MemberID, &task.RoomID, &tagID, &task.Title, &task.Status, &task.CreatedAt, &task.UpdatedAt, &completedAt)
		if err != nil {
			return nil, err
		}
		if tagID.Valid {
			task.TagID = tagID.String
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// PomodoroSession operations

func (r *Repository) CreatePomodoroSession(session *models.PomodoroSession) error {
	query := `INSERT INTO pomodoro_sessions (id, member_id, room_id, tag_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at, paused_at, rest_duration, is_long_break, planned_rest_duration, planned_long_break_duration, sessions_before_long_break, session_index) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, session.ID, session.MemberID, session.RoomID, session.TagID, session.TaskID, session.Duration, session.PlannedDuration, boolToInt(session.IsFollowed), session.LeaderID, session.StartedAt, session.EndedAt, session.PausedAt, session.RestDuration, boolToInt(session.IsLongBreak), session.PlannedRestDuration, session.PlannedLongBreakDuration, session.SessionsBeforeLongBreak, session.SessionIndex)
	return err
}

func (r *Repository) GetActiveSessionByMemberID(memberID string) (*models.PomodoroSession, error) {
	query := `SELECT id, member_id, room_id, tag_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at, paused_at, rest_duration, is_long_break, planned_rest_duration, planned_long_break_duration, sessions_before_long_break, session_index FROM pomodoro_sessions WHERE member_id = ? AND ended_at IS NULL`
	row := r.db.QueryRow(query, memberID)
	return r.scanPomodoroSession(row)
}

// EndActiveSessionByMemberID ends any active pomodoro session for a member
func (r *Repository) EndActiveSessionByMemberID(memberID string) error {
	now := time.Now()
	session, err := r.GetActiveSessionByMemberID(memberID)
	if err != nil {
		return err // No active session, nothing to end
	}
	duration := int(now.Sub(session.StartedAt).Seconds())
	if duration < 60 {
		duration = 0 // Don't count sessions stopped within 1 minute
	}
	return r.UpdatePomodoroSession(session.ID, &now, duration)
}

func (r *Repository) UpdatePomodoroSession(sessionID string, endedAt *time.Time, duration int) error {
	query := `UPDATE pomodoro_sessions SET ended_at = ?, duration = ? WHERE id = ?`
	_, err := r.db.Exec(query, endedAt, duration, sessionID)
	return err
}

// EndSessionWithRest ends a session and sets rest info
func (r *Repository) EndSessionWithRest(sessionID string, endedAt *time.Time, duration int, restDuration int, isLongBreak bool) error {
	query := `UPDATE pomodoro_sessions SET ended_at = ?, duration = ?, rest_duration = ?, is_long_break = ? WHERE id = ?`
	_, err := r.db.Exec(query, endedAt, duration, restDuration, boolToInt(isLongBreak), sessionID)
	return err
}

// PauseSession marks an active session as paused
func (r *Repository) PauseSession(sessionID string) error {
	query := `UPDATE pomodoro_sessions SET paused_at = ?, rest_duration = 0 WHERE id = ?`
	_, err := r.db.Exec(query, time.Now(), sessionID)
	return err
}

// ResumeSession resumes a paused session, adjusting started_at
func (r *Repository) ResumeSession(sessionID string, pausedMillis int) error {
	// Adjust started_at forward by the pause duration
	query := `UPDATE pomodoro_sessions SET paused_at = NULL, started_at = datetime(started_at, '+' || ? || ' seconds') WHERE id = ?`
	seconds := float64(pausedMillis) / 1000.0
	_, err := r.db.Exec(query, seconds, sessionID)
	return err
}

// GetLatestSessionByMemberID returns the most recent session (active or completed)
func (r *Repository) GetLatestSessionByMemberID(memberID string) (*models.PomodoroSession, error) {
	query := `SELECT id, member_id, room_id, tag_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at, paused_at, rest_duration, is_long_break, planned_rest_duration, planned_long_break_duration, sessions_before_long_break, session_index FROM pomodoro_sessions WHERE member_id = ? ORDER BY started_at DESC LIMIT 1`
	row := r.db.QueryRow(query, memberID)
	return r.scanPomodoroSession(row)
}

func (r *Repository) GetTodaySessionsByMemberID(memberID string, date time.Time) ([]*models.PomodoroSession, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	query := `SELECT id, member_id, room_id, tag_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at, paused_at, rest_duration, is_long_break, planned_rest_duration, planned_long_break_duration, sessions_before_long_break, session_index FROM pomodoro_sessions WHERE member_id = ? AND started_at >= ? AND started_at < ? AND ended_at IS NOT NULL`
	rows, err := r.db.Query(query, memberID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.PomodoroSession
	for rows.Next() {
		session := &models.PomodoroSession{}
		var tagID, taskID, leaderID sql.NullString
		var endedAt, pausedAt sql.NullTime
		var isFollowed, isLongBreak int
		err := rows.Scan(&session.ID, &session.MemberID, &session.RoomID, &tagID, &taskID, &session.Duration, &session.PlannedDuration, &isFollowed, &leaderID, &session.StartedAt, &endedAt, &pausedAt, &session.RestDuration, &isLongBreak, &session.PlannedRestDuration, &session.PlannedLongBreakDuration, &session.SessionsBeforeLongBreak, &session.SessionIndex)
		if err != nil {
			return nil, err
		}
		session.IsFollowed = isFollowed == 1
		session.IsLongBreak = isLongBreak == 1
		if tagID.Valid {
			session.TagID = tagID.String
		}
		if taskID.Valid {
			session.TaskID = taskID.String
		}
		if leaderID.Valid {
			session.LeaderID = leaderID.String
		}
		if endedAt.Valid {
			session.EndedAt = &endedAt.Time
		}
		if pausedAt.Valid {
			session.PausedAt = &pausedAt.Time
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (r *Repository) scanPomodoroSession(row *sql.Row) (*models.PomodoroSession, error) {
	session := &models.PomodoroSession{}
	var tagID, taskID, leaderID sql.NullString
	var endedAt, pausedAt sql.NullTime
	var isFollowed, isLongBreak int
	err := row.Scan(&session.ID, &session.MemberID, &session.RoomID, &tagID, &taskID, &session.Duration, &session.PlannedDuration, &isFollowed, &leaderID, &session.StartedAt, &endedAt, &pausedAt, &session.RestDuration, &isLongBreak, &session.PlannedRestDuration, &session.PlannedLongBreakDuration, &session.SessionsBeforeLongBreak, &session.SessionIndex)
	if err != nil {
		return nil, err
	}
	session.IsFollowed = isFollowed == 1
	session.IsLongBreak = isLongBreak == 1
	if tagID.Valid {
		session.TagID = tagID.String
	}
	if taskID.Valid {
		session.TaskID = taskID.String
	}
	if leaderID.Valid {
		session.LeaderID = leaderID.String
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	if pausedAt.Valid {
		session.PausedAt = &pausedAt.Time
	}
	return session, nil
}

// UserStatus operations

func (r *Repository) UpsertUserStatus(status *models.UserStatus) error {
	query := `INSERT INTO user_statuses (id, member_id, room_id, emoji, message, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(member_id, room_id) DO UPDATE SET emoji = ?, message = ?, updated_at = ?`
	_, err := r.db.Exec(query, status.ID, status.MemberID, status.RoomID, status.Emoji, status.Message, status.UpdatedAt, status.Emoji, status.Message, status.UpdatedAt)
	return err
}

func (r *Repository) GetUserStatus(memberID, roomID string) (*models.UserStatus, error) {
	query := `SELECT id, member_id, room_id, emoji, message, updated_at FROM user_statuses WHERE member_id = ? AND room_id = ?`
	row := r.db.QueryRow(query, memberID, roomID)
	status := &models.UserStatus{}
	err := row.Scan(&status.ID, &status.MemberID, &status.RoomID, &status.Emoji, &status.Message, &status.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return status, nil
}

func (r *Repository) DeleteUserStatus(memberID, roomID string) error {
	query := `DELETE FROM user_statuses WHERE member_id = ? AND room_id = ?`
	_, err := r.db.Exec(query, memberID, roomID)
	return err
}

func (r *Repository) GetUserStatusesByRoomID(roomID string) ([]*models.UserStatus, error) {
	query := `SELECT id, member_id, room_id, emoji, message, updated_at FROM user_statuses WHERE room_id = ?`
	rows, err := r.db.Query(query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []*models.UserStatus
	for rows.Next() {
		status := &models.UserStatus{}
		err := rows.Scan(&status.ID, &status.MemberID, &status.RoomID, &status.Emoji, &status.Message, &status.UpdatedAt)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// Announcement operations

func (r *Repository) CreateAnnouncement(announcement *models.Announcement) error {
	query := `INSERT INTO announcements (id, room_id, sender_id, title, body, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, announcement.ID, announcement.RoomID, announcement.SenderID, announcement.Title, announcement.Body, announcement.CreatedAt)
	return err
}

func (r *Repository) GetAnnouncementsByRoomID(roomID string, limit int) ([]*models.Announcement, error) {
	query := `SELECT id, room_id, sender_id, title, body, created_at FROM announcements WHERE room_id = ? ORDER BY created_at DESC LIMIT ?`
	rows, err := r.db.Query(query, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var announcements []*models.Announcement
	for rows.Next() {
		a := &models.Announcement{}
		err := rows.Scan(&a.ID, &a.RoomID, &a.SenderID, &a.Title, &a.Body, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		announcements = append(announcements, a)
	}
	return announcements, nil
}

func (r *Repository) GetAnnouncementByID(id string) (*models.Announcement, error) {
	query := `SELECT id, room_id, sender_id, title, body, created_at FROM announcements WHERE id = ?`
	row := r.db.QueryRow(query, id)
	a := &models.Announcement{}
	err := row.Scan(&a.ID, &a.RoomID, &a.SenderID, &a.Title, &a.Body, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *Repository) DeleteAnnouncement(id string) error {
	query := `DELETE FROM announcements WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// Stats operations

func (r *Repository) GetMemberStats(memberID string) (int, int, error) {
	query := `SELECT COUNT(*), COALESCE(SUM(duration), 0) FROM pomodoro_sessions WHERE member_id = ? AND ended_at IS NOT NULL AND duration > 0`
	var totalPomodoros, totalDuration int
	err := r.db.QueryRow(query, memberID).Scan(&totalPomodoros, &totalDuration)
	return totalPomodoros, totalDuration, err
}

func (r *Repository) DeleteMemberPomodoroSessions(memberID string) error {
	_, err := r.db.Exec(`DELETE FROM pomodoro_sessions WHERE member_id = ?`, memberID)
	return err
}

func (r *Repository) GetRoomMembersStats(roomID string) (map[string][2]int, error) {
	query := `SELECT member_id, COUNT(*), COALESCE(SUM(duration), 0) FROM pomodoro_sessions WHERE room_id = ? AND ended_at IS NOT NULL AND duration > 0 GROUP BY member_id`
	rows, err := r.db.Query(query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][2]int)
	for rows.Next() {
		var memberID string
		var count, dur int
		if err := rows.Scan(&memberID, &count, &dur); err != nil {
			return nil, err
		}
		result[memberID] = [2]int{count, dur}
	}
	return result, nil
}

func (r *Repository) GetRoomMemberCount(roomID string) (int, error) {
	query := `SELECT COUNT(*) FROM room_members WHERE room_id = ?`
	var count int
	err := r.db.QueryRow(query, roomID).Scan(&count)
	return count, err
}

func (r *Repository) GetRoomOwner(roomID string) (*models.RoomMember, error) {
	query := `SELECT id, room_id, username, password_hash, is_owner, joined_at FROM room_members WHERE room_id = ? AND is_owner = 1 LIMIT 1`
	row := r.db.QueryRow(query, roomID)
	return r.scanMember(row)
}

func (r *Repository) GetTodayStats(roomID string, date time.Time) (int, int, int, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `SELECT COUNT(*), COALESCE(SUM(duration), 0), COUNT(DISTINCT member_id) FROM pomodoro_sessions WHERE room_id = ? AND started_at >= ? AND started_at < ? AND ended_at IS NOT NULL AND duration > 0`
	var totalPomodoros, totalDuration, activeUsers int
	err := r.db.QueryRow(query, roomID, startOfDay, endOfDay).Scan(&totalPomodoros, &totalDuration, &activeUsers)
	return totalPomodoros, totalDuration, activeUsers, err
}

// RefreshToken operations

func (r *Repository) CreateRefreshToken(token *models.RefreshToken) error {
	query := `INSERT INTO refresh_tokens (id, member_id, token_hash, expires_at, device_name, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, token.ID, token.MemberID, token.TokenHash, token.ExpiresAt, token.DeviceName, token.CreatedAt)
	return err
}

func (r *Repository) GetRefreshTokenByHash(hash string) (*models.RefreshToken, error) {
	query := `SELECT id, member_id, token_hash, expires_at, device_name, created_at, revoked_at FROM refresh_tokens WHERE token_hash = ?`
	row := r.db.QueryRow(query, hash)
	token := &models.RefreshToken{}
	var revokedAt sql.NullTime
	err := row.Scan(&token.ID, &token.MemberID, &token.TokenHash, &token.ExpiresAt, &token.DeviceName, &token.CreatedAt, &revokedAt)
	if err != nil {
		return nil, err
	}
	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}
	return token, nil
}

func (r *Repository) RevokeRefreshToken(id string) error {
	_, err := r.db.Exec(`UPDATE refresh_tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, time.Now(), id)
	return err
}

func (r *Repository) RevokeAllRefreshTokens(memberID string) (int64, error) {
	result, err := r.db.Exec(`UPDATE refresh_tokens SET revoked_at = ? WHERE member_id = ? AND revoked_at IS NULL`, time.Now(), memberID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) DeleteExpiredRefreshTokens() error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE expires_at < ?`, time.Now())
	if err != nil {
		log.Printf("Warning: failed to delete expired refresh tokens: %v", err)
	}
	return err
}

// Helper function

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
