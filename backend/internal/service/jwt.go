package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"tomatogether/backend/internal/models"
)

var (
	ErrInvalidRefreshToken  = errors.New("invalid_refresh_token")
	ErrRefreshTokenRevoked  = errors.New("refresh_token_revoked")
	ErrRefreshTokenExpired  = errors.New("refresh_token_expired")
	ErrJWTSecretNotSet      = errors.New("jwt_secret_not_set")
)

// jwtConfig holds JWT configuration read from environment variables.
type jwtConfig struct {
	Secret          []byte
	AccessExpiry    time.Duration
	RefreshExpiry   time.Duration
}

func loadJWTConfig() *jwtConfig {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil
	}

	accessExpiry := 3600 // 1 hour default
	if v := os.Getenv("JWT_ACCESS_TOKEN_EXPIRY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			accessExpiry = n
		}
	}

	refreshExpiry := 2592000 // 30 days default
	if v := os.Getenv("JWT_REFRESH_TOKEN_EXPIRY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			refreshExpiry = n
		}
	}

	return &jwtConfig{
		Secret:        []byte(secret),
		AccessExpiry:  time.Duration(accessExpiry) * time.Second,
		RefreshExpiry: time.Duration(refreshExpiry) * time.Second,
	}
}

// JWTEnabled returns true if JWT_SECRET is configured.
func (s *Service) JWTEnabled() bool {
	return s.jwtCfg != nil
}

// GenerateAccessToken creates a signed JWT access token for a member.
func (s *Service) GenerateAccessToken(member *models.RoomMember) (string, time.Time, error) {
	if s.jwtCfg == nil {
		return "", time.Time{}, ErrJWTSecretNotSet
	}

	now := time.Now()
	expiresAt := now.Add(s.jwtCfg.AccessExpiry)

	claims := &models.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   member.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		RoomID:       member.RoomID,
		Username:     member.Username,
		IsOwner:      member.IsOwner,
		IsPersistent: member.IsPersistent,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtCfg.Secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// GenerateRefreshToken creates a random refresh token and stores its hash in DB.
// Returns (rawToken, refreshToken model, error).
func (s *Service) GenerateRefreshToken(memberID, deviceName string) (string, *models.RefreshToken, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", nil, err
	}
	rawToken := hex.EncodeToString(rawBytes)

	tokenHash := hashRefreshToken(rawToken)

	rt := &models.RefreshToken{
		ID:         uuid.New().String(),
		MemberID:   memberID,
		TokenHash:  tokenHash,
		ExpiresAt:  time.Now().Add(s.getRefreshExpiry()),
		DeviceName: deviceName,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateRefreshToken(rt); err != nil {
		return "", nil, err
	}

	return rawToken, rt, nil
}

// ValidateJWT parses and validates a JWT access token.
// Returns TokenInfo without DB query (stateless).
func (s *Service) ValidateJWT(tokenString string) (*models.TokenInfo, error) {
	if s.jwtCfg == nil {
		return nil, ErrTokenInvalid
	}

	token, err := jwt.ParseWithClaims(tokenString, &models.JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtCfg.Secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*models.JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return &models.TokenInfo{
		MemberID:     claims.Subject,
		RoomID:       claims.RoomID,
		Username:     claims.Username,
		IsOwner:      claims.IsOwner,
		IsPersistent: claims.IsPersistent,
		IsJWT:        true,
	}, nil
}

// RefreshAccessToken validates a refresh token and returns a new access token pair.
// Uses refresh token rotation: the old refresh token is revoked and a new one is issued.
func (s *Service) RefreshAccessToken(refreshTokenValue string) (string, string, int64, error) {
	if s.jwtCfg == nil {
		return "", "", 0, ErrJWTSecretNotSet
	}

	tokenHash := hashRefreshToken(refreshTokenValue)
	rt, err := s.repo.GetRefreshTokenByHash(tokenHash)
	if err != nil {
		return "", "", 0, ErrInvalidRefreshToken
	}

	if rt.RevokedAt != nil {
		return "", "", 0, ErrRefreshTokenRevoked
	}

	if time.Now().After(rt.ExpiresAt) {
		return "", "", 0, ErrRefreshTokenExpired
	}

	// Get member to generate new access token
	member, err := s.repo.GetMemberByID(rt.MemberID)
	if err != nil {
		return "", "", 0, err
	}

	// Generate new access token
	accessToken, _, err := s.GenerateAccessToken(member)
	if err != nil {
		return "", "", 0, err
	}

	// Revoke old refresh token (rotation)
	if err := s.repo.RevokeRefreshToken(rt.ID); err != nil {
		return "", "", 0, err
	}

	// Issue new refresh token (preserve device name)
	newRawToken, _, err := s.GenerateRefreshToken(rt.MemberID, rt.DeviceName)
	if err != nil {
		return "", "", 0, err
	}

	expiresIn := int64(s.jwtCfg.AccessExpiry.Seconds())
	return accessToken, newRawToken, expiresIn, nil
}

// RevokeRefreshTokenByValue revokes a single refresh token by its raw value.
func (s *Service) RevokeRefreshTokenByValue(tokenValue string) error {
	tokenHash := hashRefreshToken(tokenValue)
	rt, err := s.repo.GetRefreshTokenByHash(tokenHash)
	if err != nil {
		return ErrInvalidRefreshToken
	}
	return s.repo.RevokeRefreshToken(rt.ID)
}

// RevokeAllRefreshTokens revokes all refresh tokens for a member.
func (s *Service) RevokeAllRefreshTokens(memberID string) (int64, error) {
	return s.repo.RevokeAllRefreshTokens(memberID)
}

// getRefreshExpiry returns the configured refresh token expiry duration.
func (s *Service) getRefreshExpiry() time.Duration {
	if s.jwtCfg != nil {
		return s.jwtCfg.RefreshExpiry
	}
	return 30 * 24 * time.Hour // default 30 days
}

// hashRefreshToken computes SHA-256 hash of a refresh token value.
func hashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
