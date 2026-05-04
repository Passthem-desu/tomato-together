package service

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

type UserService struct {
	userRepo     *repository.UserRepository
	roomTokenRepo *repository.RoomTokenRepository
}

func NewUserService(userRepo *repository.UserRepository, roomTokenRepo *repository.RoomTokenRepository) *UserService {
	return &UserService{
		userRepo:     userRepo,
		roomTokenRepo: roomTokenRepo,
	}
}

func (s *UserService) CreateUser(username string, passwordHash string) (*models.User, error) {
	user := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) GetUserByID(id string) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	return s.userRepo.GetByUsername(username)
}

func (s *UserService) Register(username, password string) (*models.User, error) {
	exists, err := s.userRepo.ExistsByUsername(username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.CreateUser(username, string(hashedPassword))
}

func (s *UserService) Login(username, password string) (*models.User, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if user.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *UserService) UpgradeUser(userID, password string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.UpdatePassword(userID, string(hashedPassword)); err != nil {
		return nil, err
	}

	return s.userRepo.GetByID(userID)
}

func (s *UserService) UsernameExists(username string) (bool, error) {
	return s.userRepo.ExistsByUsername(username)
}