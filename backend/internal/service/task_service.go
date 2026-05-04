package service

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

var (
	ErrTaskNotFound = errors.New("task_not_found")
)

type TaskService struct {
	taskRepo    *repository.TaskRepository
	projectRepo *repository.ProjectRepository
}

func NewTaskService(taskRepo *repository.TaskRepository, projectRepo *repository.ProjectRepository) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
	}
}

func (s *TaskService) CreateTask(userID, clientID, title string, projectID *string) (*models.Task, error) {
	task := &models.Task{
		ID:        uuid.New().String(),
		ClientID:  clientID,
		UserID:    userID,
		ProjectID: projectID,
		Title:     title,
		Status:    models.TaskStatusTODO,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.taskRepo.Create(task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) GetTask(id string) (*models.Task, error) {
	return s.taskRepo.GetByID(id)
}

func (s *TaskService) GetTaskByClientID(clientID string) (*models.Task, error) {
	return s.taskRepo.GetByClientID(clientID)
}

func (s *TaskService) GetTasksByUser(userID, status string) ([]*models.Task, error) {
	return s.taskRepo.GetByUserID(userID, status)
}

func (s *TaskService) GetTasksByProject(userID, projectID string) ([]*models.Task, error) {
	return s.taskRepo.GetByUserAndProject(userID, projectID)
}

func (s *TaskService) UpdateTask(id string, title *string, status *string, projectID *string) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return ErrTaskNotFound
	}

	if title != nil {
		task.Title = *title
	}
	if status != nil {
		// Handle completed_at logic
		if *status == models.TaskStatusDONE && task.Status != models.TaskStatusDONE {
			now := time.Now()
			task.CompletedAt = &now
		} else if *status != models.TaskStatusDONE && task.Status == models.TaskStatusDONE {
			task.CompletedAt = nil
		}
		task.Status = *status
	}
	if projectID != nil {
		task.ProjectID = projectID
	}
	task.UpdatedAt = time.Now()

	return s.taskRepo.Update(task)
}

func (s *TaskService) DeleteTask(id string) error {
	return s.taskRepo.Delete(id)
}

func (s *TaskService) SyncTasks(userID string, tasks []*models.Task) (map[string]string, error) {
	// client_id -> server_id mapping
	mapping := make(map[string]string)

	for _, task := range tasks {
		// Check if task with same client_id exists
		existing, err := s.taskRepo.GetByClientID(task.ClientID)
		if err != nil {
			return nil, err
		}

		if existing != nil {
			// Use existing server id
			mapping[task.ClientID] = existing.ID
		} else {
			// Create new task
			task.ID = uuid.New().String()
			task.UserID = userID
			task.CreatedAt = time.Now()
			task.UpdatedAt = time.Now()

			if err := s.taskRepo.Create(task); err != nil {
				return nil, err
			}
			mapping[task.ClientID] = task.ID
		}
	}

	return mapping, nil
}