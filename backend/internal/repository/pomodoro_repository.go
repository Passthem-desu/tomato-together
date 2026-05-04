package repository

import (
	"database/sql"
	"time"

	"tomatogether/backend/internal/models"
)

type PomodoroRepository struct {
	db *sql.DB
}

func NewPomodoroRepository(db *sql.DB) *PomodoroRepository {
	return &PomodoroRepository{db: db}
}

func (r *PomodoroRepository) Create(session *models.PomodoroSession) error {
	_, err := r.db.Exec(
		"INSERT INTO pomodoro_sessions (id, user_id, room_id, project_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		session.ID, session.UserID, session.RoomID, session.ProjectID, session.TaskID, session.Duration, session.PlannedDuration, session.IsFollowed, session.LeaderID, session.StartedAt, session.EndedAt,
	)
	return err
}

func (r *PomodoroRepository) GetByID(id string) (*models.PomodoroSession, error) {
	session := &models.PomodoroSession{}
	var projectID, taskID, leaderID sql.NullString
	var endedAt sql.NullTime
	err := r.db.QueryRow(
		"SELECT id, user_id, room_id, project_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at FROM pomodoro_sessions WHERE id = ?",
		id,
	).Scan(&session.ID, &session.UserID, &session.RoomID, &projectID, &taskID, &session.Duration, &session.PlannedDuration, &session.IsFollowed, &leaderID, &session.StartedAt, &endedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if projectID.Valid {
		session.ProjectID = &projectID.String
	}
	if taskID.Valid {
		session.TaskID = &taskID.String
	}
	if leaderID.Valid {
		session.LeaderID = &leaderID.String
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	return session, err
}

func (r *PomodoroRepository) GetActiveByUserID(userID string) (*models.PomodoroSession, error) {
	session := &models.PomodoroSession{}
	var projectID, taskID, leaderID sql.NullString
	err := r.db.QueryRow(
		"SELECT id, user_id, room_id, project_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at FROM pomodoro_sessions WHERE user_id = ? AND ended_at IS NULL",
		userID,
	).Scan(&session.ID, &session.UserID, &session.RoomID, &projectID, &taskID, &session.Duration, &session.PlannedDuration, &session.IsFollowed, &leaderID, &session.StartedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if projectID.Valid {
		session.ProjectID = &projectID.String
	}
	if taskID.Valid {
		session.TaskID = &taskID.String
	}
	if leaderID.Valid {
		session.LeaderID = &leaderID.String
	}
	return session, err
}

func (r *PomodoroRepository) GetActiveByRoomID(roomID string) ([]*models.PomodoroSession, error) {
	rows, err := r.db.Query(
		"SELECT id, user_id, room_id, project_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at FROM pomodoro_sessions WHERE room_id = ? AND ended_at IS NULL",
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.PomodoroSession
	for rows.Next() {
		session := &models.PomodoroSession{}
		var projectID, taskID, leaderID sql.NullString
		if err := rows.Scan(&session.ID, &session.UserID, &session.RoomID, &projectID, &taskID, &session.Duration, &session.PlannedDuration, &session.IsFollowed, &leaderID, &session.StartedAt); err != nil {
			return nil, err
		}
		if projectID.Valid {
			session.ProjectID = &projectID.String
		}
		if taskID.Valid {
			session.TaskID = &taskID.String
		}
		if leaderID.Valid {
			session.LeaderID = &leaderID.String
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (r *PomodoroRepository) Update(session *models.PomodoroSession) error {
	_, err := r.db.Exec(
		"UPDATE pomodoro_sessions SET duration = ?, ended_at = ? WHERE id = ?",
		session.Duration, session.EndedAt, session.ID,
	)
	return err
}

func (r *PomodoroRepository) GetTodayCountByUserID(userID string) (int, error) {
	var count int
	today := time.Now().Format("2006-01-02")
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM pomodoro_sessions WHERE user_id = ? AND DATE(started_at) = ?",
		userID, today,
	).Scan(&count)
	return count, err
}

func (r *PomodoroRepository) GetStatsByUserID(userID string, startDate, endDate time.Time) ([]*models.PomodoroSession, error) {
	rows, err := r.db.Query(
		"SELECT id, user_id, room_id, project_id, task_id, duration, planned_duration, is_followed, leader_id, started_at, ended_at FROM pomodoro_sessions WHERE user_id = ? AND started_at >= ? AND started_at <= ?",
		userID, startDate, endDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.PomodoroSession
	for rows.Next() {
		session := &models.PomodoroSession{}
		var projectID, taskID, leaderID sql.NullString
		var endedAt sql.NullTime
		if err := rows.Scan(&session.ID, &session.UserID, &session.RoomID, &projectID, &taskID, &session.Duration, &session.PlannedDuration, &session.IsFollowed, &leaderID, &session.StartedAt, &endedAt); err != nil {
			return nil, err
		}
		if projectID.Valid {
			session.ProjectID = &projectID.String
		}
		if taskID.Valid {
			session.TaskID = &taskID.String
		}
		if leaderID.Valid {
			session.LeaderID = &leaderID.String
		}
		if endedAt.Valid {
			session.EndedAt = &endedAt.Time
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (r *PomodoroRepository) GetRoomStats(roomID string, date string) (*RoomStats, error) {
	var totalPomodoros, totalDuration int
	var activeUsers int

	err := r.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(duration), 0), COUNT(DISTINCT user_id)
		FROM pomodoro_sessions
		WHERE room_id = ? AND DATE(started_at) = ?
	`, roomID, date).Scan(&totalPomodoros, &totalDuration, &activeUsers)
	if err != nil {
		return nil, err
	}
	return &RoomStats{
		TotalPomodoros: totalPomodoros,
		TotalDuration:   totalDuration,
		ActiveUsers:     activeUsers,
	}, nil
}

type RoomStats struct {
	TotalPomodoros int
	TotalDuration   int
	ActiveUsers     int
}