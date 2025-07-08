package ws

import (
	"log"

	"backend/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// WebSocketHandler es el handler HTTP para /ws/notifications.
func WebSocketHandler(hub *Hub, cfg *config.Config) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		token := c.Query("token")
		// log.Printf("[WebSocket] Token recibido: %s", token) // Comentado por seguridad
		userID, role, err := ValidateWebSocketToken(token, cfg)
		if err != nil {
			log.Printf("[WebSocket] Token inválido: %v", err)
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Token inválido"))
			return
		}
		log.Printf("[WebSocket] Conexión aceptada para userID=%s, role=%s", userID, role)

		client := &Client{
			conn:   c,
			userID: userID,
			role:   role,
			send:   make(chan Message, 256),
			hub:    hub,
		}
		hub.register <- client

		// 3. Iniciar goroutines de lectura y escritura
		go client.WritePump()
		client.ReadPump()
		log.Printf("[WebSocket] Conexión cerrada para userID=%s, role=%s", userID, role)
	})
}
