package sse

import (
	"encoding/json"
	"sync"
	"time"
)

// Event types
const (
	EventUserJoined         = "user_joined"
	EventUserLeft           = "user_left"
	EventUserOffline        = "user_offline"
	EventPomodoroStarted    = "pomodoro_started"
	EventPomodoroEnded      = "pomodoro_ended"
	EventPomodoroFollowed   = "pomodoro_followed"
	EventPomodoroUnfollowed = "pomodoro_unfollowed"
	EventLeaderAborted      = "leader_aborted"
	EventStatusUpdated      = "status_updated"
	EventTick               = "tick"
	EventAnnouncement       = "announcement"
	EventRoomSettingsChanged = "room_settings_changed"
	EventTokenExpired       = "token_expired"
	EventPong              = "pong"
)

// Client represents an SSE client connection
type Client struct {
	ID       string
	UserID   string
	RoomID   string
	Token    string
	Level    int // 1 = L1 (observer), 2 = L2 (member)
	Chan     chan []byte
	Hub      *Hub
	done     chan struct{}
}

func (c *Client) GetRoomID() string {
	return c.RoomID
}

// Hub manages all SSE connections
type Hub struct {
	clients    map[string]*Client
	rooms      map[string]map[string]*Client // roomID -> clientID -> client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
	ticker     *time.Ticker
	stopTick   chan struct{}
}

type BroadcastMessage struct {
	RoomID   string
	Event    string
	Data     interface{}
	Exclude  []string // client IDs to exclude
	AllRooms bool      // if true, broadcast to all rooms
}

// NewHub creates a new SSE hub
func NewHub() *Hub {
	return &Hub{
		clients:   make(map[string]*Client),
		rooms:     make(map[string]map[string]*Client),
		register:  make(chan *Client, 100),
		unregister: make(chan *Client, 100),
		broadcast: make(chan *BroadcastMessage, 100),
		stopTick:  make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			if h.rooms[client.RoomID] == nil {
				h.rooms[client.RoomID] = make(map[string]*Client)
			}
			h.rooms[client.RoomID][client.ID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				if room, ok := h.rooms[client.RoomID]; ok {
					delete(room, client.ID)
				}
				close(client.Chan)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			var targetClients map[string]*Client
			if message.AllRooms {
				// Broadcast to all rooms
				for _, room := range h.rooms {
					for _, client := range room {
						if !shouldExclude(client.ID, message.Exclude) {
							h.sendToClient(client, message.Event, message.Data)
						}
					}
				}
				h.mu.RUnlock()
				continue
			} else {
				targetClients = h.rooms[message.RoomID]
			}
			h.mu.RUnlock()

			if targetClients != nil {
				for clientID, client := range targetClients {
					if !shouldExclude(clientID, message.Exclude) {
						h.sendToClient(client, message.Event, message.Data)
					}
				}
			}

		case <-ticker.C:
			// Send tick to all rooms
			h.mu.RLock()
			for roomID, clients := range h.rooms {
				if len(clients) > 0 {
					h.mu.RUnlock()
					h.SendTick(roomID)
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()

		case <-h.stopTick:
			return
		}
	}
}

func shouldExclude(clientID string, exclude []string) bool {
	for _, id := range exclude {
		if id == clientID {
			return true
		}
	}
	return false
}

func (h *Hub) sendToClient(client *Client, event string, data interface{}) {
	select {
	case client.Chan <- formatEvent(event, data):
	default:
		// Client buffer full, skip
	}
}

// Register adds a new client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends an event to all clients in a room
func (h *Hub) Broadcast(roomID, event string, data interface{}) {
	h.broadcast <- &BroadcastMessage{
		RoomID: roomID,
		Event:  event,
		Data:   data,
	}
}

// BroadcastToAll sends an event to all clients in all rooms
func (h *Hub) BroadcastToAll(event string, data interface{}) {
	h.broadcast <- &BroadcastMessage{
		Event:    event,
		Data:     data,
		AllRooms: true,
	}
}

// SendTick sends a tick event with current room status
func (h *Hub) SendTick(roomID string) {
	h.mu.RLock()
	clients := h.rooms[roomID]
	h.mu.RUnlock()

	if clients == nil {
		return
	}

	// Collect user status for tick
	users := make([]UserTickInfo, 0, len(clients))
	for _, client := range clients {
		users = append(users, UserTickInfo{
			ID:       client.UserID,
			RoomID:   client.RoomID,
		})
	}

	h.Broadcast(roomID, EventTick, TickData{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Users:     users,
	})
}

// GetRoomClients returns all clients in a room
func (h *Hub) GetRoomClients(roomID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if room, ok := h.rooms[roomID]; ok {
		clients := make([]*Client, 0, len(room))
		for _, client := range room {
			clients = append(clients, client)
		}
		return clients
	}
	return nil
}

// GetClientByUserID returns a client by user ID in a room
func (h *Hub) GetClientByUserID(roomID, userID string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if room, ok := h.rooms[roomID]; ok {
		for _, client := range room {
			if client.UserID == userID {
				return client
			}
		}
	}
	return nil
}

// IsUserOnline checks if a user is online in a room
func (h *Hub) IsUserOnline(roomID, userID string) bool {
	return h.GetClientByUserID(roomID, userID) != nil
}

// UserTickInfo represents user info in tick events
type UserTickInfo struct {
	ID                string `json:"id"`
	RoomID            string `json:"room_id,omitempty"`
	Username          string `json:"username,omitempty"`
	RemainingSeconds  int    `json:"remaining_seconds,omitempty"`
	Status            string `json:"status,omitempty"` // focusing, following, idle, rest
}

// TickData represents the tick event data
type TickData struct {
	Timestamp string          `json:"timestamp"`
	Users     []UserTickInfo  `json:"users"`
}

// formatEvent formats an SSE event
func formatEvent(event string, data interface{}) []byte {
	jsonData, _ := json.Marshal(data)
	return []byte("event: " + event + "\ndata: " + string(jsonData) + "\n\n")
}

// Stop stops the hub's tick loop
func (h *Hub) Stop() {
	close(h.stopTick)
}