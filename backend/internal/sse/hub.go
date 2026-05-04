package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

// Hub manages all room connections and broadcasts events
type Hub struct {
	rooms      map[string]map[*Client]bool // room name -> clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mu         sync.RWMutex
	repo       *repository.Repository

	// Heartbeat configuration
	heartbeatTimeout time.Duration // 2 minutes
	tickInterval     time.Duration // 5 seconds

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// Client represents a single SSE connection
type Client struct {
	RoomName    string
	MemberID    string
	Username    string
	IsOwner     bool
	IsAuth      bool // has L2 token
	notify      chan []byte
	hub         *Hub
	connectedAt time.Time
	lastPing    time.Time
}

// Message represents an event to broadcast
type Message struct {
	RoomName string
	Event    string
	Data     interface{}
}

var globalHub *Hub

// NewHub creates a new Hub instance
func NewHub(repo *repository.Repository) *Hub {
	hub := &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 256),
		repo:       repo,

		heartbeatTimeout: 2 * time.Minute,
		tickInterval:     5 * time.Second,

		stopCh: make(chan struct{}),
	}

	globalHub = hub
	return hub
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	// Heartbeat checker goroutine
	h.wg.Add(1)
	go h.runHeartbeatChecker()

	// Tick broadcaster goroutine
	h.wg.Add(1)
	go h.runTickBroadcaster()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.RoomName] == nil {
				h.rooms[client.RoomName] = make(map[*Client]bool)
			}
			h.rooms[client.RoomName][client] = true
			h.mu.Unlock()

			log.Printf("Client registered: room=%s, member=%s", client.RoomName, client.MemberID)

			// Broadcast user_joined when authenticated client connects
			if client.IsAuth && client.MemberID != "" {
				h.broadcastToRoom(client.RoomName, "user_joined", map[string]interface{}{
					"user": map[string]interface{}{
						"id":        client.MemberID,
						"username":  client.Username,
						"is_online": true,
					},
				})
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.RoomName]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.notify)

					// Notify others that user left
					h.broadcastToRoom(client.RoomName, "user_left", map[string]interface{}{
						"user_id":  client.MemberID,
						"username": client.Username,
					})

					// If last client in room, clean up
					if len(clients) == 0 {
						delete(h.rooms, client.RoomName)
					}
				}
			}
			h.mu.Unlock()

			log.Printf("Client unregistered: room=%s, member=%s", client.RoomName, client.MemberID)

		case message := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.rooms[message.RoomName]; ok {
				data, _ := json.Marshal(message.Data)
				event := fmt.Sprintf("event: %s\ndata: %s\n\n", message.Event, string(data))
				for client := range clients {
					select {
					case client.notify <- []byte(event):
					default:
						// Client buffer full, skip
					}
				}
			}
			h.mu.RUnlock()

		case <-h.stopCh:
			return
		}
	}
}

// Stop gracefully stops the hub
func (h *Hub) Stop() {
	close(h.stopCh)
	h.wg.Wait()
}

// NewClient creates a new SSE client
func (h *Hub) NewClient(roomName, memberID, username string, isOwner, isAuth bool) *Client {
	return &Client{
		RoomName:    roomName,
		MemberID:    memberID,
		Username:    username,
		IsOwner:     isOwner,
		IsAuth:      isAuth,
		notify:      make(chan []byte, 256),
		hub:         h,
		connectedAt: time.Now(),
		lastPing:    time.Now(),
	}
}

// Register registers a client with the hub
func (c *Client) Register() {
	c.hub.register <- c
}

// Unregister unregisters a client from the hub
func (c *Client) Unregister() {
	c.hub.unregister <- c
}

// Ping updates the client's last ping time (heartbeat)
func (c *Client) Ping() {
	c.lastPing = time.Now()
}

// Notify returns the notification channel for receiving SSE events
func (c *Client) Notify() <-chan []byte {
	return c.notify
}

// BroadcastEvent sends an event to all clients in a room
func (h *Hub) BroadcastEvent(roomName, event string, data interface{}) {
	h.broadcast <- &Message{
		RoomName: roomName,
		Event:    event,
		Data:     data,
	}
}

// broadcastToRoom sends an event to all clients in a room
func (h *Hub) broadcastToRoom(roomName, event string, data interface{}) {
	h.broadcast <- &Message{
		RoomName: roomName,
		Event:    event,
		Data:     data,
	}
}

// GetOnlineUsers returns all online users in a room based on heartbeat
func (h *Hub) GetOnlineUsers(roomName string) []*models.UserInfo {
	users := make([]*models.UserInfo, 0)

	h.mu.RLock()
	clients := h.rooms[roomName]
	h.mu.RUnlock()

	for client := range clients {
		user := &models.UserInfo{
			ID:       client.MemberID,
			Username: client.Username,
			IsOwner:  client.IsOwner,
			IsOnline: true,
		}
		users = append(users, user)
	}

	return users
}

// runHeartbeatChecker periodically checks for stale connections
func (h *Hub) runHeartbeatChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.mu.Lock()
			for roomName, clients := range h.rooms {
				for client := range clients {
					if time.Since(client.lastPing) > h.heartbeatTimeout {
						// Client is stale, unregister
						delete(clients, client)
						close(client.notify)

						// Notify others
						h.broadcastToRoom(roomName, "user_left", map[string]interface{}{
							"user_id":  client.MemberID,
							"username": client.Username,
						})

						log.Printf("Client timed out: room=%s, member=%s", roomName, client.MemberID)
					}
				}
				if len(clients) == 0 {
					delete(h.rooms, roomName)
				}
			}
			h.mu.Unlock()

		case <-h.stopCh:
			return
		}
	}
}

// runTickBroadcaster sends tick events to all rooms periodically
func (h *Hub) runTickBroadcaster() {
	ticker := time.NewTicker(h.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.mu.RLock()
			for _, clients := range h.rooms {
				if len(clients) == 0 {
					continue
				}

				// Build user states for tick
				users := make([]map[string]interface{}, 0)
				for client := range clients {
					var remainingSeconds int
					var phase = "idle"

					session, err := h.repo.GetActiveSessionByMemberID(client.MemberID)
					if err == nil && session != nil {
						elapsed := int(time.Since(session.StartedAt).Seconds())
						remaining := session.PlannedDuration - elapsed
						if remaining < 0 {
							remaining = 0
						}
						remainingSeconds = remaining

						if session.PausedAt != nil {
							phase = "paused"
							pausedElapsed := int(session.PausedAt.Sub(session.StartedAt).Seconds())
							remainingSeconds = session.PlannedDuration - pausedElapsed
							if remainingSeconds < 0 {
								remainingSeconds = 0
							}
						} else if session.IsFollowed {
							phase = "following"
						} else {
							phase = "focusing"
						}
					} else {
						// Check rest phase
						latest, err := h.repo.GetLatestSessionByMemberID(client.MemberID)
						if err == nil && latest != nil && latest.EndedAt != nil && latest.RestDuration > 0 {
							restEnd := latest.EndedAt.Add(time.Duration(latest.RestDuration) * time.Second)
							if time.Now().Before(restEnd) {
								remaining := int(time.Until(restEnd).Seconds())
								if remaining < 0 {
									remaining = 0
								}
								remainingSeconds = remaining
								phase = "rest"
							}
						}
					}

					users = append(users, map[string]interface{}{
						"id":                client.MemberID,
						"username":          client.Username,
						"remaining_seconds": remainingSeconds,
						"phase":             phase,
					})
				}

				// Broadcast tick
				data, _ := json.Marshal(map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
					"users":     users,
				})
				event := fmt.Sprintf("event: tick\ndata: %s\n\n", string(data))

				for client := range clients {
					select {
					case client.notify <- []byte(event):
					default:
					}
				}
			}
			h.mu.RUnlock()

		case <-h.stopCh:
			return
		}
	}
}

// GetHub returns the global hub instance
func GetHub() *Hub {
	return globalHub
}
