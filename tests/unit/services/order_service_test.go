package services

import (
	"backend/internal/models"
	"backend/tests/testutil"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrder_BusinessHoursLogic(t *testing.T) {
	timezone := "America/Lima"
	businessStart := 6 * time.Hour // 6 AM
	businessEnd := 20 * time.Hour  // 8 PM

	tests := []struct {
		name     string
		hour     int
		expected bool
	}{
		{"Early morning (5 AM)", 5, false},
		{"Opening time (6 AM)", 6, true},
		{"Mid morning (10 AM)", 10, true},
		{"Afternoon (3 PM)", 15, true},
		{"Near closing (7:30 PM)", 19, true},
		{"Closing time (8 PM)", 20, false},
		{"Evening (9 PM)", 21, false},
		{"Late night (11 PM)", 23, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create time in Lima timezone
			lima, err := time.LoadLocation(timezone)
			require.NoError(t, err)

			testTime := time.Date(2024, 6, 15, tt.hour, 0, 0, 0, lima)
			result := models.IsWithinBusinessHours(testTime, businessStart, businessEnd, timezone)

			assert.Equal(t, tt.expected, result,
				"La hora %d:00 debe ser %v para las horas comerciales", tt.hour, tt.expected)
		})
	}
}

func TestOrder_StateTransitions(t *testing.T) {
	testCases := []struct {
		name          string
		currentStatus models.OrderStatus
		newStatus     models.OrderStatus
		shouldSucceed bool
		description   string
	}{
		// Valid transitions
		{
			name:          "PENDING to CONFIRMED",
			currentStatus: models.OrderStatusPending,
			newStatus:     models.OrderStatusConfirmed,
			shouldSucceed: true,
			description:   "El administrador o repartidor pueden confirmar pedidos pendientes",
		},
		{
			name:          "CONFIRMED to ASSIGNED",
			currentStatus: models.OrderStatusConfirmed,
			newStatus:     models.OrderStatusAssigned,
			shouldSucceed: true,
			description:   "El administrador puede asignar un repartidor a pedidos confirmados",
		},
		{
			name:          "ASSIGNED to IN_TRANSIT",
			currentStatus: models.OrderStatusAssigned,
			newStatus:     models.OrderStatusInTransit,
			shouldSucceed: true,
			description:   "El repartidor asignado puede comenzar la entrega",
		},
		{
			name:          "IN_TRANSIT to DELIVERED",
			currentStatus: models.OrderStatusInTransit,
			newStatus:     models.OrderStatusDelivered,
			shouldSucceed: true,
			description:   "El repartidor puede marcar el pedido como entregado",
		},

		// Invalid transitions
		{
			name:          "PENDING to ASSIGNED (skip CONFIRMED)",
			currentStatus: models.OrderStatusPending,
			newStatus:     models.OrderStatusAssigned,
			shouldSucceed: false,
			description:   "No se puede saltar el paso de confirmación",
		},
		{
			name:          "CONFIRMED to IN_TRANSIT (skip ASSIGNED)",
			currentStatus: models.OrderStatusConfirmed,
			newStatus:     models.OrderStatusInTransit,
			shouldSucceed: false,
			description:   "No se puede saltar el paso de asignación",
		},
		{
			name:          "DELIVERED to PENDING (final state)",
			currentStatus: models.OrderStatusDelivered,
			newStatus:     models.OrderStatusPending,
			shouldSucceed: false,
			description:   "No se puede cambiar el estado final",
		},
		{
			name:          "IN_TRANSIT to CANCELLED",
			currentStatus: models.OrderStatusInTransit,
			newStatus:     models.OrderStatusCancelled,
			shouldSucceed: false,
			description:   "No se puede cancelar pedidos en tránsito",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			order := &models.Order{
				OrderStatus: tc.currentStatus,
			}

			result := order.CanTransitionTo(tc.newStatus)
			assert.Equal(t, tc.shouldSucceed, result, tc.description)
		})
	}
}

func TestOrder_CreationWithTimestamps(t *testing.T) {
	clientID := uuid.New()

	// Test order creation with current time
	order := testutil.CreateTestOrder(t, clientID)

	// Verify basic fields
	assert.NotEqual(t, uuid.Nil, order.OrderID)
	assert.Equal(t, clientID, order.ClientID)
	assert.Equal(t, models.OrderStatusPending, order.OrderStatus)
	assert.Equal(t, 45.50, order.TotalAmount)

	// Verify location data
	assert.Equal(t, -12.046374, order.Latitude)
	assert.Equal(t, -77.042793, order.Longitude)
	assert.Equal(t, "Av. Test 123, Lima, Perú", order.DeliveryAddressText)

	// Verify timestamps
	assert.False(t, order.OrderTime.IsZero())
	assert.False(t, order.CreatedAt.IsZero())
	assert.False(t, order.UpdatedAt.IsZero())

	// Verify optional fields are nil initially
	assert.Nil(t, order.ConfirmedAt)
	assert.Nil(t, order.AssignedAt)
	assert.Nil(t, order.EstimatedArrivalTime)
	assert.Nil(t, order.DeliveredAt)
	assert.Nil(t, order.CancelledAt)
	assert.Nil(t, order.AssignedRepartidorID)
}

func TestOrderItem_SubtotalCalculation(t *testing.T) {
	tests := []struct {
		name      string
		quantity  int
		unitPrice float64
		expected  float64
	}{
		{"Single item", 1, 45.50, 45.50},
		{"Multiple items", 3, 45.50, 136.50},
		{"Fractional price", 2, 22.75, 45.50},
		{"Large quantity", 10, 15.00, 150.00},
		{"Zero quantity", 0, 45.50, 0.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderID := uuid.New()
			productID := uuid.New()

			orderItem := &models.OrderItem{
				OrderID:   orderID,
				ProductID: productID,
				Quantity:  tt.quantity,
				UnitPrice: tt.unitPrice,
			}

			// Simulate BeforeCreate hook
			err := orderItem.BeforeCreate(nil)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, orderItem.Subtotal,
				"El cálculo del subtotal debe ser correcto para %d items a %.2f cada uno",
				tt.quantity, tt.unitPrice)
			assert.NotEqual(t, uuid.Nil, orderItem.OrderItemID, "Debe generar un nuevo UUID")
		})
	}
}

func TestUser_PasswordHandling(t *testing.T) {
	user := testutil.CreateTestUser(t, models.UserRoleClient)
	originalPassword := "testpassword123"

	// Test password verification
	assert.True(t, user.CheckPassword(originalPassword), "Debe verificar la contraseña correcta")
	assert.False(t, user.CheckPassword("wrongpassword"), "Debe rechazar la contraseña incorrecta")
	assert.False(t, user.CheckPassword(""), "Debe rechazar la contraseña vacía")

	// Test password change
	newPassword := "newpassword456"
	err := user.SetPassword(newPassword)
	require.NoError(t, err)

	// Old password should no longer work
	assert.False(t, user.CheckPassword(originalPassword), "La contraseña antigua no debe funcionar")

	// New password should work
	assert.True(t, user.CheckPassword(newPassword), "La nueva contraseña debe funcionar")
}

func TestUserRole_Authorization(t *testing.T) {
	roles := []models.UserRole{
		models.UserRoleClient,
		models.UserRoleRepartidor,
		models.UserRoleAdmin,
	}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			user := testutil.CreateTestUser(t, role)
			assert.Equal(t, role, user.UserRole)

			// Test role-specific properties
			switch role {
			case models.UserRoleClient:
				// Clients should be able to create orders
				assert.Contains(t, []models.UserRole{
					models.UserRoleClient,
					models.UserRoleRepartidor,
					models.UserRoleAdmin,
				}, user.UserRole)

			case models.UserRoleRepartidor:
				// Repartidores should be able to handle orders
				assert.Contains(t, []models.UserRole{
					models.UserRoleRepartidor,
					models.UserRoleAdmin,
				}, user.UserRole)

			case models.UserRoleAdmin:
				// Admins should have full access
				assert.Equal(t, models.UserRoleAdmin, user.UserRole)
			}
		})
	}
}

func TestOrderStatus_WorkflowValidation(t *testing.T) {
	// Test complete valid workflow
	order := &models.Order{OrderStatus: models.OrderStatusPending}

	// Step 1: PENDING -> CONFIRMED
	assert.True(t, order.CanTransitionTo(models.OrderStatusConfirmed))
	order.OrderStatus = models.OrderStatusConfirmed

	// Step 2: CONFIRMED -> ASSIGNED
	assert.True(t, order.CanTransitionTo(models.OrderStatusAssigned))
	order.OrderStatus = models.OrderStatusAssigned

	// Step 3: ASSIGNED -> IN_TRANSIT
	assert.True(t, order.CanTransitionTo(models.OrderStatusInTransit))
	order.OrderStatus = models.OrderStatusInTransit

	// Step 4: IN_TRANSIT -> DELIVERED
	assert.True(t, order.CanTransitionTo(models.OrderStatusDelivered))
	order.OrderStatus = models.OrderStatusDelivered

	// Step 5: DELIVERED (final) -> no more transitions
	assert.False(t, order.CanTransitionTo(models.OrderStatusPending))
	assert.False(t, order.CanTransitionTo(models.OrderStatusConfirmed))
	assert.False(t, order.CanTransitionTo(models.OrderStatusCancelled))
}

func TestOrderStatus_CancellationRules(t *testing.T) {
	cancellableStates := []models.OrderStatus{
		models.OrderStatusPending,
		models.OrderStatusPendingOutOfHours,
		models.OrderStatusConfirmed,
	}

	nonCancellableStates := []models.OrderStatus{
		models.OrderStatusAssigned,
		models.OrderStatusInTransit,
		models.OrderStatusDelivered,
		models.OrderStatusCancelled,
	}

	// Test cancellable states
	for _, status := range cancellableStates {
		t.Run("Cancellable_"+string(status), func(t *testing.T) {
			order := &models.Order{OrderStatus: status}
			assert.True(t, order.CanTransitionTo(models.OrderStatusCancelled),
				"Debe ser capaz de cancelar el pedido en el estado %s", status)
		})
	}

	// Test non-cancellable states
	for _, status := range nonCancellableStates {
		t.Run("NonCancellable_"+string(status), func(t *testing.T) {
			order := &models.Order{OrderStatus: status}
			assert.False(t, order.CanTransitionTo(models.OrderStatusCancelled),
				"No debe ser capaz de cancelar el pedido en el estado %s", status)
		})
	}
}
