package repository

import (
	"database/sql"

	"tomatogether/backend/internal/models"
)

type UserStatusRepository struct {
	db *sql.DB
}

func NewUserStatusRepository(db *sql.DB) *UserStatusRepository {
	return &UserStatusRepository{db: db}
}

func (r *UserStatusRepository) Upsert(status *models.UserStatus) error {
	_, err := r.db.Exec(`
		INSERT INTO user_statuses (id, user_id, room_id, emoji, message, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, room_id) DO UPDATE SET
			emoji = excluded.emoji,
			message = excluded.message,
			updated_at = excluded.updated_at
	`, status.ID, status.UserID, status.RoomID, status.Emoji, status.Message, status.UpdatedAt)
	return err
}

func (r *UserStatusRepository) GetByUserAndRoom(userID, roomID string) (*models.UserStatus, error) {
	status := &models.UserStatus{}
	err := r.db.QueryRow(
		"SELECT id, user_id, room_id, emoji, message, updated_at FROM user_statuses WHERE user_id = ? AND room_id = ?",
		userID, roomID,
	).Scan(&status.ID, &status.UserID, &status.RoomID, &status.Emoji, &status.Message, &status.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return status, err
}

func (r *UserStatusRepository) Delete(userID, roomID string) error {
	_, err := r.db.Exec(
		"DELETE FROM user_statuses WHERE user_id = ? AND room_id = ?",
		userID, roomID,
	)
	return err
}

func (r *UserStatusRepository) GetByRoomID(roomID string) ([]*models.UserStatus, error) {
	rows, err := r.db.Query(
		"SELECT id, user_id, room_id, emoji, message, updated_at FROM user_statuses WHERE room_id = ?",
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []*models.UserStatus
	for rows.Next() {
		status := &models.UserStatus{}
		if err := rows.Scan(&status.ID, &status.UserID, &status.RoomID, &status.Emoji, &status.Message, &status.UpdatedAt); err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}
	return statuses, rows.Err()
}