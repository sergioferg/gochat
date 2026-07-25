package ws

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client wraps the raw WebSocket connection
type Client struct {
	Conn   *websocket.Conn
	UserID uuid.UUID
}

// Manager holds all active WebSocket connections on THIS specific server/container
type Manager struct {
	sync.RWMutex // Protects the map from concurrent map writes
	Clients      map[uuid.UUID][]*Client
}

func NewManager() *Manager {
	return &Manager{
		Clients: make(map[uuid.UUID][]*Client),
	}
}

// Registers a new connection when a user logs in
func (m *Manager) AddClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.Clients[client.UserID] = append(m.Clients[client.UserID], client)
}

// Cleans up when a user disconnects or closes the tab
func (m *Manager) RemoveClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	connections := m.Clients[client.UserID]
	for i, c := range connections {
		if c == client {
			// Remove the specific connection from the slice
			m.Clients[client.UserID] = append(connections[:i], connections[i+1:]...)
			break
		}
	}

	// If that was their last active device, remove them from the map entirely
	if len(m.Clients[client.UserID]) == 0 {
		delete(m.Clients, client.UserID)
	}
}

// SendToUser pushes a JSON payload to every active device the user has connected to THIS server
func (m *Manager) SendToUser(userID uuid.UUID, payload interface{}) {
	m.RLock()
	defer m.RUnlock()

	connections, exists := m.Clients[userID]
	if !exists {

		return
	}

	for _, client := range connections {
		err := client.Conn.WriteJSON(payload)
		if err != nil {
			// If a write fails (e.g., dropped connection), the read pump will catch it
			// and call RemoveClient, so we just ignore the error here.
			continue
		}
	}
}
