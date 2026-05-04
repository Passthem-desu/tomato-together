package service

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

var (
	ErrProjectNotFound = errors.New("project_not_found")
)

type ProjectService struct {
	projectRepo *repository.ProjectRepository
}

func NewProjectService(projectRepo *repository.ProjectRepository) *ProjectService {
	return &ProjectService{projectRepo: projectRepo}
}

func (s *ProjectService) CreateProject(userID, name string) (*models.Project, error) {
	project := &models.Project{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      name,
		CreatedAt: time.Now(),
	}
	if err := s.projectRepo.Create(project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) GetProject(id string) (*models.Project, error) {
	return s.projectRepo.GetByID(id)
}

func (s *ProjectService) GetProjectsByUser(userID string) ([]*models.Project, error) {
	return s.projectRepo.GetByUserID(userID)
}

func (s *ProjectService) UpdateProject(id, name string) error {
	project, err := s.projectRepo.GetByID(id)
	if err != nil {
		return err
	}
	if project == nil {
		return ErrProjectNotFound
	}
	project.Name = name
	return s.projectRepo.Update(project)
}

func (s *ProjectService) DeleteProject(id string) error {
	return s.projectRepo.Delete(id)
}