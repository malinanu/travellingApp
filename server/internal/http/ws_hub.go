package httpapi

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// GroupHub maintains active WebSocket connections grouped by Group ID.
type GroupHub struct {
	mu sync.RWMutex
	// map of GroupID -> map of *websocket.Conn -> UserID
	rooms map[string]map[*websocket.Conn]string
}

// NewGroupHub creates a new Hub.
func NewGroupHub() *GroupHub {
	return &GroupHub{
		rooms: make(map[string]map[*websocket.Conn]string),
	}
}

// Register adds a new connection to a group room.
func (h *GroupHub) Register(groupID, userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[groupID] == nil {
		h.rooms[groupID] = make(map[*websocket.Conn]string)
	}
	h.rooms[groupID][conn] = userID
}

// Unregister removes a connection from a group room.
func (h *GroupHub) Unregister(groupID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[groupID]; ok {
		delete(room, conn)
		if len(room) == 0 {
			delete(h.rooms, groupID)
		}
	}
	conn.Close()
}

// Broadcast sends a message to all connections in a group room, except the sender.
func (h *GroupHub) Broadcast(groupID string, senderConn *websocket.Conn, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, ok := h.rooms[groupID]
	if !ok {
		return
	}

	for conn := range room {
		if conn != senderConn {
			// use a short timeout for writing to avoid blocking
			conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				// We don't remove the connection here, let the read loop detect the error and unregister.
			}
		}
	}
}
