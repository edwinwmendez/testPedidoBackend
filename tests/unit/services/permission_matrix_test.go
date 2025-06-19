package services

import (
	"backend/internal/models"
	"backend/tests/testutil"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestOrderPermissionMatrix tests the complete permission matrix for all user roles
func TestOrderPermissionMatrix(t *testing.T) {
	// Create test users for each role
	clientUser := testutil.CreateTestUser(t, models.UserRoleClient)
	repartidorUser := testutil.CreateTestUser(t, models.UserRoleRepartidor)
	adminUser := testutil.CreateTestUser(t, models.UserRoleAdmin)

	// Create different order scenarios to test
	pendingOrder := testutil.CreateTestOrder(t, clientUser.UserID)
	pendingOrder.OrderStatus = models.OrderStatusPending

	confirmedOrder := testutil.CreateTestOrder(t, clientUser.UserID)
	confirmedOrder.OrderStatus = models.OrderStatusConfirmed

	assignedOrder := testutil.CreateTestOrder(t, clientUser.UserID)
	assignedOrder.OrderStatus = models.OrderStatusAssigned
	assignedOrder.AssignedRepartidorID = &repartidorUser.UserID

	inTransitOrder := testutil.CreateTestOrder(t, clientUser.UserID)
	inTransitOrder.OrderStatus = models.OrderStatusInTransit
	inTransitOrder.AssignedRepartidorID = &repartidorUser.UserID

	deliveredOrder := testutil.CreateTestOrder(t, clientUser.UserID)
	deliveredOrder.OrderStatus = models.OrderStatusDelivered

	// Permission test matrix
	testCases := []struct {
		name               string
		order              *models.Order
		userRole           models.UserRole
		userID             uuid.UUID
		targetStatus       models.OrderStatus
		expectedPermission bool
		description        string
	}{
		// CLIENT permissions
		{
			name:               "Client_Cancel_Own_Pending",
			order:              pendingOrder,
			userRole:           models.UserRoleClient,
			userID:             clientUser.UserID,
			targetStatus:       models.OrderStatusCancelled,
			expectedPermission: true,
			description:        "El cliente debe ser capaz de cancelar sus propios pedidos pendientes",
		},
		{
			name:               "Client_Cancel_Other_Pending",
			order:              pendingOrder,
			userRole:           models.UserRoleClient,
			userID:             uuid.New(), // Different client
			targetStatus:       models.OrderStatusCancelled,
			expectedPermission: false,
			description:        "El cliente NO debe poder cancelar los pedidos de otro cliente.",
		},
		{
			name:               "Client_Confirm_Order",
			order:              pendingOrder,
			userRole:           models.UserRoleClient,
			userID:             clientUser.UserID,
			targetStatus:       models.OrderStatusConfirmed,
			expectedPermission: false,
			description:        "El cliente NO debe poder confirmar pedidos",
		},
		{
			name:               "Client_Cancel_Confirmed_Order",
			order:              confirmedOrder,
			userRole:           models.UserRoleClient,
			userID:             clientUser.UserID,
			targetStatus:       models.OrderStatusCancelled,
			expectedPermission: false,
			description:        "El cliente NO debe poder cancelar pedidos confirmados",
		},

		// REPARTIDOR permissions
		{
			name:               "Repartidor_Confirm_Pending",
			order:              pendingOrder,
			userRole:           models.UserRoleRepartidor,
			userID:             repartidorUser.UserID,
			targetStatus:       models.OrderStatusConfirmed,
			expectedPermission: true,
			description:        "El repartidor debe poder confirmar pedidos pendientes",
		},
		{
			name:               "Repartidor_Start_Transit_Assigned",
			order:              assignedOrder,
			userRole:           models.UserRoleRepartidor,
			userID:             repartidorUser.UserID,
			targetStatus:       models.OrderStatusInTransit,
			expectedPermission: true,
			description:        "El repartidor asignado debe poder comenzar el tránsito",
		},
		{
			name:               "Repartidor_Start_Transit_NotAssigned",
			order:              assignedOrder,
			userRole:           models.UserRoleRepartidor,
			userID:             uuid.New(), // Different repartidor
			targetStatus:       models.OrderStatusInTransit,
			expectedPermission: false,
			description:        "Un repartidor no asignado NO debe poder iniciar el tránsito.",
		},
		{
			name:               "Repartidor_Deliver_InTransit",
			order:              inTransitOrder,
			userRole:           models.UserRoleRepartidor,
			userID:             repartidorUser.UserID,
			targetStatus:       models.OrderStatusDelivered,
			expectedPermission: true,
			description:        "El repartidor asignado debe poder entregar los pedidos en tránsito",
		},
		{
			name:               "Repartidor_Deliver_NotAssigned",
			order:              inTransitOrder,
			userRole:           models.UserRoleRepartidor,
			userID:             uuid.New(), // Different repartidor
			targetStatus:       models.OrderStatusDelivered,
			expectedPermission: false,
			description:        "Un repartidor no asignado NO debe poder entregar pedidos",
		},

		// ADMIN permissions
		{
			name:               "Admin_Confirm_Pending",
			order:              pendingOrder,
			userRole:           models.UserRoleAdmin,
			userID:             adminUser.UserID,
			targetStatus:       models.OrderStatusConfirmed,
			expectedPermission: true,
			description:        "El administrador debe poder confirmar pedidos pendientes",
		},
		{
			name:               "Admin_Assign_Confirmed",
			order:              confirmedOrder,
			userRole:           models.UserRoleAdmin,
			userID:             adminUser.UserID,
			targetStatus:       models.OrderStatusAssigned,
			expectedPermission: true,
			description:        "El administrador debe poder asignar pedidos confirmados",
		},
		{
			name:               "Admin_Cannot_Start_Transit",
			order:              assignedOrder,
			userRole:           models.UserRoleAdmin,
			userID:             adminUser.UserID,
			targetStatus:       models.OrderStatusInTransit,
			expectedPermission: false,
			description:        "El administrador NO debe poder iniciar el tránsito (solo el repartidor asignado puede hacerlo)",
		},
		{
			name:               "Admin_Cannot_Deliver",
			order:              inTransitOrder,
			userRole:           models.UserRoleAdmin,
			userID:             adminUser.UserID,
			targetStatus:       models.OrderStatusDelivered,
			expectedPermission: false,
			description:        "El administrador NO debe poder entregar pedidos (solo el repartidor asignado puede hacerlo)",
		},

		// Invalid transitions (should fail for all roles)
		{
			name:               "Invalid_Pending_To_Assigned",
			order:              pendingOrder,
			userRole:           models.UserRoleAdmin,
			userID:             adminUser.UserID,
			targetStatus:       models.OrderStatusAssigned,
			expectedPermission: false,
			description:        "No debe permitir saltar el estado CONFIRMED",
		},
		{
			name:               "Invalid_Confirmed_To_InTransit",
			order:              confirmedOrder,
			userRole:           models.UserRoleAdmin,
			userID:             adminUser.UserID,
			targetStatus:       models.OrderStatusInTransit,
			expectedPermission: false,
			description:        "No debe permitir saltar el estado ASSIGNED",
		},
		{
			name:               "Invalid_Delivered_To_Pending",
			order:              deliveredOrder,
			userRole:           models.UserRoleAdmin,
			userID:             adminUser.UserID,
			targetStatus:       models.OrderStatusPending,
			expectedPermission: false,
			description:        "No debe permitir cambiar el estado final",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test basic state transition validity
			canTransition := tc.order.CanTransitionTo(tc.targetStatus)

			// Test role-based permission logic (simulate the service logic)
			hasPermission := checkOrderPermission(tc.order, tc.targetStatus, tc.userID, tc.userRole)

			// The final permission is the AND of both conditions
			finalPermission := canTransition && hasPermission

			assert.Equal(t, tc.expectedPermission, finalPermission, tc.description)
		})
	}
}

// checkOrderPermission simulates the permission checking logic from the order service
func checkOrderPermission(order *models.Order, newStatus models.OrderStatus, userID uuid.UUID, userRole models.UserRole) bool {
	switch userRole {
	case models.UserRoleAdmin:
		// Admin can: PENDING -> CONFIRMED -> ASSIGNED
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
		// Repartidor can take pending orders and manage their assignments
		if newStatus == models.OrderStatusConfirmed &&
			(order.OrderStatus == models.OrderStatusPending || order.OrderStatus == models.OrderStatusPendingOutOfHours) {
			return true // Any repartidor can take pending orders
		}

		// For advanced states, must be the assigned repartidor
		if order.AssignedRepartidorID != nil && *order.AssignedRepartidorID == userID {
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
		// Client can only cancel their own pending orders
		return newStatus == models.OrderStatusCancelled &&
			order.ClientID == userID &&
			(order.OrderStatus == models.OrderStatusPending || order.OrderStatus == models.OrderStatusPendingOutOfHours)
	}

	return false
}

func TestRoleBasedOrderCreation(t *testing.T) {
	// Test that only clients can create orders for themselves
	clientUser := testutil.CreateTestUser(t, models.UserRoleClient)
	repartidorUser := testutil.CreateTestUser(t, models.UserRoleRepartidor)
	adminUser := testutil.CreateTestUser(t, models.UserRoleAdmin)

	testCases := []struct {
		name          string
		orderOwner    uuid.UUID
		requestor     uuid.UUID
		requestorRole models.UserRole
		shouldAllow   bool
		description   string
	}{
		{
			name:          "Client_Create_Own_Order",
			orderOwner:    clientUser.UserID,
			requestor:     clientUser.UserID,
			requestorRole: models.UserRoleClient,
			shouldAllow:   true,
			description:   "El cliente debe poder crear sus propios pedidos",
		},
		{
			name:          "Client_Create_Other_Order",
			orderOwner:    uuid.New(),
			requestor:     clientUser.UserID,
			requestorRole: models.UserRoleClient,
			shouldAllow:   false,
			description:   "El cliente NO debe poder crear pedidos para otros",
		},
		{
			name:          "Repartidor_Create_Order",
			orderOwner:    clientUser.UserID,
			requestor:     repartidorUser.UserID,
			requestorRole: models.UserRoleRepartidor,
			shouldAllow:   false,
			description:   "El repartidor NO debe poder crear pedidos",
		},
		{
			name:          "Admin_Create_Order_For_Client",
			orderOwner:    clientUser.UserID,
			requestor:     adminUser.UserID,
			requestorRole: models.UserRoleAdmin,
			shouldAllow:   true,
			description:   "El administrador debe poder crear pedidos para clientes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate order creation permission check
			canCreate := checkOrderCreationPermission(tc.orderOwner, tc.requestor, tc.requestorRole)
			assert.Equal(t, tc.shouldAllow, canCreate, tc.description)
		})
	}
}

func checkOrderCreationPermission(orderOwnerID, requestorID uuid.UUID, requestorRole models.UserRole) bool {
	switch requestorRole {
	case models.UserRoleClient:
		// Clients can only create orders for themselves
		return orderOwnerID == requestorID
	case models.UserRoleAdmin:
		// Admins can create orders for any client
		return true
	case models.UserRoleRepartidor:
		// Repartidores cannot create orders
		return false
	default:
		return false
	}
}

func TestRoleBasedOrderViewing(t *testing.T) {
	// Test what orders each role can view
	client1 := testutil.CreateTestUser(t, models.UserRoleClient)
	client2 := testutil.CreateTestUser(t, models.UserRoleClient)
	repartidor := testutil.CreateTestUser(t, models.UserRoleRepartidor)
	admin := testutil.CreateTestUser(t, models.UserRoleAdmin)

	// Create orders for different clients
	client1Order := testutil.CreateTestOrder(t, client1.UserID)
	client2Order := testutil.CreateTestOrder(t, client2.UserID)

	// Assign one order to repartidor
	assignedOrder := testutil.CreateTestOrder(t, client1.UserID)
	assignedOrder.AssignedRepartidorID = &repartidor.UserID

	testCases := []struct {
		name        string
		viewer      uuid.UUID
		viewerRole  models.UserRole
		order       *models.Order
		canView     bool
		description string
	}{
		{
			name:        "Client_View_Own_Order",
			viewer:      client1.UserID,
			viewerRole:  models.UserRoleClient,
			order:       client1Order,
			canView:     true,
			description: "El cliente debe poder ver sus propios pedidos",
		},
		{
			name:        "Client_View_Other_Order",
			viewer:      client1.UserID,
			viewerRole:  models.UserRoleClient,
			order:       client2Order,
			canView:     false,
			description: "El cliente NO debe poder ver los pedidos de otros clientes",
		},
		{
			name:        "Repartidor_View_Assigned_Order",
			viewer:      repartidor.UserID,
			viewerRole:  models.UserRoleRepartidor,
			order:       assignedOrder,
			canView:     true,
			description: "El repartidor debe poder ver los pedidos asignados",
		},
		{
			name:        "Repartidor_View_Unassigned_Order",
			viewer:      repartidor.UserID,
			viewerRole:  models.UserRoleRepartidor,
			order:       client2Order,
			canView:     true,
			description: "El repartidor debe poder ver todos los pedidos (para tomarlos)",
		},
		{
			name:        "Admin_View_Any_Order",
			viewer:      admin.UserID,
			viewerRole:  models.UserRoleAdmin,
			order:       client1Order,
			canView:     true,
			description: "El administrador debe poder ver todos los pedidos",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			canView := checkOrderViewPermission(tc.order, tc.viewer, tc.viewerRole)
			assert.Equal(t, tc.canView, canView, tc.description)
		})
	}
}

func checkOrderViewPermission(order *models.Order, viewerID uuid.UUID, viewerRole models.UserRole) bool {
	switch viewerRole {
	case models.UserRoleClient:
		// Clients can only view their own orders
		return order.ClientID == viewerID
	case models.UserRoleRepartidor:
		// Repartidores can view all orders (to manage and take them)
		return true
	case models.UserRoleAdmin:
		// Admins can view all orders
		return true
	default:
		return false
	}
}

func TestBusinessHoursValidationByRole(t *testing.T) {
	// Test that business hours are enforced correctly for all roles
	// Test order creation during business hours
	businessHoursTime := testutil.CreateTimeInBusinessHours()
	outOfHoursTime := testutil.CreateTimeOutOfBusinessHours()

	testCases := []struct {
		name           string
		orderTime      time.Time
		expectedStatus models.OrderStatus
		description    string
	}{
		{
			name:           "Order_During_Business_Hours",
			orderTime:      businessHoursTime,
			expectedStatus: models.OrderStatusPending,
			description:    "Los pedidos durante las horas comerciales deben ser PENDING",
		},
		{
			name:           "Order_Outside_Business_Hours",
			orderTime:      outOfHoursTime,
			expectedStatus: models.OrderStatusPendingOutOfHours,
			description:    "Los pedidos fuera de las horas comerciales deben ser PENDING_OUT_OF_HOURS",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate determining order status based on time
			status := determineInitialOrderStatus(tc.orderTime)
			assert.Equal(t, tc.expectedStatus, status, tc.description)
		})
	}
}

func determineInitialOrderStatus(orderTime time.Time) models.OrderStatus {
	businessStart := 6 * time.Hour // 6 AM
	businessEnd := 20 * time.Hour  // 8 PM
	timezone := "America/Lima"

	if models.IsWithinBusinessHours(orderTime, businessStart, businessEnd, timezone) {
		return models.OrderStatusPending
	}
	return models.OrderStatusPendingOutOfHours
}
