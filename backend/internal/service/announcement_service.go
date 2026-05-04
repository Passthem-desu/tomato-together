package service

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

var (
	ErrMustBePersistentUser = errors.New("must_be_persistent_user")
)

type AnnouncementService struct {
	announcementRepo *repository.AnnouncementRepository
	roomRepo         *repository.RoomRepository
	roomMemberRepo   *repository.RoomMemberRepository
}

func NewAnnouncementService(
	announcementRepo *repository.AnnouncementRepository,
	roomRepo *repository.RoomRepository,
	roomMemberRepo *repository.RoomMemberRepository,
) *AnnouncementService {
	return &AnnouncementService{
		announcementRepo: announcementRepo,
		roomRepo:         roomRepo,
		roomMemberRepo:   roomMemberRepo,
	}
}

func (s *AnnouncementService) CreateAnnouncement(roomID, senderID, title, body string) (*models.Announcement, error) {
	announcement := &models.Announcement{
		ID:        uuid.New().String(),
		RoomID:    roomID,
		SenderID:  senderID,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now(),
	}
	if err := s.announcementRepo.Create(announcement); err != nil {
		return nil, err
	}
	return announcement, nil
}

func (s *AnnouncementService) GetAnnouncementsByRoom(roomID string, limit int) ([]*models.Announcement, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.announcementRepo.GetByRoomID(roomID, limit)
}

func (s *AnnouncementService) IsRoomOwner(roomID, userID string) (bool, error) {
	member, err := s.roomMemberRepo.GetByRoomAndUser(roomID, userID)
	if err != nil {
		return false, err
	}
	if member == nil {
		return false, nil
	}
	return member.IsOwner, nil
}