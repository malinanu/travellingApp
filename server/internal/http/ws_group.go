package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for prototype
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// LocationUpdate represents a single live location broadcast.
type LocationUpdate struct {
	UserID string  `json:"userId"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
}

// handleGroupWS handles WebSocket requests from the peer.
func (s *Server) handleGroupWS(w http.ResponseWriter, r *http.Request) {
	groupID := r.URL.Query().Get("groupId")
	userID := r.URL.Query().Get("userId")

	if groupID == "" || userID == "" {
		http.Error(w, "missing groupId or userId", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("websocket upgrade error:", err)
		return
	}
	defer conn.Close()

	s.GroupHub.Register(groupID, userID, conn)
	defer s.GroupHub.Unregister(groupID, conn)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error: %v", err)
			}
			break
		}

		// Parse the incoming JSON to extract coordinates
		var update LocationUpdate
		if err := json.Unmarshal(message, &update); err == nil {
			// Ensure the broadcast includes the correct userID
			update.UserID = userID
			
			// Re-encode to broadcast
			broadcastMsg, _ := json.Marshal(update)
			s.GroupHub.Broadcast(groupID, conn, broadcastMsg)
		}
	}
}
