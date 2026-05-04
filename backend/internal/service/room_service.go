package service

import (
	"time"

	"github.com/google/uuid"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

type RoomService struct {
	roomRepo       *repository.RoomRepository
	roomMemberRepo *repository.RoomMemberRepository
	userRepo       *repository.UserRepository
}

func NewRoomService(roomRepo *repository.RoomRepository, roomMemberRepo *repository.RoomMemberRepository, userRepo *repository.UserRepository) *RoomService {
	return &RoomService{
		roomRepo:       roomRepo,
		roomMemberRepo: roomMemberRepo,
		userRepo:       userRepo,
	}
}

func (s *RoomService) CreateRoom(name, password string, isReadonly bool, ownerID string) (*models.Room, error) {
	exists, err := s.roomRepo.ExistsByName(name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrRoomNameTaken
	}

	room := &models.Room{
		ID:         uuid.New().String(),
		Name:       name,
		Password:   password,
		IsReadonly: isReadonly,
		CreatedAt:  time.Now(),
	}

	if err := s.roomRepo.Create(room); err != nil {
		return nil, err
	}

	// Create owner membership
	member := &models.RoomMember{
		ID:       uuid.New().String(),
		RoomID:   room.ID,
		UserID:   ownerID,
		IsOwner:  true,
		JoinedAt: time.Now(),
	}
	if err := s.roomMemberRepo.Create(member); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *RoomService) GetRoomByName(name string) (*models.Room, error) {
	return s.roomRepo.GetByName(name)
}

func (s *RoomService) GetRoomInfo(name string) (*models.RoomInfo, error) {
	room, err := s.roomRepo.GetByName(name)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, nil
	}

	count, err := s.roomRepo.GetMemberCount(room.ID)
	if err != nil {
		return nil, err
	}

	info := &models.RoomInfo{
		ID:          room.ID,
		Name:        room.Name,
		IsReadonly:  room.IsReadonly,
		HasPassword: room.HasPassword,
		MemberCount: count,
		CreatedAt:   room.CreatedAt,
	}

	// Get owner info
	owner, err := s.roomMemberRepo.GetOwnerByRoomID(room.ID)
	if err == nil && owner != nil {
		user, err := s.userRepo.GetByID(owner.UserID)
		if err == nil && user != nil {
			info.Owner = &models.UserInfo{
				ID:       user.ID,
				Username: user.Username,
			}
		}
	}

	return info, nil
}

func (s *RoomService) JoinRoom(roomName, username string, password string, userID string) (*models.RoomToken, error) {
	room, err := s.roomRepo.GetByName(roomName)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}

	// Check room password
	if room.HasPassword && room.Password != password {
		return nil, ErrInvalidPassword
	}

	// Check if user is already in a room
	existing, err := s.roomMemberRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyInRoom
	}

	// Check if room has owner
	hasOwner, err := s.roomMemberRepo.HasOwner(room.ID)
	if err != nil {
		return nil, err
	}

	// Check if user is persistent
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	// Create membership
	isOwner := false
	if !hasOwner && user != nil && user.IsPersistent() {
		isOwner = true
	}

	member := &models.RoomMember{
		ID:       uuid.New().String(),
		RoomID:   room.ID,
		UserID:   userID,
		IsOwner:  isOwner,
		JoinedAt: time.Now(),
	}
	if err := s.roomMemberRepo.Create(member); err != nil {
		return nil, err
	}

	// Create room token
	expiry := getEnvInt("ROOM_TOKEN_EXPIRY", 86400)
	token := &models.RoomToken{
		ID:            uuid.New().String(),
		UserID:        userID,
		RoomID:        room.ID,
		Token:         uuid.New().String(),
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(time.Duration(expiry) * time.Second),
		LastHeartbeat: time.Now(),
	}
	if err := s.createToken(token); err != nil {
		return nil, err
	}

	return token, nil
}

func (s *RoomService) LeaveRoom(roomName, userID string) error {
	room, err := s.roomRepo.GetByName(roomName)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrRoomNotFound
	}

	// Delete membership (but keep is_owner flag)
	if err := s.roomMemberRepo.Delete(room.ID, userID); err != nil {
		return err
	}

	return nil
}

func (s *RoomService) UpdateRoomSettings(roomName string, password *string, isReadonly *bool) error {
	room, err := s.roomRepo.GetByName(roomName)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrRoomNotFound
	}

	if password != nil {
		room.Password = *password
		room.HasPassword = *password != ""
	}
	if isReadonly != nil {
		room.IsReadonly = *isReadonly
	}

	return s.roomRepo.Update(room)
}

func (s *RoomService) SetOwner(roomName, userID string, isOwner bool) error {
	room, err := s.roomRepo.GetByName(roomName)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrRoomNotFound
	}

	return s.roomMemberRepo.UpdateOwner(room.ID, userID, isOwner)
}

func (s *RoomService) GetRoomMembers(roomID string) ([]*models.RoomMember, error) {
	return s.roomMemberRepo.GetByRoomID(roomID)
}

func (s *RoomService) GetRoomMember(roomID, userID string) (*models.RoomMember, error) {
	return s.roomMemberRepo.GetByRoomAndUser(roomID, userID)
}

func (s *RoomService) GetRoomByID(id string) (*models.Room, error) {
	return s.roomRepo.GetByID(id)
}

func (s *RoomService) createToken(token *models.RoomToken) error {
	// Implementation in room_token_service
	return nil
}