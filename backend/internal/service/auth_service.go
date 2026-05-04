package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid_credentials")
	ErrUsernameTaken      = errors.New("username_taken")
	ErrTokenInvalid       = errors.New("token_invalid")
	ErrTokenExpired       = errors.New("token_expired")
)

type AuthService struct {
	userRepo     *repository.UserRepository
	roomTokenRepo *repository.RoomTokenRepository
	jwtSecret    []byte
}

func NewAuthService(userRepo *repository.UserRepository, roomTokenRepo *repository.RoomTokenRepository) *AuthService {
	secret := getEnv("JWT_SECRET", "default-secret-change-me")
	return &AuthService{
		userRepo:     userRepo,
		roomTokenRepo: roomTokenRepo,
		jwtSecret:    []byte(secret),
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func (s *AuthService) GenerateAccessToken(userID string) (string, error) {
	expiry := getEnvInt("JWT_ACCESS_TOKEN_EXPIRY", 3600)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiry) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

func (s *AuthService) ValidateRoomToken(tokenString string) (*models.RoomToken, error) {
	token, err := s.roomTokenRepo.GetByToken(tokenString)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenInvalid
	}
	if token.IsExpired() {
		return nil, ErrTokenExpired
	}
	return token, nil
}

func (s *AuthService) RefreshRoomToken(userID, roomID string) (*models.RoomToken, error) {
	// Delete old token
	s.roomTokenRepo.DeleteByUserAndRoom(userID, roomID)

	// Create new token
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

	err := s.roomTokenRepo.Create(token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (s *AuthService) UpdateHeartbeat(tokenString string) error {
	return s.roomTokenRepo.UpdateHeartbeat(tokenString)
}

func getEnv(key string, defaultValue string) string {
	// Simple implementation - in production use os.Getenv
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	// Simple implementation - in production use os.Getenv
	return defaultValue
}