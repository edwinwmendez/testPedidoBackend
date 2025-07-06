package handlers

import (
	"strconv"
	"time"

	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// OrderHandler maneja las peticiones HTTP relacionadas con pedidos
type OrderHandler struct {
	orderService *services.OrderService
	authService  auth.Service
}

// NewOrderHandler crea un nuevo handler de pedidos
func NewOrderHandler(orderService *services.OrderService, authService auth.Service) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		authService:  authService,
	}
}

// CreateOrderRequest estructura para la creación de un pedido
type CreateOrderRequest struct {
	Items               []OrderItemRequest `json:"items" validate:"required,dive"`
	Latitude            float64            `json:"latitude" validate:"required"`
	Longitude           float64            `json:"longitude" validate:"required"`
	DeliveryAddressText string             `json:"delivery_address_text" validate:"required"`
	PaymentNote         string             `json:"payment_note"`
}

// OrderItemRequest estructura para los ítems de un pedido
type OrderItemRequest struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
}

// UpdateOrderStatusRequest estructura para actualizar el estado de un pedido
type UpdateOrderStatusRequest struct {
	NewStatus string `json:"new_status" validate:"required,oneof=PENDING PENDING_OUT_OF_HOURS CONFIRMED IN_TRANSIT DELIVERED CANCELLED"`
}

// AssignRepartidorRequest estructura para asignar un repartidor a un pedido
type AssignRepartidorRequest struct {
	RepartidorID string `json:"repartidor_id" validate:"omitempty,uuid"`
}

// SetETARequest estructura para establecer el tiempo estimado de llegada
type SetETARequest struct {
	EstimatedArrivalTime string `json:"estimated_arrival_time" validate:"required"`
}

// @Summary Crear un nuevo pedido
// @Description Crea un nuevo pedido para un cliente
// @Tags pedidos
// @Accept json
// @Produce json
// @Param order body CreateOrderRequest true "Datos del pedido"
// @Success 201 {object} models.Order
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /orders [post]
// CreateOrder crea un nuevo pedido
func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	// Obtener el usuario autenticado del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Validar que el usuario sea un cliente
	if claims.UserRole != models.UserRoleClient {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Solo los clientes pueden crear pedidos",
		})
	}

	// Parsear el cuerpo de la petición
	var req CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos de pedido inválidos",
		})
	}

	// Validar los datos
	if len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El pedido debe contener al menos un producto",
		})
	}

	// Crear el pedido en el modelo
	order := &models.Order{
		ClientID:            uuid.MustParse(claims.UserID.String()),
		Latitude:            req.Latitude,
		Longitude:           req.Longitude,
		DeliveryAddressText: req.DeliveryAddressText,
		PaymentNote:         req.PaymentNote,
		OrderTime:           time.Now(),
	}

	// Convertir los items de la petición al modelo
	var orderItems []models.OrderItem
	for _, item := range req.Items {
		orderItems = append(orderItems, models.OrderItem{
			ProductID: uuid.MustParse(item.ProductID),
			Quantity:  item.Quantity,
		})
	}

	// Crear el pedido usando el servicio
	createdOrder, err := h.orderService.CreateOrder(order, orderItems)
	if err != nil {
		// Manejar errores específicos
		switch err {
		case services.ErrOutsideBusinessHours:
			// Esto no es realmente un error, solo un estado especial
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"message": "Pedido recibido fuera del horario de atención. Será procesado al inicio del próximo turno.",
				"order":   createdOrder,
			})
		case services.ErrProductNotFound:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Uno o más productos no existen",
			})
		case services.ErrProductInactive:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Uno o más productos no están disponibles",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al crear el pedido",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(createdOrder)
}

// @Summary Obtener la lista de pedidos según el rol del usuario
// @Description Obtiene la lista de pedidos según el rol del usuario
// @Tags pedidos
// @Accept json
// @Produce json
// @Param status query string false "Filtro por estado del pedido"
// @Success 200 {array} models.Order
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /orders [get]
// GetOrders obtiene la lista de pedidos según el rol del usuario
func (h *OrderHandler) GetOrders(c *fiber.Ctx) error {
	// Obtener el usuario autenticado del contexto
	claims := c.Locals("user").(*auth.Claims)

	var orders []*models.Order
	var err error

	// Obtener parámetros de consulta opcionales
	status := c.Query("status")

	// Filtrar por estado si se proporciona
	if status != "" {
		orders, err = h.orderService.GetOrdersByStatus(models.OrderStatus(status))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al obtener los pedidos",
			})
		}
	} else {
		// Comportamiento según el rol del usuario
		switch claims.UserRole {
		case models.UserRoleClient:
			// Los clientes solo ven sus propios pedidos
			orders, err = h.orderService.GetOrdersByClientID(claims.UserID.String())
		case models.UserRoleRepartidor:
			// Los repartidores ven los pedidos asignados a ellos y los pendientes
			assignedOrders, err := h.orderService.GetOrdersByRepartidorID(claims.UserID.String())
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error al obtener los pedidos",
				})
			}

			pendingOrders, err := h.orderService.GetPendingOrders()
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error al obtener los pedidos",
				})
			}

			// Combinar los pedidos asignados y pendientes
			orders = append(assignedOrders, pendingOrders...)
		case models.UserRoleAdmin:
			// Los administradores pueden ver todos los pedidos
			orders, err = h.orderService.GetAllOrders()
		}

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al obtener los pedidos",
			})
		}
	}

	return c.JSON(orders)
}

// @Summary Obtener un pedido por su ID
// @Description Obtiene los detalles de un pedido específico por su ID
// @Tags pedidos
// @Accept json
// @Produce json
// @Param id path string true "ID del pedido"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /orders/{id} [get]
// GetOrderByID obtiene un pedido por su ID
func (h *OrderHandler) GetOrderByID(c *fiber.Ctx) error {
	// Obtener el usuario autenticado del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Obtener el ID del pedido de los parámetros
	orderID := c.Params("id")
	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pedido requerido",
		})
	}

	// Obtener el pedido
	order, err := h.orderService.GetOrderByID(orderID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pedido no encontrado",
		})
	}

	// Verificar permisos según el rol
	switch claims.UserRole {
	case models.UserRoleClient:
		// Los clientes solo pueden ver sus propios pedidos
		if order.ClientID.String() != claims.UserID.String() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No tienes permiso para ver este pedido",
			})
		}
	case models.UserRoleRepartidor:
		// Los repartidores pueden ver pedidos asignados a ellos o pendientes
		if order.AssignedRepartidorID != nil &&
			order.AssignedRepartidorID.String() != claims.UserID.String() &&
			order.OrderStatus != models.OrderStatusPending &&
			order.OrderStatus != models.OrderStatusPendingOutOfHours {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No tienes permiso para ver este pedido",
			})
		}
	case models.UserRoleAdmin:
		// Los administradores pueden ver cualquier pedido
	}

	return c.JSON(order)
}

// @Summary Obtener información del repartidor de un pedido
// @Description Obtiene la información del repartidor asignado a un pedido específico según permisos del usuario
// @Tags pedidos
// @Accept json
// @Produce json
// @Param id path string true "ID del pedido"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /orders/{id}/repartidor [get]
// GetOrderRepartidor obtiene la información del repartidor asignado a un pedido
func (h *OrderHandler) GetOrderRepartidor(c *fiber.Ctx) error {
	// Obtener el usuario autenticado del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Obtener el ID del pedido de los parámetros
	orderID := c.Params("id")
	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pedido requerido",
		})
	}

	// Obtener el pedido
	order, err := h.orderService.GetOrderByID(orderID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pedido no encontrado",
		})
	}

	// Verificar que el pedido tenga un repartidor asignado
	if order.AssignedRepartidor == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No hay repartidor asignado a este pedido",
		})
	}

	// Verificar permisos según el rol
	switch claims.UserRole {
	case models.UserRoleClient:
		// Los clientes solo pueden ver el repartidor de sus propios pedidos
		if order.ClientID.String() != claims.UserID.String() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No tienes permiso para ver esta información",
			})
		}
	case models.UserRoleRepartidor:
		// Los repartidores pueden ver información del repartidor si:
		// 1. Es información sobre ellos mismos, O
		// 2. Es un pedido que pueden ver (asignado a ellos o pendiente)
		canView := false
		
		// Si es información sobre ellos mismos
		if order.AssignedRepartidor.UserID.String() == claims.UserID.String() {
			canView = true
		}
		
		// Si es un pedido que pueden ver según lógica normal
		if order.AssignedRepartidorID != nil &&
			order.AssignedRepartidorID.String() == claims.UserID.String() {
			canView = true
		}
		
		// Si es un pedido pendiente (pueden ver todos los repartidores)
		if order.OrderStatus == models.OrderStatusPending || 
		   order.OrderStatus == models.OrderStatusPendingOutOfHours {
			canView = true
		}
		
		if !canView {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No tienes permiso para ver esta información",
			})
		}
	case models.UserRoleAdmin:
		// Los administradores pueden ver cualquier información
	}

	// Retornar solo la información del repartidor (sin datos sensibles como password)
	repartidorInfo := fiber.Map{
		"user_id":      order.AssignedRepartidor.UserID,
		"full_name":    order.AssignedRepartidor.FullName,
		"phone_number": order.AssignedRepartidor.PhoneNumber,
		"email":        order.AssignedRepartidor.Email,
		"user_role":    order.AssignedRepartidor.UserRole,
		"is_active":    order.AssignedRepartidor.IsActive,
		"created_at":   order.AssignedRepartidor.CreatedAt,
	}

	return c.JSON(repartidorInfo)
}

// @Summary Actualizar el estado de un pedido
// @Description Actualiza el estado de un pedido según el rol del usuario
// @Tags pedidos
// @Accept json
// @Produce json
// @Param id path string true "ID del pedido"
// @Param status body UpdateOrderStatusRequest true "Nuevo estado del pedido"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /orders/{id}/status [put]
// UpdateOrderStatus actualiza el estado de un pedido
func (h *OrderHandler) UpdateOrderStatus(c *fiber.Ctx) error {
	// Obtener el usuario autenticado del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Obtener el ID del pedido de los parámetros
	orderID := c.Params("id")
	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pedido requerido",
		})
	}

	// Parsear el cuerpo de la petición
	var req UpdateOrderStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Actualizar el estado del pedido
	updatedOrder, err := h.orderService.UpdateOrderStatus(
		orderID,
		models.OrderStatus(req.NewStatus),
		claims.UserID.String(),
		claims.UserRole,
	)

	if err != nil {
		switch err {
		case services.ErrOrderNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Pedido no encontrado",
			})
		case services.ErrInvalidTransition:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Transición de estado inválida",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al actualizar el estado del pedido",
			})
		}
	}

	return c.JSON(updatedOrder)
}

// @Summary Asignar un repartidor a un pedido
// @Description Asigna un repartidor a un pedido según el rol del usuario
// @Tags pedidos
// @Accept json
// @Produce json
// @Param id path string true "ID del pedido"
// @Param repartidor body AssignRepartidorRequest true "ID del repartidor a asignar"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /orders/{id}/assign [post]
// AssignRepartidor asigna un repartidor a un pedido
func (h *OrderHandler) AssignRepartidor(c *fiber.Ctx) error {
	// Obtener el usuario autenticado del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Obtener el ID del pedido de los parámetros
	orderID := c.Params("id")
	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pedido requerido",
		})
	}

	// Parsear el cuerpo de la petición
	var req AssignRepartidorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Determinar el ID del repartidor a asignar
	repartidorID := req.RepartidorID
	if repartidorID == "" && claims.UserRole == models.UserRoleRepartidor {
		// Si no se proporciona un ID y el usuario es un repartidor, asignar al usuario actual
		repartidorID = claims.UserID.String()
	}

	// Verificar permisos
	if claims.UserRole == models.UserRoleClient {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Los clientes no pueden asignar repartidores",
		})
	}

	if claims.UserRole == models.UserRoleRepartidor && repartidorID != claims.UserID.String() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Los repartidores solo pueden asignarse pedidos a sí mismos",
		})
	}

	// Asignar el repartidor
	updatedOrder, err := h.orderService.AssignRepartidor(orderID, repartidorID)
	if err != nil {
		switch err {
		case services.ErrOrderNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Pedido no encontrado",
			})
		case services.ErrInvalidOrderStatus:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "El pedido no está en un estado que permita asignación",
			})
		case services.ErrOrderAlreadyAssigned:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "El pedido ya está asignado a un repartidor",
			})
		case services.ErrUserNotFound:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Repartidor no encontrado",
			})
		case services.ErrInvalidRole:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "El usuario asignado debe ser un repartidor o administrador",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al asignar el repartidor",
			})
		}
	}

	return c.JSON(updatedOrder)
}

// @Summary Establecer el tiempo estimado de llegada para un pedido
// @Description Establece el tiempo estimado de llegada para un pedido según el rol del usuario
// @Tags pedidos
// @Accept json
// @Produce json
// @Param id path string true "ID del pedido"
// @Param eta body SetETARequest true "Tiempo estimado de llegada"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /orders/{id}/eta [put]
// SetEstimatedArrivalTime establece el tiempo estimado de llegada para un pedido
func (h *OrderHandler) SetEstimatedArrivalTime(c *fiber.Ctx) error {
	// Obtener el usuario autenticado del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Verificar que el usuario sea un repartidor o administrador
	if claims.UserRole != models.UserRoleRepartidor && claims.UserRole != models.UserRoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Solo los repartidores y administradores pueden establecer tiempos estimados de llegada",
		})
	}

	// Obtener el ID del pedido de los parámetros
	orderID := c.Params("id")
	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pedido requerido",
		})
	}

	// Parsear el cuerpo de la petición
	var req SetETARequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Parsear el tiempo estimado de llegada
	eta, err := time.Parse(time.RFC3339, req.EstimatedArrivalTime)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de tiempo inválido. Use formato ISO 8601 (YYYY-MM-DDTHH:MM:SSZ)",
		})
	}

	// Verificar que el tiempo estimado sea en el futuro
	if eta.Before(time.Now()) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El tiempo estimado de llegada debe ser en el futuro",
		})
	}

	// Establecer el tiempo estimado de llegada
	updatedOrder, err := h.orderService.SetEstimatedArrivalTime(orderID, eta)
	if err != nil {
		switch err {
		case services.ErrOrderNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Pedido no encontrado",
			})
		case services.ErrInvalidOrderStatus:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "El pedido no está en un estado que permita establecer tiempo estimado de llegada",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al establecer el tiempo estimado de llegada",
			})
		}
	}

	return c.JSON(updatedOrder)
}

// @Summary Buscar pedidos cercanos a una ubicación
// @Description Busca pedidos cercanos a una ubicación según el rol del usuario
// @Tags pedidos
// @Accept json
// @Produce json
// @Param lat query string true "Latitud"
// @Param lng query string true "Longitud"
// @Param radius query string false "Radio de búsqueda (en km)"
// @Success 200 {array} models.Order
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /orders/nearby [get]
// FindNearbyOrders encuentra pedidos cercanos a una ubicación
func (h *OrderHandler) FindNearbyOrders(c *fiber.Ctx) error {
	// Obtener el usuario autenticado del contexto
	claims := c.Locals("user").(*auth.Claims)

	// Verificar que el usuario sea un repartidor o administrador
	if claims.UserRole != models.UserRoleRepartidor && claims.UserRole != models.UserRoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Solo los repartidores y administradores pueden buscar pedidos cercanos",
		})
	}

	// Obtener parámetros de consulta
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	radiusStr := c.Query("radius", "5") // Radio predeterminado: 5 km

	// Validar y convertir parámetros
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Latitud inválida",
		})
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Longitud inválida",
		})
	}

	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil || radius <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Radio inválido",
		})
	}

	// Buscar pedidos cercanos
	orders, err := h.orderService.FindNearbyOrders(lat, lng, radius)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al buscar pedidos cercanos",
		})
	}

	return c.JSON(orders)
}

// RegisterRoutes registra las rutas del handler en el router
func (h *OrderHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler, adminOnly fiber.Handler, repartidorOrAdmin fiber.Handler) {
	orders := router.Group("/orders", authMiddleware)

	// Rutas para clientes
	orders.Post("/", h.CreateOrder)                // Crear un nuevo pedido (solo clientes)
	orders.Get("/", h.GetOrders)                   // Obtener pedidos (filtrado según rol)
	orders.Get("/:id", h.GetOrderByID)             // Obtener un pedido específico (según permisos)
	orders.Put("/:id/status", h.UpdateOrderStatus) // Actualizar estado (según permisos)

	// Rutas para repartidores y administradores
	orders.Post("/:id/assign", repartidorOrAdmin, h.AssignRepartidor)    // Asignar repartidor
	orders.Put("/:id/eta", repartidorOrAdmin, h.SetEstimatedArrivalTime) // Establecer ETA
	orders.Get("/nearby", repartidorOrAdmin, h.FindNearbyOrders)         // Buscar pedidos cercanos
	orders.Get("/:id/repartidor", h.GetOrderRepartidor)                  // Obtener info del repartidor del pedido
}
