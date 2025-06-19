package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderStatus define los estados posibles de un pedido
type OrderStatus string

const (
	OrderStatusPending           OrderStatus = "PENDING"
	OrderStatusPendingOutOfHours OrderStatus = "PENDING_OUT_OF_HOURS"
	OrderStatusConfirmed         OrderStatus = "CONFIRMED"
	OrderStatusAssigned          OrderStatus = "ASSIGNED"
	OrderStatusInTransit         OrderStatus = "IN_TRANSIT"
	OrderStatusDelivered         OrderStatus = "DELIVERED"
	OrderStatusCancelled         OrderStatus = "CANCELLED"
)

// Order representa un pedido en el sistema
type Order struct {
	OrderID              uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"order_id"`
	ClientID             uuid.UUID   `gorm:"type:uuid;not null" json:"client_id"`
	Client               User        `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	TotalAmount          float64     `gorm:"type:decimal(10,2);not null;check:total_amount >= 0" json:"total_amount"`
	Latitude             float64     `gorm:"type:numeric(9,6);not null" json:"latitude"`
	Longitude            float64     `gorm:"type:numeric(9,6);not null" json:"longitude"`
	DeliveryAddressText  string      `gorm:"type:text;not null" json:"delivery_address_text"`
	PaymentNote          string      `gorm:"type:varchar(255)" json:"payment_note"`
	OrderStatus          OrderStatus `gorm:"type:varchar(20);not null" json:"order_status"`
	OrderTime            time.Time   `gorm:"not null" json:"order_time"`
	ConfirmedAt          *time.Time  `json:"confirmed_at"`
	EstimatedArrivalTime *time.Time  `json:"estimated_arrival_time"`
	AssignedRepartidorID *uuid.UUID  `gorm:"type:uuid" json:"assigned_repartidor_id"`
	AssignedRepartidor   *User       `gorm:"foreignKey:AssignedRepartidorID" json:"assigned_repartidor,omitempty"`
	AssignedAt           *time.Time  `json:"assigned_at"`
	DeliveredAt          *time.Time  `json:"delivered_at"`
	CancelledAt          *time.Time  `json:"cancelled_at"`
	CreatedAt            time.Time   `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt            time.Time   `gorm:"not null;default:now()" json:"updated_at"`
	OrderItems           []OrderItem `gorm:"foreignKey:OrderID" json:"order_items,omitempty"`
}

// OrderItem representa un ítem dentro de un pedido
type OrderItem struct {
	OrderItemID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"order_item_id"`
	OrderID     uuid.UUID `gorm:"type:uuid;not null" json:"order_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Product     Product   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Quantity    int       `gorm:"type:integer;not null;check:quantity > 0" json:"quantity"`
	UnitPrice   float64   `gorm:"type:decimal(10,2);not null;check:unit_price > 0" json:"unit_price"`
	Subtotal    float64   `gorm:"type:decimal(10,2);not null;check:subtotal >= 0" json:"subtotal"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// BeforeCreate se ejecuta antes de crear un nuevo pedido
func (o *Order) BeforeCreate(tx *gorm.DB) (err error) {
	// Si no se proporciona un ID, generamos uno
	if o.OrderID == uuid.Nil {
		o.OrderID = uuid.New()
	}
	return nil
}

// BeforeCreate se ejecuta antes de crear un nuevo ítem de pedido
func (oi *OrderItem) BeforeCreate(tx *gorm.DB) (err error) {
	// Si no se proporciona un ID, generamos uno
	if oi.OrderItemID == uuid.Nil {
		oi.OrderItemID = uuid.New()
	}
	// Calculamos el subtotal
	oi.Subtotal = float64(oi.Quantity) * oi.UnitPrice
	return nil
}

// TableName especifica el nombre de la tabla para Order
func (Order) TableName() string {
	return "orders"
}

// TableName especifica el nombre de la tabla para OrderItem
func (OrderItem) TableName() string {
	return "order_items"
}

// IsWithinBusinessHours verifica si la hora actual está dentro del horario de atención
func IsWithinBusinessHours(t time.Time, businessStart, businessEnd time.Duration, timezone string) bool {
	// Convertir la hora a la zona horaria especificada
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Si hay error, usamos UTC
		loc = time.UTC
	}

	localTime := t.In(loc)

	// Extraer la hora y minutos como duración desde el inicio del día
	hour := time.Duration(localTime.Hour()) * time.Hour
	minute := time.Duration(localTime.Minute()) * time.Minute
	currentTime := hour + minute

	// Verificar si está dentro del horario de atención
	return currentTime >= businessStart && currentTime < businessEnd
}

// CanTransitionTo verifica si un pedido puede cambiar al estado especificado
func (o *Order) CanTransitionTo(newStatus OrderStatus) bool {
	switch o.OrderStatus {
	case OrderStatusPending, OrderStatusPendingOutOfHours:
		// Desde pendiente puede pasar a confirmado o cancelado
		return newStatus == OrderStatusConfirmed || newStatus == OrderStatusCancelled
	case OrderStatusConfirmed:
		// Desde confirmado puede pasar a asignado o cancelado
		return newStatus == OrderStatusAssigned || newStatus == OrderStatusCancelled
	case OrderStatusAssigned:
		// Desde asignado solo puede pasar a en tránsito (por el repartidor asignado)
		return newStatus == OrderStatusInTransit
	case OrderStatusInTransit:
		// Desde en tránsito solo puede pasar a entregado
		return newStatus == OrderStatusDelivered
	default:
		// Estados finales no pueden transicionar
		return false
	}
}
