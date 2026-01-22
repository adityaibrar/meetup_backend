package ws

import (
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
	defer h.mutex.Unlock()
	h.userClients[client.UserID] = append(h.userClients[client.UserID], client)
	log.Printf("User %d connected. Total connections for user: %d", client.UserID, len(h.userClients[client.UserID]))
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

	if len(h.userClients[client.UserID]) == 0 {
		delete(h.userClients, client.UserID)
	}
	log.Printf("User %d disconnected", client.UserID)
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
