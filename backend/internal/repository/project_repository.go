package repository

import (
	"database/sql"

	"tomatogether/backend/internal/models"
)

type ProjectRepository struct {
	db *sql.DB
}

func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(project *models.Project) error {
	_, err := r.db.Exec(
		"INSERT INTO projects (id, user_id, name, created_at) VALUES (?, ?, ?, ?)",
		project.ID, project.UserID, project.Name, project.CreatedAt,
	)
	return err
}

func (r *ProjectRepository) GetByID(id string) (*models.Project, error) {
	project := &models.Project{}
	err := r.db.QueryRow(
		"SELECT id, user_id, name, created_at FROM projects WHERE id = ?",
		id,
	).Scan(&project.ID, &project.UserID, &project.Name, &project.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return project, err
}

func (r *ProjectRepository) GetByUserID(userID string) ([]*models.Project, error) {
	rows, err := r.db.Query(
		"SELECT id, user_id, name, created_at FROM projects WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		project := &models.Project{}
		if err := rows.Scan(&project.ID, &project.UserID, &project.Name, &project.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func (r *ProjectRepository) Update(project *models.Project) error {
	_, err := r.db.Exec(
		"UPDATE projects SET name = ? WHERE id = ?",
		project.Name, project.ID,
	)
	return err
}

func (r *ProjectRepository) Delete(id string) error {
	// First set project_id to NULL for all tasks in this project
	_, err := r.db.Exec("UPDATE tasks SET project_id = NULL WHERE project_id = ?", id)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

func (r *ProjectRepository) GetTaskCount(projectID string) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM tasks WHERE project_id = ?",
		projectID,
	).Scan(&count)
	return count, err
}