package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/spotto/server/internal/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Only the mobile app uses this socket, and it sends no Origin header.
	// Rejecting every browser origin shuts out cross-site WebSocket
	// hijacking, on top of the bearer token the route already requires.
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == ""
	},
}

// LocationUpdate represents a single live location broadcast.
type LocationUpdate struct {
	UserID string  `json:"userId"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
}

// handleGroupWS streams live locations between members of one group. It
// sits behind Authenticated: the sender's identity comes from the token,
// never from the URL, and only the group's owner or members can join.
// Otherwise anyone holding a group code could watch members' positions.
func (s *Server) handleGroupWS(w http.ResponseWriter, r *http.Request) {
	userID := auth.FromContext(r.Context()).Subject
	groupID := r.URL.Query().Get("groupId")
	if groupID == "" {
		writeErr(w, http.StatusBadRequest, "missing groupId")
		return
	}
	ok, err := s.Store.IsGroupParticipant(r.Context(), groupID, userID)
	if err != nil {
		log.Printf("group ws membership %s/%s: %v", groupID, userID, err)
		writeErr(w, http.StatusInternalServerError, "couldn't check group membership")
		return
	}
	if !ok {
		writeErr(w, http.StatusForbidden, "you're not in this group")
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
