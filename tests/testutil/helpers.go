package testutil

import (
	"backend/internal/models"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// MockTime represents a fixed time for testing
var MockTime = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

// CreateTestUser creates a test user with default values
func CreateTestUser(t *testing.T, role models.UserRole) *models.User {
	user := &models.User{
		UserID:      uuid.New(),
		FullName:    "Test User",
		Email:       "test@example.com",
		PhoneNumber: "+51999999999",
		UserRole:    role,
		CreatedAt:   MockTime,
		UpdatedAt:   MockTime,
	}

	err := user.SetPassword("testpassword123")
	require.NoError(t, err)

	return user
}

// CreateTestProduct creates a test product with default values
func CreateTestProduct(t *testing.T) *models.Product {
	return &models.Product{
		ProductID:   uuid.New(),
		Name:        "Balón 10kg",
		Description: "Balón de gas de 10 kilogramos",
		Price:       45.50,
		IsActive:    true,
		CreatedAt:   MockTime,
		UpdatedAt:   MockTime,
	}
}

// CreateTestOrder creates a test order with default values
func CreateTestOrder(t *testing.T, clientID uuid.UUID) *models.Order {
	return &models.Order{
		OrderID:             uuid.New(),
		ClientID:            clientID,
		OrderStatus:         models.OrderStatusPending,
		TotalAmount:         45.50,
		Latitude:            -12.046374,
		Longitude:           -77.042793,
		DeliveryAddressText: "Av. Test 123, Lima, Perú",
		PaymentNote:         "Billete de 50 soles",
		OrderTime:           MockTime,
		CreatedAt:           MockTime,
		UpdatedAt:           MockTime,
	}
}

// CreateTestOrderItem creates a test order item
func CreateTestOrderItem(t *testing.T, orderID, productID uuid.UUID) *models.OrderItem {
	return &models.OrderItem{
		OrderItemID: uuid.New(),
		OrderID:     orderID,
		ProductID:   productID,
		Quantity:    1,
		UnitPrice:   45.50,
		Subtotal:    45.50,
	}
}

// AssertError checks if an error occurred and has the expected message
func AssertError(t *testing.T, expectedMsg string, err error) {
	require.Error(t, err)
	require.Contains(t, err.Error(), expectedMsg)
}

// AssertNoError checks that no error occurred
func AssertNoError(t *testing.T, err error) {
	require.NoError(t, err)
}

// HashPassword creates a bcrypt hash for testing
func HashPassword(t *testing.T, password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	return string(hash)
}

// TimeValue is a helper for dealing with time pointers in tests
func TimeValue(t time.Time) *time.Time {
	return &t
}

// StringValue is a helper for dealing with string pointers in tests
func StringValue(s string) *string {
	return &s
}

// UUIDValue is a helper for dealing with UUID pointers in tests
func UUIDValue(id uuid.UUID) *uuid.UUID {
	return &id
}

// AnyTime is a matcher for database/sql driver.Value that matches any time
type AnyTime struct{}

func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

// AnyUUID is a matcher for database/sql driver.Value that matches any UUID
type AnyUUID struct{}

func (a AnyUUID) Match(v driver.Value) bool {
	switch v.(type) {
	case string, []byte, uuid.UUID:
		return true
	default:
		return false
	}
}

// BusinessHours represents business hours for testing
var BusinessHours = struct {
	Start int
	End   int
}{
	Start: 6,  // 6 AM
	End:   20, // 8 PM
}

// CreateTimeInBusinessHours creates a time within business hours (Lima timezone)
func CreateTimeInBusinessHours() time.Time {
	lima, _ := time.LoadLocation("America/Lima")
	return time.Date(2024, 6, 15, 10, 0, 0, 0, lima) // 10 AM Lima time
}

// CreateTimeOutOfBusinessHours creates a time outside business hours (Lima timezone)
func CreateTimeOutOfBusinessHours() time.Time {
	lima, _ := time.LoadLocation("America/Lima")
	return time.Date(2024, 6, 15, 22, 0, 0, 0, lima) // 10 PM Lima time
}
