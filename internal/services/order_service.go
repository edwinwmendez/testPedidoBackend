package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"backend/config"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/ws"
)

var (
	ErrOrderNotFound        = errors.New("pedido no encontrado")
	ErrInvalidOrderStatus   = errors.New("estado de pedido inválido")
	ErrOutsideBusinessHours = errors.New("fuera del horario de atención")
	ErrInvalidTransition    = errors.New("transición de estado inválida")
	ErrOrderAlreadyAssigned = errors.New("pedido ya asignado")
	ErrUserNotFound         = errors.New("usuario no encontrado")
	ErrInvalidRole          = errors.New("rol de usuario inválido")
	ErrProductNotFound      = errors.New("producto no encontrado")
	ErrProductInactive      = errors.New("producto no está activo")
)

type OrderService struct {
	orderRepo           repositories.OrderRepository
	userRepo            repositories.UserRepository
	productRepo         repositories.ProductRepository
	notificationService *NotificationService
	config              *config.Config
	wsHub               ws.HubInterface
}

func NewOrderService(
	orderRepo repositories.OrderRepository,
	userRepo repositories.UserRepository,
	productRepo repositories.ProductRepository,
	notificationService *NotificationService,
	config *config.Config,
	wsHub ws.HubInterface,
) *OrderService {
	return &OrderService{
		orderRepo:           orderRepo,
		userRepo:            userRepo,
		productRepo:         productRepo,
		notificationService: notificationService,
		config:              config,
		wsHub:               wsHub,
	}
}

// CreateOrder crea un nuevo pedido verificando horario de atención
func (s *OrderService) CreateOrder(order *models.Order, items []models.OrderItem) (*models.Order, error) {
	// Verificar que el cliente existe
	client, err := s.userRepo.FindByID(order.ClientID.String())
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Verificar que el cliente tiene rol CLIENT
	if client.UserRole != models.UserRoleClient {
		return nil, ErrInvalidRole
	}

	// Verificar productos y calcular total
	var totalAmount float64 = 0
	for i := range items {
		product, err := s.productRepo.FindByID(items[i].ProductID.String())
		if err != nil {
			return nil, ErrProductNotFound
		}

		if !product.IsActive {
			return nil, ErrProductInactive
		}

		// Establecer precio unitario del producto al momento de la compra
		items[i].UnitPrice = product.Price
		items[i].Subtotal = float64(items[i].Quantity) * items[i].UnitPrice
		totalAmount += items[i].Subtotal
	}

	order.TotalAmount = totalAmount
	order.OrderTime = time.Now()

	// Verificar horario de atención
	isWithinHours := models.IsWithinBusinessHours(
		order.OrderTime,
		s.config.App.BusinessHoursStart,
		s.config.App.BusinessHoursEnd,
		s.config.App.TimeZone,
	)

	if isWithinHours {
		order.OrderStatus = models.OrderStatusPending
	} else {
		order.OrderStatus = models.OrderStatusPendingOutOfHours
	}

	// Crear el pedido
	if err := s.orderRepo.Create(order); err != nil {
		return nil, err
	}

	// Añadir los items al pedido
	for i := range items {
		items[i].OrderID = order.OrderID
		if err := s.orderRepo.AddOrderItem(&items[i]); err != nil {
			return nil, err
		}
	}

	// Recargar el pedido con datos completos (incluyendo Client)
	fullOrder, err := s.orderRepo.FindByID(order.OrderID.String())
	if err == nil {
		order = fullOrder
	}

	// Notificar a repartidores SIEMPRE (dentro o fuera de horario)
	s.notifyNewOrder(order)

	return order, nil
}

// GetOrderByID obtiene un pedido por su ID
func (s *OrderService) GetOrderByID(orderID string) (*models.Order, error) {
	return s.orderRepo.FindByID(orderID)
}

// GetOrdersByClientID obtiene todos los pedidos de un cliente
func (s *OrderService) GetOrdersByClientID(clientID string) ([]*models.Order, error) {
	return s.orderRepo.FindByClientID(clientID)
}

// GetOrdersByRepartidorID obtiene todos los pedidos asignados a un repartidor
func (s *OrderService) GetOrdersByRepartidorID(repartidorID string) ([]*models.Order, error) {
	return s.orderRepo.FindByRepartidorID(repartidorID)
}

// GetPendingOrders obtiene todos los pedidos pendientes
func (s *OrderService) GetPendingOrders() ([]*models.Order, error) {
	return s.orderRepo.FindPendingOrders()
}

// GetOrdersByStatus obtiene todos los pedidos con un estado específico
func (s *OrderService) GetOrdersByStatus(status models.OrderStatus) ([]*models.Order, error) {
	return s.orderRepo.FindByStatus(status)
}

// GetAllOrders obtiene todos los pedidos
func (s *OrderService) GetAllOrders() ([]*models.Order, error) {
	return s.orderRepo.FindAll()
}

// UpdateOrderStatus actualiza el estado de un pedido
func (s *OrderService) UpdateOrderStatus(orderID string, newStatus models.OrderStatus, userID string, userRole models.UserRole) (*models.Order, error) {
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	// Validar la transición de estado según el rol
	if !s.canUpdateStatus(order, newStatus, userID, userRole) {
		return nil, ErrInvalidTransition
	}

	// Si un repartidor está cambiando el estado a CONFIRMED, asignarlo automáticamente
	if userRole == models.UserRoleRepartidor && newStatus == models.OrderStatusConfirmed {
		// Solo asignar si no hay repartidor asignado aún
		if order.AssignedRepartidorID == nil {
			if err := s.orderRepo.AssignRepartidor(orderID, userID); err != nil {
				return nil, err
			}
		}
	}

	// Si se está cambiando a IN_TRANSIT pero no hay repartidor asignado, es un error
	if newStatus == models.OrderStatusInTransit && order.AssignedRepartidorID == nil {
		return nil, errors.New("no se puede cambiar a 'EN CAMINO' sin asignar un repartidor primero")
	}

	// Actualizar el estado
	if err := s.orderRepo.UpdateStatus(orderID, newStatus); err != nil {
		return nil, err
	}

	// Recargar el pedido con los datos actualizados
	updatedOrder, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, err
	}

	// Enviar notificación al cliente sobre el cambio de estado
	s.notifyStatusChange(updatedOrder)

	return updatedOrder, nil
}

// AssignRepartidor asigna un repartidor a un pedido
func (s *OrderService) AssignRepartidor(orderID string, repartidorID string) (*models.Order, error) {
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	// Verificar que el pedido esté en estado pendiente o confirmado
	if order.OrderStatus != models.OrderStatusPending && 
	   order.OrderStatus != models.OrderStatusPendingOutOfHours && 
	   order.OrderStatus != models.OrderStatusConfirmed {
		return nil, ErrInvalidOrderStatus
	}

	// Verificar que el pedido no esté ya asignado
	if order.AssignedRepartidorID != nil {
		return nil, ErrOrderAlreadyAssigned
	}

	// Verificar que el repartidor existe y tiene el rol correcto
	repartidor, err := s.userRepo.FindByID(repartidorID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if repartidor.UserRole != models.UserRoleRepartidor && repartidor.UserRole != models.UserRoleAdmin {
		return nil, ErrInvalidRole
	}

	// Asignar el repartidor y cambiar estado a asignado
	if err := s.orderRepo.AssignRepartidor(orderID, repartidorID); err != nil {
		return nil, err
	}

	// Cambiar estado a ASSIGNED después de asignar repartidor
	if err := s.orderRepo.UpdateStatus(orderID, models.OrderStatusAssigned); err != nil {
		return nil, err
	}

	// Recargar el pedido con los datos actualizados
	updatedOrder, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, err
	}

	// Notificar al cliente que su pedido ha sido asignado
	s.notifyOrderAssigned(updatedOrder)

	return updatedOrder, nil
}

// SetEstimatedArrivalTime establece el tiempo estimado de llegada para un pedido
func (s *OrderService) SetEstimatedArrivalTime(orderID string, eta time.Time) (*models.Order, error) {
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	// Verificar que el pedido esté confirmado, asignado o en tránsito
	if order.OrderStatus != models.OrderStatusConfirmed && 
	   order.OrderStatus != models.OrderStatusAssigned && 
	   order.OrderStatus != models.OrderStatusInTransit {
		return nil, ErrInvalidOrderStatus
	}

	// Establecer el tiempo estimado de llegada
	if err := s.orderRepo.SetEstimatedArrivalTime(orderID, eta); err != nil {
		return nil, err
	}

	// Recargar el pedido con los datos actualizados
	updatedOrder, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, err
	}

	// Notificar al cliente sobre el tiempo estimado de llegada
	s.notifyETA(updatedOrder)

	return updatedOrder, nil
}

// FindNearbyOrders encuentra pedidos cercanos a una ubicación
func (s *OrderService) FindNearbyOrders(lat, lng float64, radiusKm float64) ([]*models.Order, error) {
	return s.orderRepo.FindNearbyOrders(lat, lng, radiusKm)
}

// Métodos privados de ayuda

// canUpdateStatus verifica si un usuario puede actualizar el estado de un pedido
func (s *OrderService) canUpdateStatus(order *models.Order, newStatus models.OrderStatus, userID string, userRole models.UserRole) bool {
	switch userRole {
	case models.UserRoleAdmin:
		// Admin puede: PENDING -> CONFIRMED -> ASSIGNED
		validTransitions := map[models.OrderStatus][]models.OrderStatus{
			models.OrderStatusPending:           {models.OrderStatusConfirmed, models.OrderStatusCancelled},
			models.OrderStatusPendingOutOfHours: {models.OrderStatusConfirmed, models.OrderStatusCancelled},
			models.OrderStatusConfirmed:         {models.OrderStatusAssigned, models.OrderStatusCancelled},
		}
		
		if allowedStates, exists := validTransitions[order.OrderStatus]; exists {
			for _, allowed := range allowedStates {
				if newStatus == allowed {
					return true
				}
			}
		}
		return false
		
	case models.UserRoleRepartidor:
		// Repartidor puede tomar pedidos PENDING y manejar sus asignaciones
		if newStatus == models.OrderStatusConfirmed && 
		   (order.OrderStatus == models.OrderStatusPending || order.OrderStatus == models.OrderStatusPendingOutOfHours) {
			return true // Cualquier repartidor puede tomar pedidos
		}
		
		// Para estados avanzados, debe ser el repartidor asignado
		if order.AssignedRepartidorID != nil && order.AssignedRepartidorID.String() == userID {
			validTransitions := map[models.OrderStatus][]models.OrderStatus{
				models.OrderStatusAssigned:  {models.OrderStatusInTransit},
				models.OrderStatusInTransit: {models.OrderStatusDelivered},
			}
			
			if allowedStates, exists := validTransitions[order.OrderStatus]; exists {
				for _, allowed := range allowedStates {
					if newStatus == allowed {
						return true
					}
				}
			}
		}
		return false
		
	case models.UserRoleClient:
		// Cliente solo puede cancelar sus pedidos si están en estado inicial
		return newStatus == models.OrderStatusCancelled && 
		       order.ClientID.String() == userID && 
		       (order.OrderStatus == models.OrderStatusPending || order.OrderStatus == models.OrderStatusPendingOutOfHours)
	}
	
	return false
}

// Métodos de notificación

func (s *OrderService) notifyNewOrder(order *models.Order) {
	// Guard clause: if websocket hub is not available, skip websocket notification
	if s.wsHub == nil {
		log.Printf("[WebSocket] Hub not configured, skipping websocket notification for order %s", order.OrderID.String())
		// Still send notification service if available
		message := fmt.Sprintf("Nuevo pedido #%s disponible", order.OrderID.String()[:8])
		if s.notificationService != nil {
			s.notificationService.SendToRepartidores(message, order.OrderID.String())
		}
		return
	}

	// Log para depuración
	log.Printf("[WebSocket] Ejecutando notifyNewOrder para pedido %s", order.OrderID.String())
	
	// Notificar a los repartidores sobre un nuevo pedido
	message := fmt.Sprintf("Nuevo pedido #%s disponible", order.OrderID.String()[:8])
	if s.notificationService != nil {
		s.notificationService.SendToRepartidores(message, order.OrderID.String())
	}
	// WebSocket: enviar a todos los repartidores
	type NewOrderPayload struct {
		OrderID    string  `json:"order_id"`
		Status     string  `json:"status"`
		ClientID   string  `json:"client_id"`
		ClientName string  `json:"client_name"`
		Address    string  `json:"address"`
		Total      float64 `json:"total_amount"`
		OrderTime  string  `json:"order_time"`
	}
	clientName := ""
	if order.Client.FullName != "" {
		clientName = order.Client.FullName
	}
	payload := NewOrderPayload{
		OrderID:    order.OrderID.String(),
		Status:     string(order.OrderStatus),
		ClientID:   order.ClientID.String(),
		ClientName: clientName,
		Address:    order.DeliveryAddressText,
		Total:      order.TotalAmount,
		OrderTime:  order.OrderTime.Format(time.RFC3339),
	}
	msg := ws.Message{
		Type:    ws.NewOrderAvailable,
		Payload: ws.MustMarshalPayload(payload),
	}
	log.Printf("[WebSocket] Enviando mensaje de nuevo pedido a rol REPARTIDOR: %+v", payload)
	s.wsHub.SendToRole("REPARTIDOR", msg)
	log.Printf("[WebSocket] También enviando a ADMIN")
	s.wsHub.SendToRole("ADMIN", msg)
	log.Printf("[WebSocket] Mensajes enviados al hub WebSocket")
}

func (s *OrderService) notifyStatusChange(order *models.Order) {
	var message string
	switch order.OrderStatus {
	case models.OrderStatusConfirmed:
		message = "Tu pedido ha sido confirmado."
	case models.OrderStatusAssigned:
		var repartidorName string
		if order.AssignedRepartidor != nil {
			repartidorName = order.AssignedRepartidor.FullName
		} else {
			repartidorName = "un repartidor"
		}
		message = fmt.Sprintf("Tu pedido ha sido asignado a %s.", repartidorName)
	case models.OrderStatusInTransit:
		message = "Tu pedido está en camino."
	case models.OrderStatusDelivered:
		message = "Tu pedido ha sido entregado."
	case models.OrderStatusCancelled:
		message = "Tu pedido ha sido cancelado."
	default:
		message = "El estado de tu pedido ha sido actualizado."
	}
	if s.notificationService != nil {
		s.notificationService.SendToClient(order.ClientID.String(), message, order.OrderID.String())
	}
	// WebSocket: notificar al cliente y a los repartidores
	type StatusUpdatePayload struct {
		OrderID          string  `json:"order_id"`
		Status           string  `json:"status"`
		Message          string  `json:"message"`
		EstimatedArrival *string `json:"estimated_arrival_time,omitempty"`
	}
	var eta *string
	if order.EstimatedArrivalTime != nil {
		formatted := order.EstimatedArrivalTime.Format(time.RFC3339)
		eta = &formatted
	}
	payload := StatusUpdatePayload{
		OrderID:          order.OrderID.String(),
		Status:           string(order.OrderStatus),
		Message:          message,
		EstimatedArrival: eta,
	}
	// Only send WebSocket messages if hub is available
	if s.wsHub != nil {
		msg := ws.Message{
			Type:    ws.OrderStatusUpdate,
			Payload: ws.MustMarshalPayload(payload),
		}
		s.wsHub.SendToUser(order.ClientID.String(), msg)
		s.wsHub.SendToRole("REPARTIDOR", msg)
		s.wsHub.SendToRole("ADMIN", msg)
	}
}

func (s *OrderService) notifyOrderConfirmed(order *models.Order) {
	message := "Tu pedido ha sido confirmado y será preparado para entrega."
	s.notificationService.SendToClient(order.ClientID.String(), message, order.OrderID.String())
}

func (s *OrderService) notifyOrderAssigned(order *models.Order) {
	var repartidorName string
	if order.AssignedRepartidor != nil {
		repartidorName = order.AssignedRepartidor.FullName
	} else {
		repartidorName = "un repartidor"
	}
	message := fmt.Sprintf("Tu pedido ha sido asignado a %s y pronto iniciará la entrega.", repartidorName)
	if s.notificationService != nil {
		s.notificationService.SendToClient(order.ClientID.String(), message, order.OrderID.String())
	}
	
	// WebSocket: notificar al cliente sobre la asignación
	type StatusUpdatePayload struct {
		OrderID          string  `json:"order_id"`
		Status           string  `json:"status"`
		Message          string  `json:"message"`
		EstimatedArrival *string `json:"estimated_arrival_time,omitempty"`
		RepartidorName   string  `json:"repartidor_name,omitempty"`
	}
	
	var eta *string
	if order.EstimatedArrivalTime != nil {
		formatted := order.EstimatedArrivalTime.Format(time.RFC3339)
		eta = &formatted
	}
	
	payload := StatusUpdatePayload{
		OrderID:          order.OrderID.String(),
		Status:           string(order.OrderStatus),
		Message:          message,
		EstimatedArrival: eta,
		RepartidorName:   repartidorName,
	}
	
	// Only send WebSocket messages if hub is available
	if s.wsHub != nil {
		wsMsg := ws.Message{
			Type:    ws.OrderStatusUpdate,
			Payload: ws.MustMarshalPayload(payload),
		}
		
		log.Printf("[WebSocket] Enviando notificación de asignación al cliente: %s", order.ClientID.String())
		s.wsHub.SendToUser(order.ClientID.String(), wsMsg)
		
		// También notificar a repartidores y admin
		s.wsHub.SendToRole("REPARTIDOR", wsMsg)
		s.wsHub.SendToRole("ADMIN", wsMsg)
	}
}

func (s *OrderService) notifyETA(order *models.Order) {
	if order.EstimatedArrivalTime == nil {
		return
	}

	formattedTime := order.EstimatedArrivalTime.Format("15:04")
	message := fmt.Sprintf("Tu pedido llegará aproximadamente a las %s", formattedTime)
	s.notificationService.SendToClient(order.ClientID.String(), message, order.OrderID.String())
}
