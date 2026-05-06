package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenInfo — unified token context (replaces *RoomToken in context).
type TokenInfo struct {
	MemberID     string
	RoomID       string
	Username     string
	IsOwner      bool
	IsPersistent bool
	IsJWT        bool
	Token        string // raw token value for pass-through to service methods
}

// JWTClaims for access token.
type JWTClaims struct {
	jwt.RegisteredClaims
	RoomID       string `json:"room_id"`
	Username     string `json:"username"`
	IsOwner      bool   `json:"is_owner"`
	IsPersistent bool   `json:"is_persistent"`
}

// RefreshToken model.
type RefreshToken struct {
	ID         string     `json:"id"`
	MemberID   string     `json:"member_id"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expires_at"`
	DeviceName string     `json:"device_name"`
	CreatedAt  time.Time  `json:"created_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
}
