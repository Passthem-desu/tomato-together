package repository

import (
	"database/sql"

	"tomatogether/backend/internal/models"
)

type RoomMemberRepository struct {
	db *sql.DB
}

func NewRoomMemberRepository(db *sql.DB) *RoomMemberRepository {
	return &RoomMemberRepository{db: db}
}

func (r *RoomMemberRepository) Create(member *models.RoomMember) error {
	_, err := r.db.Exec(
		"INSERT INTO room_members (id, room_id, user_id, is_owner, joined_at) VALUES (?, ?, ?, ?, ?)",
		member.ID, member.RoomID, member.UserID, member.IsOwner, member.JoinedAt,
	)
	return err
}

func (r *RoomMemberRepository) GetByRoomAndUser(roomID, userID string) (*models.RoomMember, error) {
	member := &models.RoomMember{}
	err := r.db.QueryRow(
		"SELECT id, room_id, user_id, is_owner, joined_at FROM room_members WHERE room_id = ? AND user_id = ?",
		roomID, userID,
	).Scan(&member.ID, &member.RoomID, &member.UserID, &member.IsOwner, &member.JoinedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return member, err
}

func (r *RoomMemberRepository) GetByRoomID(roomID string) ([]*models.RoomMember, error) {
	rows, err := r.db.Query(
		"SELECT id, room_id, user_id, is_owner, joined_at FROM room_members WHERE room_id = ?",
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*models.RoomMember
	for rows.Next() {
		member := &models.RoomMember{}
		if err := rows.Scan(&member.ID, &member.RoomID, &member.UserID, &member.IsOwner, &member.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (r *RoomMemberRepository) GetByUserID(userID string) (*models.RoomMember, error) {
	member := &models.RoomMember{}
	err := r.db.QueryRow(
		"SELECT id, room_id, user_id, is_owner, joined_at FROM room_members WHERE user_id = ?",
		userID,
	).Scan(&member.ID, &member.RoomID, &member.UserID, &member.IsOwner, &member.JoinedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return member, err
}

func (r *RoomMemberRepository) UpdateOwner(roomID, userID string, isOwner bool) error {
	_, err := r.db.Exec(
		"UPDATE room_members SET is_owner = ? WHERE room_id = ? AND user_id = ?",
		isOwner, roomID, userID,
	)
	return err
}

func (r *RoomMemberRepository) Delete(roomID, userID string) error {
	_, err := r.db.Exec(
		"DELETE FROM room_members WHERE room_id = ? AND user_id = ?",
		roomID, userID,
	)
	return err
}

func (r *RoomMemberRepository) GetOwnerByRoomID(roomID string) (*models.RoomMember, error) {
	member := &models.RoomMember{}
	err := r.db.QueryRow(
		"SELECT id, room_id, user_id, is_owner, joined_at FROM room_members WHERE room_id = ? AND is_owner = 1 LIMIT 1",
		roomID,
	).Scan(&member.ID, &member.RoomID, &member.UserID, &member.IsOwner, &member.JoinedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return member, err
}

func (r *RoomMemberRepository) HasOwner(roomID string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM room_members WHERE room_id = ? AND is_owner = 1",
		roomID,
	).Scan(&count)
	return count > 0, err
}