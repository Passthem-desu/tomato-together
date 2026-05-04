package repository

import (
	"database/sql"
	"time"

	"tomatogether/backend/internal/models"
)

type RoomTokenRepository struct {
	db *sql.DB
}

func NewRoomTokenRepository(db *sql.DB) *RoomTokenRepository {
	return &RoomTokenRepository{db: db}
}

func (r *RoomTokenRepository) Create(token *models.RoomToken) error {
	_, err := r.db.Exec(
		"INSERT INTO room_tokens (id, user_id, room_id, token, created_at, expires_at, last_heartbeat) VALUES (?, ?, ?, ?, ?, ?, ?)",
		token.ID, token.UserID, token.RoomID, token.Token, token.CreatedAt, token.ExpiresAt, token.LastHeartbeat,
	)
	return err
}

func (r *RoomTokenRepository) GetByToken(token string) (*models.RoomToken, error) {
	rt := &models.RoomToken{}
	err := r.db.QueryRow(
		"SELECT id, user_id, room_id, token, created_at, expires_at, last_heartbeat FROM room_tokens WHERE token = ?",
		token,
	).Scan(&rt.ID, &rt.UserID, &rt.RoomID, &rt.Token, &rt.CreatedAt, &rt.ExpiresAt, &rt.LastHeartbeat)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rt, err
}

func (r *RoomTokenRepository) GetByUserAndRoom(userID, roomID string) (*models.RoomToken, error) {
	rt := &models.RoomToken{}
	err := r.db.QueryRow(
		"SELECT id, user_id, room_id, token, created_at, expires_at, last_heartbeat FROM room_tokens WHERE user_id = ? AND room_id = ?",
		userID, roomID,
	).Scan(&rt.ID, &rt.UserID, &rt.RoomID, &rt.Token, &rt.CreatedAt, &rt.ExpiresAt, &rt.LastHeartbeat)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rt, err
}

func (r *RoomTokenRepository) GetByUserID(userID string) (*models.RoomToken, error) {
	rt := &models.RoomToken{}
	err := r.db.QueryRow(
		"SELECT id, user_id, room_id, token, created_at, expires_at, last_heartbeat FROM room_tokens WHERE user_id = ?",
		userID,
	).Scan(&rt.ID, &rt.UserID, &rt.RoomID, &rt.Token, &rt.CreatedAt, &rt.ExpiresAt, &rt.LastHeartbeat)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rt, err
}

func (r *RoomTokenRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM room_tokens WHERE id = ?", id)
	return err
}

func (r *RoomTokenRepository) DeleteByToken(token string) error {
	_, err := r.db.Exec("DELETE FROM room_tokens WHERE token = ?", token)
	return err
}

func (r *RoomTokenRepository) DeleteByUserAndRoom(userID, roomID string) error {
	_, err := r.db.Exec(
		"DELETE FROM room_tokens WHERE user_id = ? AND room_id = ?",
		userID, roomID,
	)
	return err
}

func (r *RoomTokenRepository) UpdateHeartbeat(token string) error {
	_, err := r.db.Exec(
		"UPDATE room_tokens SET last_heartbeat = ? WHERE token = ?",
		time.Now(), token,
	)
	return err
}

func (r *RoomTokenRepository) GetExpiredTokens() ([]*models.RoomToken, error) {
	rows, err := r.db.Query(
		"SELECT id, user_id, room_id, token, created_at, expires_at, last_heartbeat FROM room_tokens WHERE expires_at < ?",
		time.Now(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*models.RoomToken
	for rows.Next() {
		rt := &models.RoomToken{}
		if err := rows.Scan(&rt.ID, &rt.UserID, &rt.RoomID, &rt.Token, &rt.CreatedAt, &rt.ExpiresAt, &rt.LastHeartbeat); err != nil {
			return nil, err
		}
		tokens = append(tokens, rt)
	}
	return tokens, rows.Err()
}

func (r *RoomTokenRepository) GetStaleTokens(timeout time.Duration) ([]*models.RoomToken, error) {
	threshold := time.Now().Add(-timeout)
	rows, err := r.db.Query(
		"SELECT id, user_id, room_id, token, created_at, expires_at, last_heartbeat FROM room_tokens WHERE last_heartbeat < ?",
		threshold,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*models.RoomToken
	for rows.Next() {
		rt := &models.RoomToken{}
		if err := rows.Scan(&rt.ID, &rt.UserID, &rt.RoomID, &rt.Token, &rt.CreatedAt, &rt.ExpiresAt, &rt.LastHeartbeat); err != nil {
			return nil, err
		}
		tokens = append(tokens, rt)
	}
	return tokens, rows.Err()
}