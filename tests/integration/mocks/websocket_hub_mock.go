package mocks

import (
	"backend/internal/ws"
	"sync"
)

// Ensure MockWebSocketHub implements HubInterface
var _ ws.HubInterface = (*MockWebSocketHub)(nil)

// MockWebSocketHub simulates WebSocket Hub for testing
type MockWebSocketHub struct {
	// Store messages sent to specific users
	UserMessages map[string][]ws.Message
	// Store messages sent to specific roles
	RoleMessages map[string][]ws.Message
	// Store broadcast messages
	BroadcastMessages []ws.Message
	// Mutex for thread safety
	mu sync.RWMutex
}

// NewMockWebSocketHub creates a new mock WebSocket hub
func NewMockWebSocketHub() *MockWebSocketHub {
	return &MockWebSocketHub{
		UserMessages:      make(map[string][]ws.Message),
		RoleMessages:      make(map[string][]ws.Message),
		BroadcastMessages: make([]ws.Message, 0),
	}
}

// SendToUser simulates sending a message to a specific user
func (m *MockWebSocketHub) SendToUser(userID string, msg ws.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.UserMessages[userID] == nil {
		m.UserMessages[userID] = make([]ws.Message, 0)
	}
	m.UserMessages[userID] = append(m.UserMessages[userID], msg)
}

// SendToRole simulates sending a message to all users of a specific role
func (m *MockWebSocketHub) SendToRole(role string, msg ws.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.RoleMessages[role] == nil {
		m.RoleMessages[role] = make([]ws.Message, 0)
	}
	m.RoleMessages[role] = append(m.RoleMessages[role], msg)
}

// Broadcast simulates broadcasting a message to all connected clients
func (m *MockWebSocketHub) Broadcast(msg ws.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.BroadcastMessages = append(m.BroadcastMessages, msg)
}

// GetMessagesForUser returns all messages sent to a specific user
func (m *MockWebSocketHub) GetMessagesForUser(userID string) []ws.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if messages, exists := m.UserMessages[userID]; exists {
		// Return a copy to avoid race conditions
		result := make([]ws.Message, len(messages))
		copy(result, messages)
		return result
	}
	return []ws.Message{}
}

// GetMessagesForRole returns all messages sent to a specific role
func (m *MockWebSocketHub) GetMessagesForRole(role string) []ws.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if messages, exists := m.RoleMessages[role]; exists {
		// Return a copy to avoid race conditions
		result := make([]ws.Message, len(messages))
		copy(result, messages)
		return result
	}
	return []ws.Message{}
}

// GetBroadcastMessages returns all broadcast messages
func (m *MockWebSocketHub) GetBroadcastMessages() []ws.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make([]ws.Message, len(m.BroadcastMessages))
	copy(result, m.BroadcastMessages)
	return result
}

// Reset clears all stored messages
func (m *MockWebSocketHub) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.UserMessages = make(map[string][]ws.Message)
	m.RoleMessages = make(map[string][]ws.Message)
	m.BroadcastMessages = make([]ws.Message, 0)
}

// GetMessageCount returns the total number of messages sent
func (m *MockWebSocketHub) GetMessageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	count := len(m.BroadcastMessages)
	
	for _, messages := range m.UserMessages {
		count += len(messages)
	}
	
	for _, messages := range m.RoleMessages {
		count += len(messages)
	}
	
	return count
}