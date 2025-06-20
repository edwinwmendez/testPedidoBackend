package ws

import (
	"encoding/json"
	"log"
)

type MessageType string

const (
	OrderStatusUpdate MessageType = "order_status_update"
	NewOrderAvailable MessageType = "new_order_available"
	CategoryUpdate    MessageType = "category_update"
	ProductUpdate     MessageType = "product_update"
	ChatMessage       MessageType = "chat_message" // Futuro
)

type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type OrderStatusUpdatePayload struct {
	OrderID          string `json:"order_id"`
	NewStatus        string `json:"new_status"`
	Message          string `json:"message"`
	EstimatedArrival string `json:"estimated_arrival_time,omitempty"`
}

type NewOrderAvailablePayload struct {
	OrderID       string `json:"order_id"`
	ClientAddress string `json:"client_address"`
	TotalAmount   string `json:"total_amount"`
	OrderTime     string `json:"order_time"`
}

type CategoryUpdatePayload struct {
	CategoryID string `json:"category_id"`
	Action     string `json:"action"` // created, updated, deleted
	Category   string `json:"category,omitempty"`
}

type ProductUpdatePayload struct {
	ProductID string `json:"product_id"`
	Action    string `json:"action"` // created, updated, deleted, stock_updated
	Product   string `json:"product,omitempty"`
}

// MustMarshalPayload serializa un struct a json.RawMessage y hace log si falla.
func MustMarshalPayload(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("[WebSocket] Error serializando payload: %v", err)
		return nil
	}
	return b
}
