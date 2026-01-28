package ws

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client

	// Inbound messages from the clients.
	Broadcast chan []byte

	// Map to quickly find clients by UserID (critical for private messaging)
	userClients map[uint][]*Client

	// Mutex to protect the userClients map
	mutex sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:   make(chan []byte),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		userClients: make(map[uint][]*Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.clients[client] = true
			h.limitRegister(client)
		case client := <-h.Unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				h.limitUnregister(client)
			}
		case message := <-h.Broadcast:
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// limitRegister registers a client to the specific user map
func (h *Hub) limitRegister(client *Client) {
	h.mutex.Lock()

	// Add client to userClients map
	h.userClients[client.UserID] = append(h.userClients[client.UserID], client)
	count := len(h.userClients[client.UserID])

	// Collect list of currently online users to send to the new client
	// We do this while holding the lock
	onlineUserIDs := make([]uint, 0, len(h.userClients))
	for userID := range h.userClients {
		onlineUserIDs = append(onlineUserIDs, userID)
	}

	h.mutex.Unlock()

	log.Printf("User %d connected. Total connections for user: %d", client.UserID, count)

	// NOTE: We do NOT update database for online status anymore
	// Online status is purely in-memory via WebSocket hub

	// 2. Broadcast Status Change (Online)
	statusJSON, _ := json.Marshal(map[string]interface{}{
		"type":      "user_status",
		"user_id":   client.UserID,
		"is_online": true,
	})
	go func() {
		h.Broadcast <- statusJSON
	}()

	// 3. Send Initial Online List to THIS Client
	// This fixes the bug where only one user sees the other, but not vice versa.
	if len(onlineUserIDs) > 0 {
		initialStatusJSON, _ := json.Marshal(map[string]interface{}{
			"type":     "online_users_list",
			"user_ids": onlineUserIDs,
		})
		go func() {
			client.Send <- initialStatusJSON
		}()
	}

	// NOTE: We do NOT call SendUnreadMessages here anymore.
	// Unread messages are now sent only when the user joins a specific room
	// via SendUnreadMessagesForRoom, which is called from the join_room handler.
	// This prevents race conditions where messages are sent before the client
	// has set its activeRoomId.
}

// limitUnregister removes a client from the specific user map
func (h *Hub) limitUnregister(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	userConns := h.userClients[client.UserID]
	for i, conn := range userConns {
		if conn == client {
			// Remove the client from the slice
			h.userClients[client.UserID] = append(userConns[:i], userConns[i+1:]...)
			break
		}
	}

	count := len(h.userClients[client.UserID])
	if count == 0 {
		delete(h.userClients, client.UserID)

		// NOTE: We do NOT update database for online status anymore
		// Online status is purely in-memory via WebSocket hub

		// Broadcast Status Change (Offline)
		statusJSON, _ := json.Marshal(map[string]interface{}{
			"type":      "user_status",
			"user_id":   client.UserID,
			"is_online": false,
		})

		go func() {
			h.Broadcast <- statusJSON
		}()

		log.Printf("User %d disconnected (Offline)", client.UserID)
	} else {
		log.Printf("User %d disconnected (Still has %d connections)", client.UserID, count)
	}
}

// SendToUser sends a message to a specific user (all their active connections)
func (h *Hub) SendToUser(userID uint, message []byte) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if clients, ok := h.userClients[userID]; ok {
		for _, client := range clients {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
}

// IsUserInRoom checks if a user has any active connection in the specified room
func (h *Hub) IsUserInRoom(userID uint, roomID uint) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if clients, ok := h.userClients[userID]; ok {
		for _, client := range clients {
			// locking client to safely read ActiveRoomID
			client.mu.Lock()
			activeRoom := client.ActiveRoomID
			client.mu.Unlock()
			if activeRoom == roomID {
				return true
			}
		}
	}
	return false
}

// GetUsersInRoom returns all user IDs currently active in a specific room
func (h *Hub) GetUsersInRoom(roomID uint) []uint {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	var usersInRoom []uint
	seen := make(map[uint]bool)

	for userID, clients := range h.userClients {
		for _, client := range clients {
			client.mu.Lock()
			activeRoom := client.ActiveRoomID
			client.mu.Unlock()

			if activeRoom == roomID && !seen[userID] {
				usersInRoom = append(usersInRoom, userID)
				seen[userID] = true
				break // Only need to add once per user
			}
		}
	}
	return usersInRoom
}

// IsUserOnline checks if a user has any active WebSocket connection (in-memory check)
func (h *Hub) IsUserOnline(userID uint) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	clients, ok := h.userClients[userID]
	return ok && len(clients) > 0
}
