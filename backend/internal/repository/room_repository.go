package repository

import (
	"database/sql"

	"tomatogether/backend/internal/models"
)

type RoomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) Create(room *models.Room) error {
	_, err := r.db.Exec(
		"INSERT INTO rooms (id, name, password, is_readonly, created_at) VALUES (?, ?, ?, ?, ?)",
		room.ID, room.Name, room.Password, room.IsReadonly, room.CreatedAt,
	)
	return err
}

func (r *RoomRepository) GetByID(id string) (*models.Room, error) {
	room := &models.Room{}
	var password string
	err := r.db.QueryRow(
		"SELECT id, name, password, is_readonly, created_at FROM rooms WHERE id = ?",
		id,
	).Scan(&room.ID, &room.Name, &password, &room.IsReadonly, &room.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	room.Password = password
	room.HasPassword = password != ""
	return room, err
}

func (r *RoomRepository) GetByName(name string) (*models.Room, error) {
	room := &models.Room{}
	var password string
	err := r.db.QueryRow(
		"SELECT id, name, password, is_readonly, created_at FROM rooms WHERE name = ?",
		name,
	).Scan(&room.ID, &room.Name, &password, &room.IsReadonly, &room.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	room.Password = password
	room.HasPassword = password != ""
	return room, err
}

func (r *RoomRepository) Update(room *models.Room) error {
	_, err := r.db.Exec(
		"UPDATE rooms SET name = ?, password = ?, is_readonly = ? WHERE id = ?",
		room.Name, room.Password, room.IsReadonly, room.ID,
	)
	return err
}

func (r *RoomRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM rooms WHERE id = ?", id)
	return err
}

func (r *RoomRepository) ExistsByName(name string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM rooms WHERE name = ?",
		name,
	).Scan(&count)
	return count > 0, err
}

func (r *RoomRepository) GetMemberCount(roomID string) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM room_members WHERE room_id = ?",
		roomID,
	).Scan(&count)
	return count, err
}