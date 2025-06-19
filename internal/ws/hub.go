package ws

import (
	"log"
	"sync"
)

// Hub gestiona todas las conexiones activas y el broadcast de mensajes.
type Hub struct {
	clients    map[string]*Client // key: userID
	byRole     map[string]map[string]*Client // role -> userID -> *Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	mu         sync.RWMutex
}

// NewHub crea una nueva instancia del Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		byRole:     make(map[string]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message),
	}
}

// Run inicia el loop principal del hub.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = client
			if h.byRole[client.role] == nil {
				h.byRole[client.role] = make(map[string]*Client)
			}
			h.byRole[client.role][client.userID] = client
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				delete(h.byRole[client.role], client.userID)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client.userID)
					delete(h.byRole[client.role], client.userID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToUser envía un mensaje a un usuario específico.
func (h *Hub) SendToUser(userID string, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if client, ok := h.clients[userID]; ok {
		client.send <- msg
	}
}

// SendToRole envía un mensaje a todos los usuarios de un rol específico.
func (h *Hub) SendToRole(role string, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	log.Printf("[WebSocket Hub] Enviando mensaje tipo '%s' a rol '%s'", msg.Type, role)
	log.Printf("[WebSocket Hub] Clientes conectados por rol: %v", func() map[string]int {
		counts := make(map[string]int)
		for r, clients := range h.byRole {
			counts[r] = len(clients)
		}
		return counts
	}())
	
	clients := h.byRole[role]
	log.Printf("[WebSocket Hub] Enviando a %d clientes del rol '%s'", len(clients), role)
	
	for userID, client := range clients {
		log.Printf("[WebSocket Hub] Enviando mensaje a cliente %s (rol: %s)", userID, role)
		select {
		case client.send <- msg:
			log.Printf("[WebSocket Hub] Mensaje enviado exitosamente a cliente %s", userID)
		default:
			log.Printf("[WebSocket Hub] Canal lleno para cliente %s, cerrando conexión", userID)
			close(client.send)
			delete(h.clients, userID)
			delete(h.byRole[role], userID)
		}
	}
}

// Broadcast envía un mensaje a todos los clientes conectados.
func (h *Hub) Broadcast(msg Message) {
	h.broadcast <- msg
}
