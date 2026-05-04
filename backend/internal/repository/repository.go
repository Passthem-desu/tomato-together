package repository

import (
	"database/sql"
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
	query := `INSERT INTO room_tokens (id, member_id, room_id, token, created_at, expires_at, last_heartbeat) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, token.ID, token.MemberID, token.RoomID, token.Token, token.CreatedAt, token.ExpiresAt, token.LastHeartbeat)
	return err
}

func (r *Repository) GetTokenByValue(tokenValue string) (*models.RoomToken, error) {
	query := `SELECT id, member_id, room_id, token, created_at, expires_at, last_heartbeat FROM room_tokens WHERE token = ?`
	row := r.db.QueryRow(query, tokenValue)
	return r.scanToken(row)
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
	query := `SELECT id, member_id, room_id, token, created_at, expires_at, last_heartbeat FROM room_tokens WHERE member_id = ? AND room_id = ?`
	row := r.db.QueryRow(query, memberID, roomID)
	return r.scanToken(row)
}

func (r *Repository) scanToken(row *sql.Row) (*models.RoomToken, error) {
	token := &models.RoomToken{}
	err := row.Scan(&token.ID, &token.MemberID, &token.RoomID, &token.Token, &token.CreatedAt, &token.ExpiresAt, &token.LastHeartbeat)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// Project operations

func (r *Repository) CreateProject(project *models.Project) error {
	query := `INSERT INTO projects (id, member_id, room_id, name, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, project.ID, project.MemberID, project.RoomID, project.Name, project.CreatedAt)
	return err
}

func (r *Repository) GetProjectByID(id string) (*models.Project, error) {
	query := `SELECT id, member_id, room_id, name, created_at FROM projects WHERE id = ?`
	row := r.db.QueryRow(query, id)
	project := &models.Project{}
	err := row.Scan(&project.ID, &project.MemberID, &project.RoomID, &project.Name, &project.CreatedAt)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (r *Repository) GetProjectsByMemberID(memberID string) ([]*models.Project, error) {
	query := `SELECT id, member_id, room_id, name, created_at FROM projects WHERE member_id = ?`
	return r.scanProjects(query, memberID)
}

func (r *Repository) GetProjectsByMemberAndRoom(memberID, roomID string) ([]*models.Project, error) {
	query := `SELECT id, member_id, room_id, name, created_at FROM projects WHERE member_id = ? AND room_id = ?`
	return r.scanProjects(query, memberID, roomID)
}

func (r *Repository) UpdateProject(projectID, name string) error {
	query := `UPDATE projects SET name = ? WHERE id = ?`
	_, err := r.db.Exec(query, name, projectID)
	return err
}

func (r *Repository) DeleteProject(projectID string) error {
	query := `DELETE FROM projects WHERE id = ?`
	_, err := r.db.Exec(query, projectID)
	return err
}

func (r *Repository) scanProjects(query string, args ...interface{}) ([]*models.Project, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		project := &models.Project{}
		err := rows.Scan(&project.ID, &project.MemberID, &project.RoomID, &project.Name, &project.CreatedAt)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, nil
}

// Task operations

func (r *Repository) CreateTask(task *models.Task) error {
	query := `INSERT INTO tasks (id, client_id, member_id, room_id, project_id, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, task.ID, task.ClientID, task.MemberID, task.RoomID, task.ProjectID, task.Title, task.Status, task.CreatedAt, task.UpdatedAt)
	return err
}

func (r *Repository) GetTaskByID(id string) (*models.Task, error) {
	query := `SELECT id, client_id, member_id, room_id, project_id, title, status, created_at, updated_at, completed_at FROM tasks WHERE id = ?`
	return r.scanTask(query, id)
}

func (r *Repository) GetTasksByMemberAndRoom(memberID, roomID string, status string) ([]*models.Task, error) {
	query := `SELECT id, client_id, member_id, room_id, project_id, title, status, created_at, updated_at, completed_at FROM tasks WHERE member_id = ? AND room_id = ?`
	args := []interface{}{memberID, roomID}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	return r.scanTasks(query, args...)
}

func (r *Repository) UpdateTask(taskID string, title, status, projectID string, completedAt *time.Time) error {
	query := `UPDATE tasks SET title = COALESCE(NULLIF(?, ''), title), status = COALESCE(NULLIF(?, ''), status), project_id = ?, updated_at = ?`
	args := []interface{}{title, status, projectID, time.Now()}
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
	var projectID sql.NullString
	var completedAt sql.NullTime
	err := row.Scan(&task.ID, &task.ClientID, &task.MemberID, &task.RoomID, &projectID, &task.Title, &task.Status, &task.CreatedAt, &task.UpdatedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	if projectID.Valid {
		task.ProjectID = projectID.String
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
		var projectID sql.NullString
		var completedAt sql.NullTime
		err := rows.Scan(&task.ID, &task.ClientID, &task.MemberID, &task.RoomID, &projectID, &task.Title, &task.Status, &task.CreatedAt, &task.UpdatedAt, &completedAt)
		if err != nil {
			return nil, err
		}
		if projectID.Valid {
			task.ProjectID = projectID.String
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
	query := `INSERT INTO pomodoro_sessions (id, member_id, room_id, project_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, session.ID, session.MemberID, session.RoomID, session.ProjectID, session.TaskID, session.Duration, session.PlannedDuration, boolToInt(session.IsFollowed), session.LeaderID, session.StartedAt, session.EndedAt)
	return err
}

func (r *Repository) GetActiveSessionByMemberID(memberID string) (*models.PomodoroSession, error) {
	query := `SELECT id, member_id, room_id, project_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at FROM pomodoro_sessions WHERE member_id = ? AND ended_at IS NULL`
	row := r.db.QueryRow(query, memberID)
	return r.scanPomodoroSession(row)
}

func (r *Repository) UpdatePomodoroSession(sessionID string, endedAt *time.Time, duration int) error {
	query := `UPDATE pomodoro_sessions SET ended_at = ?, duration = ? WHERE id = ?`
	_, err := r.db.Exec(query, endedAt, duration, sessionID)
	return err
}

func (r *Repository) GetTodaySessionsByMemberID(memberID string, date time.Time) ([]*models.PomodoroSession, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	query := `SELECT id, member_id, room_id, project_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at FROM pomodoro_sessions WHERE member_id = ? AND started_at >= ? AND started_at < ?`
	rows, err := r.db.Query(query, memberID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.PomodoroSession
	for rows.Next() {
		session := &models.PomodoroSession{}
		var projectID, taskID, leaderID sql.NullString
		var endedAt sql.NullTime
		var isFollowed int
		err := rows.Scan(&session.ID, &session.MemberID, &session.RoomID, &projectID, &taskID, &session.Duration, &session.PlannedDuration, &isFollowed, &leaderID, &session.StartedAt, &endedAt)
		if err != nil {
			return nil, err
		}
		session.IsFollowed = isFollowed == 1
		if projectID.Valid {
			session.ProjectID = projectID.String
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
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (r *Repository) scanPomodoroSession(row *sql.Row) (*models.PomodoroSession, error) {
	session := &models.PomodoroSession{}
	var projectID, taskID, leaderID sql.NullString
	var endedAt sql.NullTime
	var isFollowed int
	err := row.Scan(&session.ID, &session.MemberID, &session.RoomID, &projectID, &taskID, &session.Duration, &session.PlannedDuration, &isFollowed, &leaderID, &session.StartedAt, &endedAt)
	if err != nil {
		return nil, err
	}
	session.IsFollowed = isFollowed == 1
	if projectID.Valid {
		session.ProjectID = projectID.String
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

// Stats operations

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

	query := `SELECT COUNT(*), COALESCE(SUM(duration), 0), COUNT(DISTINCT member_id) FROM pomodoro_sessions WHERE room_id = ? AND started_at >= ? AND started_at < ? AND ended_at IS NOT NULL`
	var totalPomodoros, totalDuration, activeUsers int
	err := r.db.QueryRow(query, roomID, startOfDay, endOfDay).Scan(&totalPomodoros, &totalDuration, &activeUsers)
	return totalPomodoros, totalDuration, activeUsers, err
}

// Helper function

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
