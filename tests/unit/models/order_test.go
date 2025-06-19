package models

import (
	"backend/internal/models"
	"backend/tests/testutil"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOrder_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name        string
		fromStatus  models.OrderStatus
		toStatus    models.OrderStatus
		expected    bool
		description string
	}{
		// From PENDING
		{
			name:        "pending to confirmed",
			fromStatus:  models.OrderStatusPending,
			toStatus:    models.OrderStatusConfirmed,
			expected:    true,
			description: "Debe permitir que los pedidos pendientes se confirmen",
		},
		{
			name:        "pending to cancelled",
			fromStatus:  models.OrderStatusPending,
			toStatus:    models.OrderStatusCancelled,
			expected:    true,
			description: "Debe permitir que los pedidos pendientes se cancelen",
		},
		{
			name:        "pending to assigned (invalid)",
			fromStatus:  models.OrderStatusPending,
			toStatus:    models.OrderStatusAssigned,
			expected:    false,
			description: "No debe permitir que los pedidos pendientes se asignen a repartidores",
		},

		// From PENDING_OUT_OF_HOURS
		{
			name:        "pending out of hours to confirmed",
			fromStatus:  models.OrderStatusPendingOutOfHours,
			toStatus:    models.OrderStatusConfirmed,
			expected:    true,
			description: "Debe permitir que los pedidos fuera de horario se confirmen",
		},
		{
			name:        "pending out of hours to cancelled",
			fromStatus:  models.OrderStatusPendingOutOfHours,
			toStatus:    models.OrderStatusCancelled,
			expected:    true,
			description: "Debe permitir que los pedidos fuera de horario se cancelen",
		},

		// From CONFIRMED
		{
			name:        "confirmed to assigned",
			fromStatus:  models.OrderStatusConfirmed,
			toStatus:    models.OrderStatusAssigned,
			expected:    true,
			description: "Debe permitir que los pedidos confirmados se asignen a repartidores",
		},
		{
			name:        "confirmed to cancelled",
			fromStatus:  models.OrderStatusConfirmed,
			toStatus:    models.OrderStatusCancelled,
			expected:    true,
			description: "Debe permitir que los pedidos confirmados se cancelen",
		},
		{
			name:        "confirmed to in transit (invalid)",
			fromStatus:  models.OrderStatusConfirmed,
			toStatus:    models.OrderStatusInTransit,
			expected:    false,
			description: "No debe permitir que los pedidos confirmados se salten a en tránsito",
		},

		// From ASSIGNED
		{
			name:        "assigned to in transit",
			fromStatus:  models.OrderStatusAssigned,
			toStatus:    models.OrderStatusInTransit,
			expected:    true,
			description: "Debe permitir que los pedidos asignados comiencen su tránsito",
		},
		{
			name:        "assigned to delivered (invalid)",
			fromStatus:  models.OrderStatusAssigned,
			toStatus:    models.OrderStatusDelivered,
			expected:    false,
			description: "No debe permitir que los pedidos asignados se salten a entregados",
		},

		// From IN_TRANSIT
		{
			name:        "in transit to delivered",
			fromStatus:  models.OrderStatusInTransit,
			toStatus:    models.OrderStatusDelivered,
			expected:    true,
			description: "Debe permitir que los pedidos en tránsito se entreguen",
		},
		{
			name:        "in transit to cancelled (invalid)",
			fromStatus:  models.OrderStatusInTransit,
			toStatus:    models.OrderStatusCancelled,
			expected:    false,
			description: "No debe permitir que los pedidos en tránsito se cancelen",
		},

		// From DELIVERED (final state)
		{
			name:        "delivered to any state (invalid)",
			fromStatus:  models.OrderStatusDelivered,
			toStatus:    models.OrderStatusPending,
			expected:    false,
			description: "No debe permitir que los pedidos entregados se cambien de estado",
		},

		// From CANCELLED (final state)
		{
			name:        "cancelled to any state (invalid)",
			fromStatus:  models.OrderStatusCancelled,
			toStatus:    models.OrderStatusConfirmed,
			expected:    false,
			description: "No debe permitir que los pedidos cancelados se cambien de estado",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &models.Order{
				OrderStatus: tt.fromStatus,
			}

			result := order.CanTransitionTo(tt.toStatus)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestIsWithinBusinessHours(t *testing.T) {
	timezone := "America/Lima"
	businessStart := 6 * time.Hour // 6 AM
	businessEnd := 20 * time.Hour  // 8 PM

	// Create times in Lima timezone (UTC-5)
	lima, _ := time.LoadLocation("America/Lima")

	tests := []struct {
		name        string
		time        time.Time
		expected    bool
		description string
	}{
		{
			name:        "morning within hours",
			time:        time.Date(2024, 6, 15, 10, 0, 0, 0, lima), // 10 AM Lima time
			expected:    true,
			description: "Debe estar dentro de las horas comerciales en la mañana",
		},
		{
			name:        "afternoon within hours",
			time:        time.Date(2024, 6, 15, 15, 30, 0, 0, lima), // 3:30 PM Lima time
			expected:    true,
			description: "Debe estar dentro de las horas comerciales en la tarde",
		},
		{
			name:        "evening boundary (7:59 PM)",
			time:        time.Date(2024, 6, 15, 19, 59, 0, 0, lima), // 7:59 PM Lima time
			expected:    true,
			description: "Debe estar dentro de las horas comerciales justo antes de cerrar",
		},
		{
			name:        "evening boundary (8:00 PM)",
			time:        time.Date(2024, 6, 15, 20, 0, 0, 0, lima), // 8:00 PM Lima time
			expected:    false,
			description: "Debe estar fuera de las horas comerciales en el momento de cerrar",
		},
		{
			name:        "early morning (5:59 AM)",
			time:        time.Date(2024, 6, 15, 5, 59, 0, 0, lima), // 5:59 AM Lima time
			expected:    false,
			description: "Debe estar fuera de las horas comerciales antes de abrir",
		},
		{
			name:        "morning boundary (6:00 AM)",
			time:        time.Date(2024, 6, 15, 6, 0, 0, 0, lima), // 6:00 AM Lima time
			expected:    true,
			description: "Debe estar dentro de las horas comerciales en el momento de abrir",
		},
		{
			name:        "late night",
			time:        time.Date(2024, 6, 15, 23, 0, 0, 0, lima), // 11:00 PM Lima time
			expected:    false,
			description: "Debe estar fuera de las horas comerciales en la noche",
		},
		{
			name:        "midnight",
			time:        time.Date(2024, 6, 15, 0, 0, 0, 0, lima), // 12:00 AM Lima time
			expected:    false,
			description: "Debe estar fuera de las horas comerciales en la medianoche",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.IsWithinBusinessHours(tt.time, businessStart, businessEnd, timezone)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestIsWithinBusinessHours_InvalidTimezone(t *testing.T) {
	// Test with invalid timezone - should fallback to UTC
	invalidTimezone := "Invalid/Timezone"
	businessStart := 6 * time.Hour
	businessEnd := 20 * time.Hour

	// 10 AM UTC should be within business hours
	testTime := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)

	result := models.IsWithinBusinessHours(testTime, businessStart, businessEnd, invalidTimezone)
	assert.True(t, result, "Debe caer de regreso a UTC y seguir funcionando correctamente")
}

func TestOrder_BeforeCreate(t *testing.T) {
	clientID := uuid.New()
	order := testutil.CreateTestOrder(t, clientID)

	// Reset the UUID to test generation
	order.OrderID = uuid.Nil

	err := order.BeforeCreate(nil)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, order.OrderID, "Debe generar un nuevo UUID")
}

func TestOrderItem_BeforeCreate(t *testing.T) {
	orderID := uuid.New()
	productID := uuid.New()

	orderItem := &models.OrderItem{
		OrderID:   orderID,
		ProductID: productID,
		Quantity:  2,
		UnitPrice: 45.50,
	}

	err := orderItem.BeforeCreate(nil)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, orderItem.OrderItemID, "Debe generar un nuevo UUID")
	assert.Equal(t, 91.0, orderItem.Subtotal, "Debe calcular el subtotal correctamente")
}

func TestOrderStatus_Constants(t *testing.T) {
	// Test that order status constants are defined correctly
	assert.Equal(t, models.OrderStatus("PENDING"), models.OrderStatusPending)
	assert.Equal(t, models.OrderStatus("PENDING_OUT_OF_HOURS"), models.OrderStatusPendingOutOfHours)
	assert.Equal(t, models.OrderStatus("CONFIRMED"), models.OrderStatusConfirmed)
	assert.Equal(t, models.OrderStatus("ASSIGNED"), models.OrderStatusAssigned)
	assert.Equal(t, models.OrderStatus("IN_TRANSIT"), models.OrderStatusInTransit)
	assert.Equal(t, models.OrderStatus("DELIVERED"), models.OrderStatusDelivered)
	assert.Equal(t, models.OrderStatus("CANCELLED"), models.OrderStatusCancelled)
}

func TestOrder_StatusWorkflow(t *testing.T) {
	// Test the complete workflow of an order
	order := &models.Order{
		OrderStatus: models.OrderStatusPending,
	}

	// Test valid workflow
	validTransitions := []models.OrderStatus{
		models.OrderStatusConfirmed,
		models.OrderStatusAssigned,
		models.OrderStatusInTransit,
		models.OrderStatusDelivered,
	}

	for _, status := range validTransitions {
		assert.True(t, order.CanTransitionTo(status),
			"Debe permitir la transición desde %s a %s", order.OrderStatus, status)
		order.OrderStatus = status
	}

	// Test that final state cannot transition anywhere
	assert.False(t, order.CanTransitionTo(models.OrderStatusPending),
		"No debe permitir la transición desde el estado final")
}
