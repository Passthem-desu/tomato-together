package service

import (
	"time"

	"github.com/google/uuid"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

type UserStatusService struct {
	statusRepo *repository.UserStatusRepository
}

func NewUserStatusService(statusRepo *repository.UserStatusRepository) *UserStatusService {
	return &UserStatusService{statusRepo: statusRepo}
}

func (s *UserStatusService) UpdateStatus(userID, roomID, emoji, message string) error {
	status := &models.UserStatus{
		ID:        uuid.New().String(),
		UserID:    userID,
		RoomID:    roomID,
		Emoji:     emoji,
		Message:   message,
		UpdatedAt: time.Now(),
	}
	return s.statusRepo.Upsert(status)
}

func (s *UserStatusService) GetStatus(userID, roomID string) (*models.UserStatus, error) {
	return s.statusRepo.GetByUserAndRoom(userID, roomID)
}

func (s *UserStatusService) ClearStatus(userID, roomID string) error {
	return s.statusRepo.Delete(userID, roomID)
}

func (s *UserStatusService) GetStatusesByRoom(roomID string) ([]*models.UserStatus, error) {
	return s.statusRepo.GetByRoomID(roomID)
}