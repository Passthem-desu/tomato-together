package repository

import (
	"database/sql"

	"tomatogether/backend/internal/models"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(task *models.Task) error {
	_, err := r.db.Exec(
		"INSERT INTO tasks (id, client_id, user_id, project_id, title, status, created_at, updated_at, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		task.ID, task.ClientID, task.UserID, task.ProjectID, task.Title, task.Status, task.CreatedAt, task.UpdatedAt, task.CompletedAt,
	)
	return err
}

func (r *TaskRepository) GetByID(id string) (*models.Task, error) {
	task := &models.Task{}
	var projectID sql.NullString
	var completedAt sql.NullTime
	err := r.db.QueryRow(
		"SELECT id, client_id, user_id, project_id, title, status, created_at, updated_at, completed_at FROM tasks WHERE id = ?",
		id,
	).Scan(&task.ID, &task.ClientID, &task.UserID, &projectID, &task.Title, &task.Status, &task.CreatedAt, &task.UpdatedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if projectID.Valid {
		task.ProjectID = &projectID.String
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	return task, err
}

func (r *TaskRepository) GetByClientID(clientID string) (*models.Task, error) {
	task := &models.Task{}
	var projectID sql.NullString
	var completedAt sql.NullTime
	err := r.db.QueryRow(
		"SELECT id, client_id, user_id, project_id, title, status, created_at, updated_at, completed_at FROM tasks WHERE client_id = ?",
		clientID,
	).Scan(&task.ID, &task.ClientID, &task.UserID, &projectID, &task.Title, &task.Status, &task.CreatedAt, &task.UpdatedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if projectID.Valid {
		task.ProjectID = &projectID.String
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	return task, err
}

func (r *TaskRepository) GetByUserID(userID string, status string) ([]*models.Task, error) {
	query := "SELECT id, client_id, user_id, project_id, title, status, created_at, updated_at, completed_at FROM tasks WHERE user_id = ?"
	args := []interface{}{userID}
	
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTasks(rows)
}

func (r *TaskRepository) GetByUserAndProject(userID, projectID string) ([]*models.Task, error) {
	rows, err := r.db.Query(
		"SELECT id, client_id, user_id, project_id, title, status, created_at, updated_at, completed_at FROM tasks WHERE user_id = ? AND project_id = ? ORDER BY created_at DESC",
		userID, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTasks(rows)
}

func (r *TaskRepository) Update(task *models.Task) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET title = ?, project_id = ?, status = ?, updated_at = ?, completed_at = ? WHERE id = ?",
		task.Title, task.ProjectID, task.Status, task.UpdatedAt, task.CompletedAt, task.ID,
	)
	return err
}

func (r *TaskRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

func (r *TaskRepository) CreateBatch(tasks []*models.Task) error {
	for _, task := range tasks {
		if err := r.Create(task); err != nil {
			return err
		}
	}
	return nil
}

func (r *TaskRepository) scanTasks(rows *sql.Rows) ([]*models.Task, error) {
	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		var projectID sql.NullString
		var completedAt sql.NullTime
		if err := rows.Scan(&task.ID, &task.ClientID, &task.UserID, &projectID, &task.Title, &task.Status, &task.CreatedAt, &task.UpdatedAt, &completedAt); err != nil {
			return nil, err
		}
		if projectID.Valid {
			task.ProjectID = &projectID.String
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}