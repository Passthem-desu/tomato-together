package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

var (
	ErrRoomNotFound        = errors.New("room_not_found")
	ErrMemberNotFound     = errors.New("member_not_found")
	ErrInvalidPassword    = errors.New("invalid_password")
	ErrInvalidRoomPassword = errors.New("invalid_room_password")
	ErrRoomRequiresPassword = errors.New("room_requires_password")
	ErrRoomIsReadonly     = errors.New("room_is_readonly")
	ErrUsernameTaken     = errors.New("username_taken")
	ErrRoomNameTaken     = errors.New("room_name_taken")
	ErrNoActiveSession    = errors.New("no_active_session")
	ErrAlreadyFollowing   = errors.New("already_following")
	ErrNotFollowing       = errors.New("not_following")
	ErrNotRoomOwner       = errors.New("not_room_owner")
	ErrSessionNotActive   = errors.New("session_not_active")
	ErrTokenExpired       = errors.New("token_expired")
	ErrTokenInvalid       = errors.New("token_invalid")
	ErrMustBePersistent   = errors.New("must_be_persistent_user")
	ErrMustBeOwner        = errors.New("must_be_owner")
	ErrProjectNotFound    = errors.New("project_not_found")
	ErrTaskNotFound       = errors.New("task_not_found")
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// Token expiry duration
const tokenExpiry = 24 * time.Hour

// Room operations

func (s *Service) CreateRoom(req *models.CreateRoomRequest) (*models.RoomResponse, error) {
	// Check if room name is taken
	_, err := s.repo.GetRoomByName(req.RoomName)
	if err == nil {
		return nil, ErrRoomNameTaken
	}

	// Generate room ID
	roomID := uuid.New().String()

	// Hash room password if provided
	var roomPasswordHash string
	if req.RoomPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.RoomPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		roomPasswordHash = string(hash)
	}

	// Create room
	room := &models.Room{
		ID:           roomID,
		Name:         req.RoomName,
		PasswordHash: roomPasswordHash,
		IsReadonly:   req.IsReadonly,
		CreatedAt:    time.Now(),
	}
	if err := s.repo.CreateRoom(room); err != nil {
		return nil, err
	}

	// Generate member ID
	memberID := uuid.New().String()

	// Hash user password if provided
	var userPasswordHash string
	isPersistent := false
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		userPasswordHash = string(hash)
		isPersistent = true
	}

	// Create owner member
	member := &models.RoomMember{
		ID:           memberID,
		RoomID:       roomID,
		Username:     req.Username,
		PasswordHash: userPasswordHash,
		IsOwner:      true,
		JoinedAt:     time.Now(),
		IsPersistent: isPersistent,
	}
	if err := s.repo.CreateMember(member); err != nil {
		return nil, err
	}

	// Create token
	tokenValue, err := generateToken()
	if err != nil {
		return nil, err
	}
	token := &models.RoomToken{
		ID:            uuid.New().String(),
		MemberID:      memberID,
		RoomID:        roomID,
		Token:         tokenValue,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(tokenExpiry),
		LastHeartbeat: time.Now(),
	}
	if err := s.repo.CreateToken(token); err != nil {
		return nil, err
	}

	return &models.RoomResponse{
		Room: &models.RoomInfo{
			ID:          room.ID,
			Name:        room.Name,
			IsReadonly:  room.IsReadonly,
			HasPassword: room.PasswordHash != "",
		},
		Member: &models.MemberInfo{
			ID:           member.ID,
			Username:     member.Username,
			IsOwner:      member.IsOwner,
			IsPersistent: member.IsPersistent,
		},
		Token: token.Token,
	}, nil
}

func (s *Service) JoinRoom(roomName string, req *models.JoinRoomRequest) (*models.RoomResponse, error) {
	// Get room
	room, err := s.repo.GetRoomByName(roomName)
	if err != nil {
		return nil, ErrRoomNotFound
	}

	// Check room password
	if room.PasswordHash != "" {
		if req.RoomPassword == "" || !checkPassword(req.RoomPassword, room.PasswordHash) {
			return nil, ErrInvalidRoomPassword
		}
	}

	// Check if readonly
	if room.IsReadonly {
		return nil, ErrRoomIsReadonly
	}

	// Check if username is already taken
	existingMember, err := s.repo.GetMemberByUsername(room.ID, req.Username)
	if err == nil && existingMember != nil {
		// Username exists in this room
		if existingMember.PasswordHash != "" {
			// Existing persistent user - must verify password
			if req.Password == "" || !checkPassword(req.Password, existingMember.PasswordHash) {
				return nil, ErrInvalidPassword
			}
			// Password correct - return existing user
			tokenValue, err := generateToken()
			if err != nil {
				return nil, err
			}
			// Delete existing token if any (due to UNIQUE constraint)
			s.repo.DeleteTokenByMemberAndRoom(existingMember.ID, room.ID)
			token := &models.RoomToken{
				ID:            uuid.New().String(),
				MemberID:      existingMember.ID,
				RoomID:        room.ID,
				Token:         tokenValue,
				CreatedAt:     time.Now(),
				ExpiresAt:     time.Now().Add(tokenExpiry),
				LastHeartbeat: time.Now(),
			}
			if err := s.repo.CreateToken(token); err != nil {
				return nil, err
			}
			return &models.RoomResponse{
				Room: &models.RoomInfo{
					ID:          room.ID,
					Name:        room.Name,
					IsReadonly:  room.IsReadonly,
					HasPassword: room.PasswordHash != "",
				},
				Member: &models.MemberInfo{
					ID:           existingMember.ID,
					Username:     existingMember.Username,
					IsOwner:      existingMember.IsOwner,
					IsPersistent: existingMember.IsPersistent,
				},
				Token: token.Token,
			}, nil
		} else {
			// Username taken by anonymous user - cannot join with same username
			return nil, ErrUsernameTaken
		}
	}

	// Username not taken - create new member
	memberID := uuid.New().String()

	// Hash user password if provided
	var userPasswordHash string
	isPersistent := false
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		userPasswordHash = string(hash)
		isPersistent = true
	}

	// New users are NEVER owners (only room creator can be owner)
	member := &models.RoomMember{
		ID:           memberID,
		RoomID:       room.ID,
		Username:     req.Username,
		PasswordHash: userPasswordHash,
		IsOwner:      false, // New joining users are never owners
		JoinedAt:     time.Now(),
		IsPersistent: isPersistent,
	}
	if err := s.repo.CreateMember(member); err != nil {
		return nil, err
	}

	// Create token
	tokenValue, err := generateToken()
	if err != nil {
		return nil, err
	}
	token := &models.RoomToken{
		ID:            uuid.New().String(),
		MemberID:      memberID,
		RoomID:        room.ID,
		Token:         tokenValue,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(tokenExpiry),
		LastHeartbeat: time.Now(),
	}
	if err := s.repo.CreateToken(token); err != nil {
		return nil, err
	}

	return &models.RoomResponse{
		Room: &models.RoomInfo{
			ID:          room.ID,
			Name:        room.Name,
			IsReadonly:  room.IsReadonly,
			HasPassword: room.PasswordHash != "",
		},
		Member: &models.MemberInfo{
			ID:           member.ID,
			Username:     member.Username,
			IsOwner:      member.IsOwner,
			IsPersistent: member.IsPersistent,
		},
		Token: token.Token,
	}, nil
}

func (s *Service) LeaveRoom(tokenValue string) error {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	// Only delete token, member record is kept
	return s.repo.DeleteToken(token.ID)
}

func (s *Service) GetRoomInfo(roomName string) (*models.Room, error) {
	room, err := s.repo.GetRoomByName(roomName)
	if err != nil {
		return nil, ErrRoomNotFound
	}
	return room, nil
}

func (s *Service) GetRoomUsers(tokenValue string) ([]*models.UserInfo, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	members, err := s.repo.GetMembersByRoomID(token.RoomID)
	if err != nil {
		return nil, err
	}

	// Get all statuses for the room
	statuses, err := s.repo.GetUserStatusesByRoomID(token.RoomID)
	if err != nil {
		return nil, err
	}
	statusMap := make(map[string]*models.UserStatus)
	for _, status := range statuses {
		statusMap[status.MemberID] = status
	}

	var users []*models.UserInfo
	for _, member := range members {
		user := &models.UserInfo{
			ID:          member.ID,
			Username:    member.Username,
			IsOwner:     member.IsOwner,
			IsPersistent: member.IsPersistent,
			IsOnline:    true, // All members in the room are considered online for now
		}

		// Get status
		if status, ok := statusMap[member.ID]; ok {
			user.Status = &models.StatusInfo{
				Emoji:   status.Emoji,
				Message: status.Message,
			}
		}

		// Get active session
		session, err := s.repo.GetActiveSessionByMemberID(member.ID)
		if err == nil && session != nil {
			user.Pomodoro = &models.PomodoroInfo{
				IsActive: true,
			}
			if session.IsFollowed && session.LeaderID != "" {
				leader, _ := s.repo.GetMemberByID(session.LeaderID)
				if leader != nil {
					user.Pomodoro.IsFollowing = true
					user.Pomodoro.LeaderUsername = leader.Username
				}
			}
		}

		users = append(users, user)
	}

	return users, nil
}

func (s *Service) GetRoomMemberCount(roomID string) (int, error) {
	return s.repo.GetRoomMemberCount(roomID)
}

func (s *Service) GetRoomOwner(roomID string) (*models.RoomMember, error) {
	return s.repo.GetRoomOwner(roomID)
}

func (s *Service) GetRoomStats(roomID string, date time.Time) (int, int, int, error) {
	return s.repo.GetTodayStats(roomID, date)
}

func (s *Service) CheckUsername(roomName, username string) (*models.UserCheckResponse, error) {
	room, err := s.repo.GetRoomByName(roomName)
	if err != nil {
		return nil, ErrRoomNotFound
	}

	member, err := s.repo.GetMemberByUsername(room.ID, username)
	if err != nil {
		// User not found - this is OK for new users
		return &models.UserCheckResponse{
			Exists:       false,
			IsPersistent: false,
		}, nil
	}

	return &models.UserCheckResponse{
		Exists:       true,
		IsPersistent: member.IsPersistent,
	}, nil
}

func (s *Service) UpdateRoomSettings(tokenValue string, req *models.UpdateRoomSettingsRequest) error {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	// Verify owner
	member, err := s.repo.GetMemberByID(token.MemberID)
	if err != nil {
		return err
	}
	if !member.IsOwner {
		return ErrMustBeOwner
	}

	var passwordHash string
	if req.RoomPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.RoomPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		passwordHash = string(hash)
	}

	return s.repo.UpdateRoomSettings(token.RoomID, passwordHash, req.IsReadonly)
}

func (s *Service) SetRoomOwner(tokenValue, targetMemberID string, req *models.SetOwnerRequest) error {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	// Verify owner
	member, err := s.repo.GetMemberByID(token.MemberID)
	if err != nil {
		return err
	}
	if !member.IsOwner {
		return ErrMustBeOwner
	}

	// Verify target is persistent
	target, err := s.repo.GetMemberByID(targetMemberID)
	if err != nil {
		return ErrMemberNotFound
	}
	if !target.IsPersistent {
		return ErrMustBePersistent
	}

	return s.repo.SetMemberOwner(targetMemberID, req.IsOwner)
}

// Member operations

func (s *Service) GetMe(tokenValue string) (*models.RoomMember, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	return s.repo.GetMemberByID(token.MemberID)
}

func (s *Service) UpgradeToPersistent(tokenValue string, req *models.UpgradeRequest) error {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repo.UpdateMemberPassword(token.MemberID, string(hash))
}

func (s *Service) Login(req *models.LoginRequest) (*models.RoomResponse, error) {
	// Get room
	room, err := s.repo.GetRoomByName(req.RoomName)
	if err != nil {
		return nil, ErrRoomNotFound
	}

	// Check room password
	if room.PasswordHash != "" {
		if req.RoomPassword == "" || !checkPassword(req.RoomPassword, room.PasswordHash) {
			return nil, ErrInvalidRoomPassword
		}
	}

	// Get member
	member, err := s.repo.GetMemberByUsername(room.ID, req.Username)
	if err != nil {
		return nil, ErrMemberNotFound
	}

	// Verify password
	if member.PasswordHash == "" || !checkPassword(req.Password, member.PasswordHash) {
		return nil, ErrInvalidPassword
	}

	// Create new token
	tokenValue, err := generateToken()
	if err != nil {
		return nil, err
	}
	token := &models.RoomToken{
		ID:            uuid.New().String(),
		MemberID:      member.ID,
		RoomID:        room.ID,
		Token:         tokenValue,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(tokenExpiry),
		LastHeartbeat: time.Now(),
	}
	if err := s.repo.CreateToken(token); err != nil {
		return nil, err
	}

	return &models.RoomResponse{
		Room: &models.RoomInfo{
			ID:          room.ID,
			Name:        room.Name,
			IsReadonly:  room.IsReadonly,
			HasPassword: room.PasswordHash != "",
		},
		Member: &models.MemberInfo{
			ID:           member.ID,
			Username:     member.Username,
			IsOwner:      member.IsOwner,
			IsPersistent: member.IsPersistent,
		},
		Token: token.Token,
	}, nil
}

// Pomodoro operations

func (s *Service) StartPomodoro(tokenValue string, req *models.StartPomodoroRequest) (*models.PomodoroStatusResponse, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	// Check if already in a session
	existing, err := s.repo.GetActiveSessionByMemberID(token.MemberID)
	if err == nil && existing != nil {
		return nil, ErrAlreadyFollowing
	}

	// Set default values
	plannedDuration := req.PlannedDuration
	if plannedDuration == 0 {
		plannedDuration = 1500
	}
	restDuration := req.RestDuration
	if restDuration == 0 {
		restDuration = 300
	}
	longBreakDuration := req.LongBreakDuration
	if longBreakDuration == 0 {
		longBreakDuration = 900
	}
	sessionsBeforeLongBreak := req.SessionsBeforeLongBreak
	if sessionsBeforeLongBreak == 0 {
		sessionsBeforeLongBreak = 4
	}

	// Create session
	session := &models.PomodoroSession{
		ID:              uuid.New().String(),
		MemberID:        token.MemberID,
		RoomID:          token.RoomID,
		ProjectID:       req.ProjectID,
		TaskID:          req.TaskID,
		PlannedDuration: plannedDuration,
		IsFollowed:      false,
		StartedAt:       time.Now(),
	}
	if err := s.repo.CreatePomodoroSession(session); err != nil {
		return nil, err
	}

	// Count today's sessions
	return &models.PomodoroStatusResponse{
		IsActive:          true,
		Status:            "focusing",
		SessionID:         session.ID,
		StartedAt:         session.StartedAt.Format(time.RFC3339),
		RemainingSeconds:  plannedDuration,
		PlannedDuration:   plannedDuration,
		RestDuration:      restDuration,
		LongBreakDuration: longBreakDuration,
		SessionsBeforeLongBreak: sessionsBeforeLongBreak,
	}, nil
}

func (s *Service) FollowPomodoro(tokenValue string, req *models.FollowPomodoroRequest) (*models.PomodoroStatusResponse, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	// Check if already following
	existing, err := s.repo.GetActiveSessionByMemberID(token.MemberID)
	if err == nil && existing != nil {
		return nil, ErrAlreadyFollowing
	}

	// Get leader session
	leaderSession, err := s.repo.GetActiveSessionByMemberID(req.LeaderID)
	if err != nil {
		return nil, ErrNoActiveSession
	}

	// Get leader info
	leader, err := s.repo.GetMemberByID(req.LeaderID)
	if err != nil {
		return nil, ErrMemberNotFound
	}

	// Calculate remaining seconds
	elapsed := int(time.Since(leaderSession.StartedAt).Seconds())
	remaining := leaderSession.PlannedDuration - elapsed
	if remaining < 0 {
		remaining = 0
	}

	// Create follow session
	session := &models.PomodoroSession{
		ID:              uuid.New().String(),
		MemberID:        token.MemberID,
		RoomID:          token.RoomID,
		PlannedDuration: leaderSession.PlannedDuration,
		IsFollowed:      true,
		LeaderID:        req.LeaderID,
		StartedAt:       leaderSession.StartedAt,
	}
	if err := s.repo.CreatePomodoroSession(session); err != nil {
		return nil, err
	}

	return &models.PomodoroStatusResponse{
		IsActive:          true,
		Status:            "following",
		SessionID:         session.ID,
		StartedAt:         leaderSession.StartedAt.Format(time.RFC3339),
		RemainingSeconds:  remaining,
		LeaderID:          req.LeaderID,
		LeaderUsername:    leader.Username,
		PlannedDuration:   leaderSession.PlannedDuration,
	}, nil
}

func (s *Service) UnfollowPomodoro(tokenValue string, req *models.FollowRoomRequest) (string, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return "", err
	}

	// Get active session
	session, err := s.repo.GetActiveSessionByMemberID(token.MemberID)
	if err != nil || session == nil {
		return "", ErrNotFollowing
	}

	if !session.IsFollowed {
		return "", ErrNotFollowing
	}

	// End the session
	now := time.Now()
	duration := int(now.Sub(session.StartedAt).Seconds())
	if err := s.repo.UpdatePomodoroSession(session.ID, &now, duration); err != nil {
		return "", err
	}

	return "idle", nil
}

func (s *Service) EndPomodoro(tokenValue string, req *models.EndPomodoroRequest) (*models.PomodoroStatusResponse, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	// Get active session
	session, err := s.repo.GetActiveSessionByMemberID(token.MemberID)
	if err != nil || session == nil {
		return nil, ErrNoActiveSession
	}

	// End the session
	now := time.Now()
	duration := int(now.Sub(session.StartedAt).Seconds())
	if err := s.repo.UpdatePomodoroSession(session.ID, &now, duration); err != nil {
		return nil, err
	}

	// Count today's sessions
	sessions, _ := s.repo.GetTodaySessionsByMemberID(token.MemberID, time.Now())
	sessionsCompleted := len(sessions)

	// Determine if should take long break
	longBreakDuration := 900
	restDuration := 300
	shouldTakeLongBreak := sessionsCompleted%4 == 0 && sessionsCompleted > 0

	if shouldTakeLongBreak {
		restDuration = longBreakDuration
	}

	return &models.PomodoroStatusResponse{
		IsActive:        false,
		Status:          "rest",
		SessionID:       session.ID,
		Duration:        duration,
		PlannedDuration: session.PlannedDuration,
		RestDuration:    restDuration,
		LongBreakDuration: longBreakDuration,
		ShouldTakeLongBreak: shouldTakeLongBreak,
		SessionsCompleted: sessionsCompleted,
	}, nil
}

func (s *Service) GetPomodoroStatus(tokenValue string) (*models.PomodoroStatusResponse, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	// Get active session
	session, err := s.repo.GetActiveSessionByMemberID(token.MemberID)
	if err != nil || session == nil {
		return &models.PomodoroStatusResponse{
			IsActive: false,
			Status:   "idle",
		}, nil
	}

	// Calculate remaining seconds
	elapsed := int(time.Since(session.StartedAt).Seconds())
	remaining := session.PlannedDuration - elapsed
	if remaining < 0 {
		remaining = 0
	}

	response := &models.PomodoroStatusResponse{
		IsActive:         true,
		Status:           "focusing",
		SessionID:        session.ID,
		StartedAt:        session.StartedAt.Format(time.RFC3339),
		RemainingSeconds: remaining,
		PlannedDuration:  session.PlannedDuration,
	}

	if session.IsFollowed && session.LeaderID != "" {
		leader, _ := s.repo.GetMemberByID(session.LeaderID)
		if leader != nil {
			response.LeaderID = session.LeaderID
			response.LeaderUsername = leader.Username
			response.Status = "following"
		}
	}

	return response, nil
}

// Status operations

func (s *Service) UpdateStatus(tokenValue string, req *models.UpdateStatusRequest) (*models.UserStatus, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	status := &models.UserStatus{
		ID:        uuid.New().String(),
		MemberID:  token.MemberID,
		RoomID:    token.RoomID,
		Emoji:     req.Emoji,
		Message:   req.Message,
		UpdatedAt: time.Now(),
	}

	if err := s.repo.UpsertUserStatus(status); err != nil {
		return nil, err
	}

	return status, nil
}

func (s *Service) DeleteStatus(tokenValue string) error {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	return s.repo.DeleteUserStatus(token.MemberID, token.RoomID)
}

// Token validation

func (s *Service) ValidateToken(tokenValue string) (*models.RoomToken, error) {
	token, err := s.repo.GetTokenByValue(tokenValue)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	if time.Now().After(token.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return token, nil
}

func (s *Service) RefreshToken(tokenValue string) error {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	return s.repo.UpdateTokenHeartbeat(token.ID)
}

// Project operations

func (s *Service) GetProjects(memberID, roomID string) ([]*models.Project, error) {
	return s.repo.GetProjectsByMemberAndRoom(memberID, roomID)
}

func (s *Service) CreateProject(memberID, roomID, name string) (*models.Project, error) {
	project := &models.Project{
		ID:        uuid.New().String(),
		MemberID:  memberID,
		RoomID:    roomID,
		Name:      name,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateProject(project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *Service) UpdateProject(projectID, memberID, name string) error {
	project, err := s.repo.GetProjectByID(projectID)
	if err != nil || project.MemberID != memberID {
		return ErrProjectNotFound
	}
	return s.repo.UpdateProject(projectID, name)
}

func (s *Service) DeleteProject(projectID, memberID string) error {
	project, err := s.repo.GetProjectByID(projectID)
	if err != nil || project.MemberID != memberID {
		return ErrProjectNotFound
	}
	return s.repo.DeleteProject(projectID)
}

// Task operations

func (s *Service) GetTasks(memberID, roomID, status string) ([]*models.Task, error) {
	return s.repo.GetTasksByMemberAndRoom(memberID, roomID, status)
}

func (s *Service) CreateTask(memberID, roomID, clientID, title, projectID string) (*models.Task, error) {
	task := &models.Task{
		ID:        uuid.New().String(),
		ClientID:  clientID,
		MemberID:  memberID,
		RoomID:    roomID,
		ProjectID: projectID,
		Title:     title,
		Status:    "TODO",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.CreateTask(task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Service) UpdateTask(taskID, memberID, title, status, projectID string) error {
	task, err := s.repo.GetTaskByID(taskID)
	if err != nil || task.MemberID != memberID {
		return ErrTaskNotFound
	}
	var completedAt *time.Time
	if status == "DONE" {
		now := time.Now()
		completedAt = &now
	}
	return s.repo.UpdateTask(taskID, title, status, projectID, completedAt)
}

func (s *Service) DeleteTask(taskID, memberID string) error {
	task, err := s.repo.GetTaskByID(taskID)
	if err != nil || task.MemberID != memberID {
		return ErrTaskNotFound
	}
	return s.repo.DeleteTask(taskID)
}

func (s *Service) SyncTasks(memberID, roomID string, tasks []models.SyncTaskItem) (*models.SyncTasksResponse, error) {
	result := &models.SyncTasksResponse{}
	for _, task := range tasks {
		createdAt, _ := time.Parse(time.RFC3339, task.CreatedAt)
		t := &models.Task{
			ID:        uuid.New().String(),
			ClientID:  task.ClientID,
			MemberID:  memberID,
			RoomID:    roomID,
			ProjectID: task.ProjectID,
			Title:     task.Title,
			Status:    task.Status,
			CreatedAt: createdAt,
			UpdatedAt: time.Now(),
		}
		if err := s.repo.CreateTask(t); err == nil {
			result.Synced++
			result.Tasks = append(result.Tasks, models.SyncTaskResult{
				ClientID: task.ClientID,
				ServerID: t.ID,
			})
		}
	}
	return result, nil
}

// Announcement operations

func (s *Service) CreateAnnouncement(roomName, senderID, title, body string) (*models.Announcement, error) {
	room, err := s.repo.GetRoomByName(roomName)
	if err != nil {
		return nil, ErrRoomNotFound
	}

	announcement := &models.Announcement{
		ID:        uuid.New().String(),
		RoomID:    room.ID,
		SenderID:  senderID,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateAnnouncement(announcement); err != nil {
		return nil, err
	}
	return announcement, nil
}

func (s *Service) GetAnnouncements(roomName string, limit int) ([]*models.Announcement, error) {
	room, err := s.repo.GetRoomByName(roomName)
	if err != nil {
		return nil, ErrRoomNotFound
	}
	return s.repo.GetAnnouncementsByRoomID(room.ID, limit)
}

// Helper functions

func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
