package ws

// HubInterface defines the interface for WebSocket hub operations
type HubInterface interface {
	SendToUser(userID string, msg Message)
	SendToRole(role string, msg Message)
	Broadcast(msg Message)
}

// Ensure Hub implements HubInterface
var _ HubInterface = (*Hub)(nil)
