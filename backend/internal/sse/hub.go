package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/repository"
)

const (
	maxGlobalConnections  = 500 // max total SSE connections
	maxPerIPConnections   = 20  // max connections per IP
	maxL1PerIPConnections = 5   // max unauthenticated connections per IP
)

// Hub manages all room connections and broadcasts events
type Hub struct {
	rooms      map[string]map[*Client]bool // room name -> clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mu         sync.RWMutex
	repo       *repository.Repository

	// Connection tracking per IP
	connectionsByIP map[string]int

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
	IP          string
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
		rooms:           make(map[string]map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		broadcast:       make(chan *Message, 256),
		repo:            repo,
		connectionsByIP: make(map[string]int),

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
			h.connectionsByIP[client.IP]++
			shouldBroadcast := client.IsAuth && client.MemberID != ""
			roomName := client.RoomName
			memberID := client.MemberID
			username := client.Username
			h.mu.Unlock()

			log.Printf("Client registered: room=%s, member=%s", roomName, memberID)

			// Broadcast user_joined when authenticated client connects (outside lock, Sec #16 fix)
			if shouldBroadcast {
				h.broadcastToRoom(roomName, "user_joined", map[string]interface{}{
					"user": map[string]interface{}{
						"id":        memberID,
						"username":  username,
						"is_online": true,
					},
				})
			}

		case client := <-h.unregister:
			h.mu.Lock()
			var shouldBroadcast bool
			var roomName, memberID, username string
			if clients, ok := h.rooms[client.RoomName]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.notify)

					roomName = client.RoomName
					memberID = client.MemberID
					username = client.Username

					// Only broadcast user_left if this was the LAST connection for this member
					shouldBroadcast = memberID != ""
					if shouldBroadcast {
						for remainingClient := range clients {
							if remainingClient.MemberID == memberID {
								shouldBroadcast = false
								break
							}
						}
					}

					// Decrement IP connection counter
					h.connectionsByIP[client.IP]--
					if h.connectionsByIP[client.IP] <= 0 {
						delete(h.connectionsByIP, client.IP)
					}

					// If last client in room, clean up
					if len(clients) == 0 {
						delete(h.rooms, client.RoomName)
					}
				}
			}
			h.mu.Unlock()

			// Notify others that user left (outside lock, Sec #16 fix)
			if shouldBroadcast {
				h.broadcastToRoom(roomName, "user_left", map[string]interface{}{
					"user_id":  memberID,
					"username": username,
				})
			}

			log.Printf("Client unregistered: room=%s, member=%s", roomName, memberID)

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

// CanConnect checks if a new SSE connection should be allowed
func (h *Hub) CanConnect(r *http.Request, roomName, clientIP string, isAuth bool) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Global limit
	totalConnections := 0
	for _, clients := range h.rooms {
		totalConnections += len(clients)
	}
	if totalConnections >= maxGlobalConnections {
		return false
	}

	// Per-IP limit
	currentIPConns := h.connectionsByIP[clientIP]
	if currentIPConns >= maxPerIPConnections {
		return false
	}

	// Stricter limit for unauthenticated (L1 spectator) connections
	if !isAuth && currentIPConns >= maxL1PerIPConnections {
		return false
	}

	return true
}

// NewClient creates a new SSE client
func (h *Hub) NewClient(roomName, memberID, username string, isOwner, isAuth bool, ip string) *Client {
	return &Client{
		RoomName:    roomName,
		MemberID:    memberID,
		Username:    username,
		IsOwner:     isOwner,
		IsAuth:      isAuth,
		IP:          ip,
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

// countMemberConnections counts clients in room with matching memberID.
// Uses RLock internally.
func (h *Hub) countMemberConnections(memberID, roomName string) int {
	if memberID == "" {
		return 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.rooms[roomName]; ok {
		count := 0
		for client := range clients {
			if client.MemberID == memberID {
				count++
			}
		}
		return count
	}
	return 0
}

// GetOnlineUsers returns all online users in a room based on heartbeat
func (h *Hub) GetOnlineUsers(roomName string) []*models.UserInfo {
	h.mu.RLock()
	clients := h.rooms[roomName]
	h.mu.RUnlock()

	seen := make(map[string]bool)
	users := make([]*models.UserInfo, 0)

	for client := range clients {
		if client.MemberID == "" {
			continue
		}
		if seen[client.MemberID] {
			continue
		}
		seen[client.MemberID] = true
		users = append(users, &models.UserInfo{
			ID:       client.MemberID,
			Username: client.Username,
			IsOwner:  client.IsOwner,
			IsOnline: true,
		})
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
			// Collect stale clients first, then broadcast outside lock (Sec #16 fix)
			type staleInfo struct {
				roomName string
				memberID string
				username string
			}
			var staleClients []staleInfo

			h.mu.Lock()
			for roomName, clients := range h.rooms {
				for client := range clients {
					if time.Since(client.lastPing) > h.heartbeatTimeout {
						// Client is stale, unregister
						delete(clients, client)
						close(client.notify)

						// Decrement IP connection counter
						h.connectionsByIP[client.IP]--
						if h.connectionsByIP[client.IP] <= 0 {
							delete(h.connectionsByIP, client.IP)
						}

						staleClients = append(staleClients, staleInfo{
							roomName: roomName,
							memberID: client.MemberID,
							username: client.Username,
						})

						log.Printf("Client timed out: room=%s, member=%s", roomName, client.MemberID)
					}
				}
				if len(clients) == 0 {
					delete(h.rooms, roomName)
				}
			}
			h.mu.Unlock()

			// Broadcast user_left for stale clients only if last connection per member
			for _, info := range staleClients {
				if info.memberID != "" && h.countMemberConnections(info.memberID, info.roomName) == 0 {
					h.broadcastToRoom(info.roomName, "user_left", map[string]interface{}{
						"user_id":  info.memberID,
						"username": info.username,
					})
				}
			}

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
						// time.Since() uses Go's monotonic clock — safe against wall-clock adjustments.
						// Go 1.9+ guarantees monotonic clock for time.Since/time.Until.
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
