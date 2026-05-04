package repository

import (
	"database/sql"

	"tomatogether/backend/internal/models"
)

type AnnouncementRepository struct {
	db *sql.DB
}

func NewAnnouncementRepository(db *sql.DB) *AnnouncementRepository {
	return &AnnouncementRepository{db: db}
}

func (r *AnnouncementRepository) Create(announcement *models.Announcement) error {
	_, err := r.db.Exec(
		"INSERT INTO announcements (id, room_id, sender_id, title, body, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		announcement.ID, announcement.RoomID, announcement.SenderID, announcement.Title, announcement.Body, announcement.CreatedAt,
	)
	return err
}

func (r *AnnouncementRepository) GetByRoomID(roomID string, limit int) ([]*models.Announcement, error) {
	rows, err := r.db.Query(
		"SELECT id, room_id, sender_id, title, body, created_at FROM announcements WHERE room_id = ? ORDER BY created_at DESC LIMIT ?",
		roomID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var announcements []*models.Announcement
	for rows.Next() {
		announcement := &models.Announcement{}
		if err := rows.Scan(&announcement.ID, &announcement.RoomID, &announcement.SenderID, &announcement.Title, &announcement.Body, &announcement.CreatedAt); err != nil {
			return nil, err
		}
		announcements = append(announcements, announcement)
	}
	return announcements, rows.Err()
}

func (r *AnnouncementRepository) GetByID(id string) (*models.Announcement, error) {
	announcement := &models.Announcement{}
	err := r.db.QueryRow(
		"SELECT id, room_id, sender_id, title, body, created_at FROM announcements WHERE id = ?",
		id,
	).Scan(&announcement.ID, &announcement.RoomID, &announcement.SenderID, &announcement.Title, &announcement.Body, &announcement.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return announcement, err
}