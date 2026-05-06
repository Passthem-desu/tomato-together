package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
	"tomatogether/backend/internal/sse"
)

// Field length limits
const (
	MaxUsernameLen       = 30
	MaxRoomNameLen       = 50
	MaxPasswordLen       = 128
	MaxTagNameLen        = 30
	MaxTaskTitleLen      = 200
	MaxAnnouncementTitle = 100
	MaxAnnouncementBody  = 1000
	MaxEmojiLen          = 10
	MaxStatusMessageLen  = 200
	MinPasswordLen       = 6
)

var (
	ErrRoomNotFound         = errors.New("room_not_found")
	ErrMemberNotFound       = errors.New("member_not_found")
	ErrInvalidPassword      = errors.New("invalid_password")
	ErrInvalidRoomPassword  = errors.New("invalid_room_password")
	ErrRoomRequiresPassword = errors.New("room_requires_password")
	ErrRoomIsReadonly       = errors.New("room_is_readonly")
	ErrUsernameTaken        = errors.New("username_taken")
	ErrRoomNameTaken        = errors.New("room_name_taken")
	ErrNoActiveSession      = errors.New("no_active_session")
	ErrAlreadyFollowing     = errors.New("already_following")
	ErrNotFollowing         = errors.New("not_following")
	ErrNotRoomOwner         = errors.New("not_room_owner")
	ErrSessionNotActive     = errors.New("session_not_active")
	ErrTokenExpired         = errors.New("token_expired")
	ErrTokenInvalid         = errors.New("token_invalid")
	ErrTokenRevoked         = errors.New("token_revoked")
	ErrMustBePersistent     = errors.New("must_be_persistent_user")
	ErrMustBeOwner          = errors.New("must_be_owner")
	ErrTagNotFound          = errors.New("tag_not_found")
	ErrTaskNotFound         = errors.New("task_not_found")
	ErrPasswordTooShort     = errors.New("password_too_short")
	ErrFieldTooLong         = errors.New("field_too_long")
)

type Service struct {
	repo   *repository.Repository
	jwtCfg *jwtConfig
}

func New(repo *repository.Repository) *Service {
	return &Service{
		repo:   repo,
		jwtCfg: loadJWTConfig(),
	}
}

// Token expiry duration
const tokenExpiry = 24 * time.Hour

// Room operations

func (s *Service) CreateRoom(req *models.CreateRoomRequest) (*models.RoomResponse, error) {
	// Validate input lengths
	if len(req.RoomName) > MaxRoomNameLen {
		return nil, ErrFieldTooLong
	}
	if len(req.Username) > MaxUsernameLen {
		return nil, ErrFieldTooLong
	}
	if req.Password != "" && len(req.Password) < MinPasswordLen {
		return nil, ErrPasswordTooShort
	}
	if len(req.Password) > MaxPasswordLen {
		return nil, ErrFieldTooLong
	}

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

	// Generate JWT if the user is persistent and JWT is configured
	var accessToken, refreshToken string
	var expiresIn int64
	if member.IsPersistent && s.jwtCfg != nil {
		at, _, err := s.GenerateAccessToken(member)
		if err == nil {
			accessToken = at
			expiresIn = int64(s.jwtCfg.AccessExpiry.Seconds())
		}
		rt, _, err := s.GenerateRefreshToken(memberID, "")
		if err == nil {
			refreshToken = rt
		}
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
		Token:        token.Token,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

func (s *Service) JoinRoom(roomName string, req *models.JoinRoomRequest) (*models.RoomResponse, error) {
	// Validate input lengths
	if len(req.Username) > MaxUsernameLen {
		return nil, ErrFieldTooLong
	}
	if req.Password != "" && len(req.Password) < MinPasswordLen {
		return nil, ErrPasswordTooShort
	}
	if len(req.Password) > MaxPasswordLen {
		return nil, ErrFieldTooLong
	}

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
			// Password correct - create new token (multi-device: keep existing tokens)
			tokenValue, err := generateToken()
			if err != nil {
				return nil, err
			}
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
			resp := &models.RoomResponse{
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
			}
			s.addJWTToResponse(resp, existingMember)
			return resp, nil
		} else {
			// Existing anonymous user - allow re-join and inherit data
			if req.Password == "" {
				// Both old and new are anonymous - allow re-use (Sec #17: wrapped in transaction)
				tokenValue, err := generateToken()
				if err != nil {
					return nil, err
				}
				token := &models.RoomToken{
					ID:            uuid.New().String(),
					MemberID:      existingMember.ID,
					RoomID:        room.ID,
					Token:         tokenValue,
					CreatedAt:     time.Now(),
					ExpiresAt:     time.Now().Add(tokenExpiry),
					LastHeartbeat: time.Now(),
				}
				err = s.repo.RunInTx(func(tx *sql.Tx) error {
					if err := s.repo.DeleteTokenByMemberAndRoomInTx(tx, existingMember.ID, room.ID); err != nil {
						return err
					}
					return s.repo.CreateTokenInTx(tx, token)
				})
				if err != nil {
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
				// New request has password, old is anonymous - upgrade to persistent (Sec #17: wrapped in transaction)
				hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
				if err != nil {
					return nil, err
				}
				tokenValue, err := generateToken()
				if err != nil {
					return nil, err
				}
				token := &models.RoomToken{
					ID:            uuid.New().String(),
					MemberID:      existingMember.ID,
					RoomID:        room.ID,
					Token:         tokenValue,
					CreatedAt:     time.Now(),
					ExpiresAt:     time.Now().Add(tokenExpiry),
					LastHeartbeat: time.Now(),
				}
				err = s.repo.RunInTx(func(tx *sql.Tx) error {
					if err := s.repo.UpdateMemberPasswordInTx(tx, existingMember.ID, string(hash)); err != nil {
						return err
					}
					if err := s.repo.DeleteTokenByMemberAndRoomInTx(tx, existingMember.ID, room.ID); err != nil {
						return err
					}
					return s.repo.CreateTokenInTx(tx, token)
				})
				if err != nil {
					return nil, err
				}
				resp := &models.RoomResponse{
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
						IsPersistent: true,
					},
					Token: token.Token,
				}
				s.addJWTToResponse(resp, existingMember)
				return resp, nil
			}
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

	resp := &models.RoomResponse{
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
	}
	s.addJWTToResponse(resp, member)
	return resp, nil
}

func (s *Service) LeaveRoom(tokenValue string) error {
	tokenInfo, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	// End any active pomodoro session
	s.repo.EndActiveSessionByMemberID(tokenInfo.MemberID)

	rt, err := s.repo.GetTokenByValue(tokenValue)
	if err == nil {
		s.repo.DeleteToken(rt.ID)
	}

	return nil
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

	// Get online users from SSE Hub
	room, err := s.repo.GetRoomByID(token.RoomID)
	if err != nil {
		return nil, err
	}

	// Get per-member stats
	statsMap, _ := s.repo.GetRoomMembersStats(token.RoomID)

	onlineUsers := sse.GetHub().GetOnlineUsers(room.Name)
	onlineMap := make(map[string]bool)
	for _, u := range onlineUsers {
		onlineMap[u.ID] = true
	}

	var users []*models.UserInfo
	for _, member := range members {
		user := &models.UserInfo{
			ID:           member.ID,
			Username:     member.Username,
			IsOwner:      member.IsOwner,
			IsPersistent: member.IsPersistent,
			IsOnline:     onlineMap[member.ID],
		}

		// Attach stats
		if s, ok := statsMap[member.ID]; ok {
			user.TotalPomodoros = s[0]
			user.TotalDuration = s[1]
		}

		// Get status
		if status, ok := statusMap[member.ID]; ok {
			user.Status = &models.StatusInfo{
				Emoji:   status.Emoji,
				Message: status.Message,
			}
		}

		// Get phase from active session or rest state
		session, err := s.repo.GetActiveSessionByMemberID(member.ID)
		if err == nil && session != nil {
			elapsed := int(time.Since(session.StartedAt).Seconds())
			remaining := session.PlannedDuration - elapsed
			if remaining < 0 {
				remaining = 0
			}
			phase := "focusing"
			if session.PausedAt != nil {
				phase = "paused"
			}
			user.Pomodoro = &models.PomodoroInfo{
				Phase:            phase,
				RemainingSeconds: remaining,
			}
			if session.IsFollowed && session.LeaderID != "" {
				leader, _ := s.repo.GetMemberByID(session.LeaderID)
				if leader != nil {
					user.Pomodoro.IsFollowing = true
					user.Pomodoro.LeaderUsername = leader.Username
				}
			}
		} else {
			// Check if in rest phase
			latest, err := s.repo.GetLatestSessionByMemberID(member.ID)
			if err == nil && latest != nil && latest.EndedAt != nil && latest.RestDuration > 0 {
				restEnd := latest.EndedAt.Add(time.Duration(latest.RestDuration) * time.Second)
				if time.Now().Before(restEnd) {
					remaining := int(time.Until(restEnd).Seconds())
					if remaining < 0 {
						remaining = 0
					}
					user.Pomodoro = &models.PomodoroInfo{
						Phase:            "rest",
						RemainingSeconds: remaining,
					}
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

func (s *Service) CheckRoomPassword(roomName, password string) (bool, error) {
	room, err := s.repo.GetRoomByName(roomName)
	if err != nil {
		return false, ErrRoomNotFound
	}
	if room.PasswordHash == "" {
		return true, nil // No password set
	}
	return checkPassword(password, room.PasswordHash), nil
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
	if len(req.Password) < MinPasswordLen {
		return ErrPasswordTooShort
	}
	if len(req.Password) > MaxPasswordLen {
		return ErrFieldTooLong
	}

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

func (s *Service) ChangePassword(tokenValue string, req *models.ChangePasswordRequest) error {
	if len(req.NewPassword) < MinPasswordLen {
		return ErrPasswordTooShort
	}
	if len(req.NewPassword) > MaxPasswordLen {
		return ErrFieldTooLong
	}

	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	member, err := s.repo.GetMemberByID(token.MemberID)
	if err != nil {
		return err
	}

	// Must be a persistent user (have existing password)
	if member.PasswordHash == "" {
		return ErrMustBePersistent
	}

	// Verify old password
	if !checkPassword(req.OldPassword, member.PasswordHash) {
		return ErrInvalidPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repo.UpdateMemberPassword(token.MemberID, string(hash))
}

func (s *Service) Login(req *models.LoginRequest) (*models.RoomResponse, error) {
	// Validate input
	if len(req.Username) > MaxUsernameLen {
		return nil, ErrFieldTooLong
	}

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

	resp := &models.RoomResponse{
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
	}
	s.addJWTToResponse(resp, member)
	return resp, nil
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
		ID:                       uuid.New().String(),
		MemberID:                 token.MemberID,
		RoomID:                   token.RoomID,
		TagID:                    req.TagID,
		TaskID:                   req.TaskID,
		PlannedDuration:          plannedDuration,
		PlannedRestDuration:      restDuration,
		PlannedLongBreakDuration: longBreakDuration,
		SessionsBeforeLongBreak:  sessionsBeforeLongBreak,
		SessionIndex:             req.SessionIndex,
		IsFollowed:               false,
		StartedAt:                time.Now(),
	}
	if err := s.repo.CreatePomodoroSession(session); err != nil {
		return nil, err
	}

	go s.broadcastPomodoroStarted(token.RoomID, token.MemberID, session.ID, session.StartedAt)

	return &models.PomodoroStatusResponse{
		Phase:                   "focusing",
		SessionID:               session.ID,
		StartedAt:               session.StartedAt.Format(time.RFC3339),
		RemainingSeconds:        plannedDuration,
		PlannedDuration:         plannedDuration,
		RestDuration:            restDuration,
		LongBreakDuration:       longBreakDuration,
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

	// Broadcast SSE event
	go s.broadcastPomodoroFollowed(token.RoomID, token.MemberID, req.LeaderID, leader.Username)

	return &models.PomodoroStatusResponse{
		Phase:            "following",
		SessionID:        session.ID,
		StartedAt:        leaderSession.StartedAt.Format(time.RFC3339),
		RemainingSeconds: remaining,
		LeaderID:         req.LeaderID,
		LeaderUsername:   leader.Username,
		PlannedDuration:  leaderSession.PlannedDuration,
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
	if duration < 60 {
		duration = 0 // Don't count sessions stopped within 1 minute
	}
	if err := s.repo.UpdatePomodoroSession(session.ID, &now, duration); err != nil {
		return "", err
	}

	// Broadcast SSE event
	go s.broadcastPomodoroUnfollowed(token.RoomID, token.MemberID)

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

	// Use session's stored SessionIndex (set at creation time, immutable)
	sessionsCompleted := session.SessionIndex
	if sessionsCompleted <= 0 {
		// Fallback for old sessions without session_index
		sessions, _ := s.repo.GetTodaySessionsByMemberID(token.MemberID, time.Now())
		sessionsCompleted = len(sessions) + 1
	}

	// Determine rest duration from session's stored settings
	restDuration := session.PlannedRestDuration
	if restDuration == 0 {
		restDuration = 300
	}
	longBreakDuration := session.PlannedLongBreakDuration
	if longBreakDuration == 0 {
		longBreakDuration = 900
	}
	sessionsBefore := session.SessionsBeforeLongBreak
	if sessionsBefore == 0 {
		sessionsBefore = 4
	}
	shouldTakeLongBreak := sessionsCompleted%sessionsBefore == 0 && sessionsCompleted > 0

	// End the session
	now := time.Now()
	duration := int(now.Sub(session.StartedAt).Seconds())

	if req.Aborted {
		// Don't count sessions stopped within 1 minute (accidental start)
		recordedDuration := duration
		if duration < 60 {
			recordedDuration = 0
		}
		if err := s.repo.UpdatePomodoroSession(session.ID, &now, recordedDuration); err != nil {
			return nil, err
		}
		go s.broadcastPomodoroEnded(token.RoomID, token.MemberID, session.ID, recordedDuration, "aborted")
		return &models.PomodoroStatusResponse{
			Phase:             "idle",
			SessionID:         session.ID,
			Duration:          recordedDuration,
			PlannedDuration:   session.PlannedDuration,
			SessionsCompleted: sessionsCompleted,
		}, nil
	}

	// Normal end: enter rest phase
	actualRest := restDuration
	if shouldTakeLongBreak {
		actualRest = longBreakDuration
	}
	if err := s.repo.EndSessionWithRest(session.ID, &now, duration, actualRest, shouldTakeLongBreak); err != nil {
		return nil, err
	}

	go s.broadcastPomodoroEnded(token.RoomID, token.MemberID, session.ID, duration, "rest")

	return &models.PomodoroStatusResponse{
		Phase:               "rest",
		SessionID:           session.ID,
		RemainingSeconds:    actualRest,
		Duration:            duration,
		PlannedDuration:     session.PlannedDuration,
		RestDuration:        actualRest,
		LongBreakDuration:   longBreakDuration,
		IsLongBreak:         shouldTakeLongBreak,
		ShouldTakeLongBreak: shouldTakeLongBreak,
		SessionsCompleted:   sessionsCompleted,
	}, nil
}

func (s *Service) GetPomodoroStatus(tokenValue string) (*models.PomodoroStatusResponse, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	// 1. Check active session (focusing or paused)
	session, err := s.repo.GetActiveSessionByMemberID(token.MemberID)
	if err == nil && session != nil {
		elapsed := int(time.Since(session.StartedAt).Seconds())
		remaining := session.PlannedDuration - elapsed
		if remaining < 0 {
			remaining = 0
		}

		phase := "focusing"
		var pausedAt string
		if session.PausedAt != nil {
			phase = "paused"
			pausedAt = session.PausedAt.Format(time.RFC3339)
			// When paused, remaining is frozen at pause time
			pausedElapsed := int(session.PausedAt.Sub(session.StartedAt).Seconds())
			remaining = session.PlannedDuration - pausedElapsed
			if remaining < 0 {
				remaining = 0
			}
		}

		response := &models.PomodoroStatusResponse{
			Phase:            phase,
			SessionID:        session.ID,
			StartedAt:        session.StartedAt.Format(time.RFC3339),
			PausedAt:         pausedAt,
			RemainingSeconds: remaining,
			PlannedDuration:  session.PlannedDuration,
		}

		if session.IsFollowed && session.LeaderID != "" {
			leader, _ := s.repo.GetMemberByID(session.LeaderID)
			if leader != nil {
				response.LeaderID = session.LeaderID
				response.LeaderUsername = leader.Username
				response.Phase = "following"
			}
		}

		return response, nil
	}

	// 2. Check rest phase (most recent session with rest_duration > 0 and not expired)
	latest, err := s.repo.GetLatestSessionByMemberID(token.MemberID)
	if err == nil && latest != nil && latest.EndedAt != nil && latest.RestDuration > 0 {
		restEnd := latest.EndedAt.Add(time.Duration(latest.RestDuration) * time.Second)
		if time.Now().Before(restEnd) {
			remaining := int(time.Until(restEnd).Seconds())
			if remaining < 0 {
				remaining = 0
			}
			return &models.PomodoroStatusResponse{
				Phase:            "rest",
				RemainingSeconds: remaining,
				RestDuration:     latest.RestDuration,
				IsLongBreak:      latest.IsLongBreak,
			}, nil
		}
	}

	// 3. Idle
	return &models.PomodoroStatusResponse{Phase: "idle"}, nil
}

// Pause / Resume / SkipRest

func (s *Service) PausePomodoro(tokenValue string) (*models.PomodoroStatusResponse, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	session, err := s.repo.GetActiveSessionByMemberID(token.MemberID)
	if err != nil || session == nil {
		return nil, ErrNoActiveSession
	}
	if session.PausedAt != nil {
		return nil, ErrAlreadyFollowing // Already paused
	}

	if err := s.repo.PauseSession(session.ID); err != nil {
		return nil, err
	}

	go s.broadcastPhaseChanged(token.RoomID, token.MemberID, "paused")
	return s.GetPomodoroStatus(tokenValue)
}

func (s *Service) ResumePomodoro(tokenValue string) (*models.PomodoroStatusResponse, error) {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	session, err := s.repo.GetActiveSessionByMemberID(token.MemberID)
	if err != nil || session == nil {
		return nil, ErrNoActiveSession
	}
	if session.PausedAt == nil {
		return nil, ErrSessionNotActive // Not paused
	}

	pausedMillis := int(time.Since(*session.PausedAt).Milliseconds())
	if err := s.repo.ResumeSession(session.ID, pausedMillis); err != nil {
		return nil, err
	}

	go s.broadcastPhaseChanged(token.RoomID, token.MemberID, "focusing")
	return s.GetPomodoroStatus(tokenValue)
}

func (s *Service) SkipRest(tokenValue string) error {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	latest, err := s.repo.GetLatestSessionByMemberID(token.MemberID)
	if err != nil || latest == nil {
		return ErrNoActiveSession
	}

	now := time.Now()
	var duration int

	if latest.EndedAt == nil {
		active, err := s.repo.GetActiveSessionByMemberID(token.MemberID)
		if err == nil && active != nil && active.ID == latest.ID {
			if active.PausedAt != nil {
				duration = int(active.PausedAt.Sub(active.StartedAt).Seconds())
			} else {
				duration = int(now.Sub(active.StartedAt).Seconds())
			}
			if duration < 60 {
				duration = 0
			}
		} else {
			duration = latest.Duration
		}
	} else {
		duration = latest.Duration
	}

	err = s.repo.EndSessionWithRest(latest.ID, &now, duration, 0, false)
	if err != nil {
		return err
	}
	go s.broadcastPhaseChanged(token.RoomID, token.MemberID, "idle")
	return nil
}

// Status operations

func (s *Service) UpdateStatus(tokenValue string, req *models.UpdateStatusRequest) (*models.UserStatus, error) {
	if len(req.Emoji) > MaxEmojiLen {
		return nil, ErrFieldTooLong
	}
	if len(req.Message) > MaxStatusMessageLen {
		return nil, ErrFieldTooLong
	}

	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	// Empty emoji and empty message = clear status
	if req.Emoji == "" && req.Message == "" {
		if err := s.repo.DeleteUserStatus(token.MemberID, token.RoomID); err != nil {
			return nil, err
		}
		go s.broadcastStatusUpdated(token.RoomID, token.MemberID, "", "")
		return nil, nil
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

	// Broadcast SSE event
	go s.broadcastStatusUpdated(token.RoomID, token.MemberID, req.Emoji, req.Message)

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

func (s *Service) ValidateToken(tokenValue string) (*models.TokenInfo, error) {
	// 1. Try JWT parsing first (stateless, no DB query)
	if tokenInfo, err := s.ValidateJWT(tokenValue); err == nil {
		tokenInfo.Token = tokenValue
		return tokenInfo, nil
	}

	// 2. JWT failed — fall back to RoomToken DB lookup
	token, err := s.repo.GetTokenByValue(tokenValue)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Check if token has been revoked
	revoked, err := s.repo.IsTokenRevoked(tokenValue)
	if err == nil && revoked {
		return nil, ErrTokenRevoked
	}

	if time.Now().After(token.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Look up member to get username/is_owner/is_persistent
	member, err := s.repo.GetMemberByID(token.MemberID)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	return &models.TokenInfo{
		MemberID:     token.MemberID,
		RoomID:       token.RoomID,
		Username:     member.Username,
		IsOwner:      member.IsOwner,
		IsPersistent: member.IsPersistent,
		IsJWT:        false,
		Token:        tokenValue,
	}, nil
}

func (s *Service) RefreshToken(tokenValue string) error {
	tokenInfo, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}
	// Refresh heartbeat via room_tokens table
	// For JWT tokens, heartbeat is not needed (stateless), but we maintain backward compat
	if !tokenInfo.IsJWT {
		token, err := s.repo.GetTokenByValue(tokenValue)
		if err != nil {
			return err
		}
		return s.repo.UpdateTokenHeartbeat(token.ID)
	}
	return nil
}

// Tag operations

func (s *Service) GetTags(memberID, roomID string) ([]*models.Tag, error) {
	return s.repo.GetTagsByMemberAndRoom(memberID, roomID)
}

func (s *Service) CreateTag(memberID, roomID, name string) (*models.Tag, error) {
	if len(name) > MaxTagNameLen {
		return nil, ErrFieldTooLong
	}
	tag := &models.Tag{
		ID:        uuid.New().String(),
		MemberID:  memberID,
		RoomID:    roomID,
		Name:      name,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateTag(tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *Service) UpdateTag(tagID, memberID, name string) error {
	tag, err := s.repo.GetTagByID(tagID)
	if err != nil || tag.MemberID != memberID {
		return ErrTagNotFound
	}
	return s.repo.UpdateTag(tagID, name)
}

func (s *Service) DeleteTag(tagID, memberID string) error {
	tag, err := s.repo.GetTagByID(tagID)
	if err != nil || tag.MemberID != memberID {
		return ErrTagNotFound
	}
	return s.repo.DeleteTag(tagID)
}

// Task operations

func (s *Service) GetTasks(memberID, roomID, status string) ([]*models.Task, error) {
	return s.repo.GetTasksByMemberAndRoom(memberID, roomID, status)
}

func (s *Service) CreateTask(memberID, roomID, clientID, title, tagID string) (*models.Task, error) {
	if len(title) > MaxTaskTitleLen {
		return nil, ErrFieldTooLong
	}
	task := &models.Task{
		ID:        uuid.New().String(),
		ClientID:  clientID,
		MemberID:  memberID,
		RoomID:    roomID,
		TagID:     tagID,
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

func (s *Service) UpdateTask(taskID, memberID, title, status, tagID string) error {
	task, err := s.repo.GetTaskByID(taskID)
	if err != nil || task.MemberID != memberID {
		return ErrTaskNotFound
	}
	var completedAt *time.Time
	if status == "DONE" {
		now := time.Now()
		completedAt = &now
	}
	return s.repo.UpdateTask(taskID, title, status, tagID, completedAt)
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
			TagID:     task.TagID,
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
	// Validate field lengths
	if len(title) > MaxAnnouncementTitle {
		return nil, ErrFieldTooLong
	}
	if len(body) > MaxAnnouncementBody {
		return nil, ErrFieldTooLong
	}

	room, err := s.repo.GetRoomByName(roomName)
	if err != nil {
		return nil, ErrRoomNotFound
	}

	// Verify sender is the room owner (Sec #1 fix)
	member, err := s.repo.GetMemberByID(senderID)
	if err != nil {
		return nil, ErrMemberNotFound
	}
	if !member.IsOwner {
		return nil, ErrMustBeOwner
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

func (s *Service) DeleteAnnouncement(announcementID, memberID string) error {
	a, err := s.repo.GetAnnouncementByID(announcementID)
	if err != nil {
		return errors.New("announcement_not_found")
	}
	// Only the sender or room owner can delete
	if a.SenderID != memberID {
		// Check if member is room owner
		member, err := s.repo.GetMemberByID(memberID)
		if err != nil || !member.IsOwner {
			return ErrNotRoomOwner
		}
	}
	return s.repo.DeleteAnnouncement(announcementID)
}

func (s *Service) addJWTToResponse(resp *models.RoomResponse, member *models.RoomMember) {
	if member == nil || !member.IsPersistent || s.jwtCfg == nil {
		return
	}
	at, _, err := s.GenerateAccessToken(member)
	if err == nil {
		resp.AccessToken = at
		resp.ExpiresIn = int64(s.jwtCfg.AccessExpiry.Seconds())
	}
	rt, _, err := s.GenerateRefreshToken(member.ID, "")
	if err == nil {
		resp.RefreshToken = rt
	}
}

// Helper functions

func (s *Service) GetMemberStats(memberID string) (int, int, error) {
	return s.repo.GetMemberStats(memberID)
}

func (s *Service) ResetMemberStats(memberID string) error {
	return s.repo.DeleteMemberPomodoroSessions(memberID)
}

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

func (s *Service) broadcastPhaseChanged(roomID, memberID, phase string) {
	hub := sse.GetHub()
	if hub == nil {
		return
	}
	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return
	}
	member, err := s.repo.GetMemberByID(memberID)
	if err != nil {
		return
	}
	hub.BroadcastEvent(room.Name, "phase_changed", map[string]interface{}{
		"user_id":  memberID,
		"username": member.Username,
		"phase":    phase,
	})
}

// SSE broadcast helper functions

func (s *Service) broadcastPomodoroStarted(roomID, memberID, sessionID string, startedAt time.Time) {
	hub := sse.GetHub()
	if hub == nil {
		return
	}

	// Get room name and member info
	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return
	}

	member, err := s.repo.GetMemberByID(memberID)
	if err != nil {
		return
	}

	hub.BroadcastEvent(room.Name, "pomodoro_started", map[string]interface{}{
		"user_id":    memberID,
		"username":   member.Username,
		"session_id": sessionID,
		"started_at": startedAt.Format(time.RFC3339),
	})

	log.Printf("Broadcast pomodoro_started: room=%s, user=%s", room.Name, member.Username)
}

func (s *Service) broadcastPomodoroEnded(roomID, memberID, sessionID string, duration int, status string) {
	hub := sse.GetHub()
	if hub == nil {
		return
	}

	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return
	}

	member, err := s.repo.GetMemberByID(memberID)
	if err != nil {
		return
	}

	hub.BroadcastEvent(room.Name, "pomodoro_ended", map[string]interface{}{
		"user_id":    memberID,
		"username":   member.Username,
		"session_id": sessionID,
		"duration":   duration,
		"status":     status,
	})

	log.Printf("Broadcast pomodoro_ended: room=%s, user=%s", room.Name, member.Username)
}

func (s *Service) broadcastPomodoroFollowed(roomID, memberID, leaderID, leaderUsername string) {
	hub := sse.GetHub()
	if hub == nil {
		return
	}

	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return
	}

	member, err := s.repo.GetMemberByID(memberID)
	if err != nil {
		return
	}

	hub.BroadcastEvent(room.Name, "pomodoro_followed", map[string]interface{}{
		"user_id":         memberID,
		"username":        member.Username,
		"leader_id":       leaderID,
		"leader_username": leaderUsername,
	})

	log.Printf("Broadcast pomodoro_followed: room=%s, user=%s follows %s", room.Name, member.Username, leaderUsername)
}

func (s *Service) broadcastPomodoroUnfollowed(roomID, memberID string) {
	hub := sse.GetHub()
	if hub == nil {
		return
	}

	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return
	}

	member, err := s.repo.GetMemberByID(memberID)
	if err != nil {
		return
	}

	hub.BroadcastEvent(room.Name, "pomodoro_unfollowed", map[string]interface{}{
		"user_id":  memberID,
		"username": member.Username,
	})

	log.Printf("Broadcast pomodoro_unfollowed: room=%s, user=%s", room.Name, member.Username)
}

func (s *Service) broadcastStatusUpdated(roomID, memberID, emoji, message string) {
	hub := sse.GetHub()
	if hub == nil {
		return
	}

	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return
	}

	member, err := s.repo.GetMemberByID(memberID)
	if err != nil {
		return
	}

	hub.BroadcastEvent(room.Name, "status_updated", map[string]interface{}{
		"user_id":  memberID,
		"username": member.Username,
		"emoji":    emoji,
		"message":  message,
	})

	log.Printf("Broadcast status_updated: room=%s, user=%s", room.Name, member.Username)
}

// KickMember kicks a member from a room (owner only) — Sec #3 fix
func (s *Service) KickMember(tokenValue, targetMemberID string) error {
	token, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	// Verify requester is owner
	member, err := s.repo.GetMemberByID(token.MemberID)
	if err != nil {
		return err
	}
	if !member.IsOwner {
		return ErrMustBeOwner
	}

	// Verify target is in the same room
	target, err := s.repo.GetMemberByID(targetMemberID)
	if err != nil {
		return ErrMemberNotFound
	}
	if target.RoomID != token.RoomID {
		return ErrMemberNotFound
	}

	// Cannot kick yourself
	if token.MemberID == targetMemberID {
		return errors.New("cannot_kick_self")
	}

	// Broadcast kicked event to notify the target before revoking tokens
	go s.broadcastUserKicked(target.RoomID, targetMemberID, target.Username)

	// End target's active pomodoro session
	s.repo.EndActiveSessionByMemberID(targetMemberID)

	// Revoke all tokens for target member (find and revoke each)
	targetToken, err := s.repo.GetTokenByMemberAndRoom(targetMemberID, target.RoomID)
	if err == nil && targetToken != nil {
		s.repo.RevokeToken(targetToken.Token)
	}

	// Delete all tokens for the target member
	s.repo.DeleteTokensByMemberID(targetMemberID)

	return nil
}

// CleanupRevocations removes expired token revocation records
func (s *Service) CleanupRevocations() {
	s.repo.CleanupExpiredRevocations()
}

func (s *Service) broadcastUserKicked(roomID, memberID, username string) {
	hub := sse.GetHub()
	if hub == nil {
		return
	}
	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return
	}
	hub.BroadcastEvent(room.Name, "kicked", map[string]interface{}{
		"user_id":  memberID,
		"username": username,
	})
	log.Printf("Broadcast kicked: room=%s, user=%s", room.Name, username)
}

func (s *Service) broadcastUserJoined(roomID, memberID string) {
	hub := sse.GetHub()
	if hub == nil {
		return
	}

	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return
	}

	member, err := s.repo.GetMemberByID(memberID)
	if err != nil {
		return
	}

	hub.BroadcastEvent(room.Name, "user_joined", map[string]interface{}{
		"user": map[string]interface{}{
			"id":        member.ID,
			"username":  member.Username,
			"is_online": true,
		},
	})

	log.Printf("Broadcast user_joined: room=%s, user=%s", room.Name, member.Username)
}
