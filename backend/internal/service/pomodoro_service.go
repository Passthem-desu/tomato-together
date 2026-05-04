package service

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

var (
	ErrNoActiveSession = errors.New("no_active_session")
	ErrAlreadyFollowing = errors.New("already_following")
	ErrNotFollowing    = errors.New("not_following")
	ErrSessionNotActive = errors.New("session_not_active")
)

type PomodoroService struct {
	pomodoroRepo   *repository.PomodoroRepository
	roomMemberRepo *repository.RoomMemberRepository
	userRepo       *repository.UserRepository
}

func NewPomodoroService(pomodoroRepo *repository.PomodoroRepository, roomMemberRepo *repository.RoomMemberRepository) *PomodoroService {
	return &PomodoroService{
		pomodoroRepo:   pomodoroRepo,
		roomMemberRepo: roomMemberRepo,
	}
}

func (s *PomodoroService) StartPomodoro(userID, roomID string, projectID, taskID *string, plannedDuration, restDuration, longBreakDuration, sessionsBeforeLongBreak int) (*models.PomodoroSession, error) {
	// Check if user already has an active session
	active, err := s.pomodoroRepo.GetActiveByUserID(userID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, ErrAlreadyFollowing // or a more specific error
	}

	session := &models.PomodoroSession{
		ID:              uuid.New().String(),
		UserID:          userID,
		RoomID:          roomID,
		ProjectID:       projectID,
		TaskID:          taskID,
		Duration:        0,
		PlannedDuration: plannedDuration,
		IsFollowed:      false,
		StartedAt:       time.Now(),
	}

	if err := s.pomodoroRepo.Create(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *PomodoroService) FollowPomodoro(userID, roomID, leaderID string) (*models.PomodoroSession, error) {
	// Get leader's active session
	leaderSession, err := s.pomodoroRepo.GetActiveByUserID(leaderID)
	if err != nil {
		return nil, err
	}
	if leaderSession == nil {
		return nil, ErrNoActiveSession
	}

	// Check if user already has an active session
	active, err := s.pomodoroRepo.GetActiveByUserID(userID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, ErrAlreadyFollowing
	}

	// Create following session
	session := &models.PomodoroSession{
		ID:              uuid.New().String(),
		UserID:          userID,
		RoomID:          roomID,
		PlannedDuration: leaderSession.PlannedDuration,
		IsFollowed:      true,
		LeaderID:        &leaderID,
		StartedAt:       leaderSession.StartedAt,
	}

	if err := s.pomodoroRepo.Create(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *PomodoroService) UnfollowPomodoro(userID string) error {
	session, err := s.pomodoroRepo.GetActiveByUserID(userID)
	if err != nil {
		return err
	}
	if session == nil || !session.IsFollowed {
		return ErrNotFollowing
	}

	// End the session
	now := time.Now()
	session.Duration = int(now.Sub(session.StartedAt).Seconds())
	session.EndedAt = &now
	return s.pomodoroRepo.Update(session)
}

func (s *PomodoroService) EndPomodoro(userID string, aborted bool) (*models.PomodoroSession, error) {
	session, err := s.pomodoroRepo.GetActiveByUserID(userID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrNoActiveSession
	}

	now := time.Now()
	session.Duration = int(now.Sub(session.StartedAt).Seconds())
	session.EndedAt = &now

	if err := s.pomodoroRepo.Update(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *PomodoroService) GetActiveSession(userID string) (*models.PomodoroSession, error) {
	return s.pomodoroRepo.GetActiveByUserID(userID)
}

func (s *PomodoroService) GetActiveSessionsByRoom(roomID string) ([]*models.PomodoroSession, error) {
	return s.pomodoroRepo.GetActiveByRoomID(roomID)
}

func (s *PomodoroService) GetTodayCount(userID string) (int, error) {
	return s.pomodoroRepo.GetTodayCountByUserID(userID)
}

func (s *PomodoroService) GetRoomStats(roomID, date string) (*repository.RoomStats, error) {
	return s.pomodoroRepo.GetRoomStats(roomID, date)
}

func (s *PomodoroService) GetUserByID(userID string) (*models.User, error) {
	return s.userRepo.GetByID(userID)
}