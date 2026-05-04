package service

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

var (
	ErrRoomNotFound   = errors.New("room_not_found")
	ErrRoomNameTaken  = errors.New("room_name_taken")
	ErrInvalidPassword = errors.New("invalid_password")
	ErrAlreadyInRoom  = errors.New("already_in_room")
	ErrNotRoomMember  = errors.New("not_room_member")
	ErrNotRoomOwner   = errors.New("not_room_owner")
	ErrCannotRemoveSelfOwner = errors.New("cannot_remove_self_owner")
	ErrRoomIsReadonly = errors.New("room_is_readonly")
	ErrRoomRequiresPassword = errors.New("room_requires_password")
)

type RoomTokenService struct {
	roomTokenRepo *repository.RoomTokenRepository
}

func NewRoomTokenService(roomTokenRepo *repository.RoomTokenRepository) *RoomTokenService {
	return &RoomTokenService{roomTokenRepo: roomTokenRepo}
}

func (s *RoomTokenService) CreateToken(userID, roomID string) (*models.RoomToken, error) {
	expiry := getEnvInt("ROOM_TOKEN_EXPIRY", 86400)
	token := &models.RoomToken{
		ID:            uuid.New().String(),
		UserID:        userID,
		RoomID:        roomID,
		Token:         uuid.New().String(),
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(time.Duration(expiry) * time.Second),
		LastHeartbeat: time.Now(),
	}
	if err := s.roomTokenRepo.Create(token); err != nil {
		return nil, err
	}
	return token, nil
}

func (s *RoomTokenService) GetByToken(token string) (*models.RoomToken, error) {
	return s.roomTokenRepo.GetByToken(token)
}

func (s *RoomTokenService) DeleteByUserAndRoom(userID, roomID string) error {
	return s.roomTokenRepo.DeleteByUserAndRoom(userID, roomID)
}

func (s *RoomTokenService) UpdateHeartbeat(token string) error {
	return s.roomTokenRepo.UpdateHeartbeat(token)
}

func (s *RoomTokenService) GetExpiredTokens() ([]*models.RoomToken, error) {
	return s.roomTokenRepo.GetExpiredTokens()
}

func (s *RoomTokenService) GetStaleTokens(timeout time.Duration) ([]*models.RoomToken, error) {
	return s.roomTokenRepo.GetStaleTokens(timeout)
}