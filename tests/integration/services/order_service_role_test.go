package services

import (
	"backend/config"
	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/services"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Request DTOs for testing (these should ideally be in a separate package)
type CreateOrderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type CreateOrderRequest struct {
	ClientID            string                   `json:"client_id"`
	Items               []CreateOrderItemRequest `json:"items"`
	Latitude            float64                  `json:"latitude"`
	Longitude           float64                  `json:"longitude"`
	DeliveryAddressText string                   `json:"delivery_address_text"`
	PaymentNote         string                   `json:"payment_note"`
}

// Helper function to convert CreateOrderRequest to models
func (req *CreateOrderRequest) ToModels() (*models.Order, []models.OrderItem, error) {
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return nil, nil, err
	}

	order := &models.Order{
		ClientID:            clientID,
		Latitude:            req.Latitude,
		Longitude:           req.Longitude,
		DeliveryAddressText: req.DeliveryAddressText,
		PaymentNote:         req.PaymentNote,
	}

	items := make([]models.OrderItem, len(req.Items))
	for i, item := range req.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			return nil, nil, err
		}

		items[i] = models.OrderItem{
			ProductID: productID,
			Quantity:  item.Quantity,
		}
	}

	return order, items, nil
}

// OrderServiceRoleTestSuite tests order service with role-based permissions using real database
type OrderServiceRoleTestSuite struct {
	suite.Suite
	db             *gorm.DB
	config         *config.Config
	authService    auth.Service
	orderService   *services.OrderService
	userService    *services.UserService
	productService *services.ProductService

	// Test users for each role
	clientUser     *models.User
	repartidorUser *models.User
	adminUser      *models.User

	// Test data
	testProduct *models.Product
}

// SetupSuite runs once before the test suite
func (suite *OrderServiceRoleTestSuite) SetupSuite() {
	// Test configuration
	suite.config = &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     "5433",
			User:     "postgres",
			Password: "postgres",
			DBName:   "exactogas_test",
			SSLMode:  "disable",
		},
		JWT: config.JWTConfig{
			Secret:          "test-jwt-secret-key",
			AccessTokenExp:  15 * time.Minute,
			RefreshTokenExp: 7 * 24 * time.Hour,
		},
		App: config.AppConfig{
			BusinessHoursStart: 6 * time.Hour,  // 6 AM
			BusinessHoursEnd:   20 * time.Hour, // 8 PM
			TimeZone:           "America/Lima",
		},
	}

	// Connect to test database with foreign key constraint disabled
	var err error
	suite.db, err = gorm.Open(postgres.Open(suite.config.Database.GetDSN()), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		suite.T().Skip("PostgreSQL test database not available")
		return
	}

	// Auto-migrate
	err = suite.db.AutoMigrate(&models.User{}, &models.Product{}, &models.Order{}, &models.OrderItem{})
	require.NoError(suite.T(), err)

	// Drop any incorrect foreign key constraints that GORM might have created
	suite.db.Exec("ALTER TABLE products DROP CONSTRAINT IF EXISTS fk_order_items_product")
	suite.db.Exec("ALTER TABLE order_items DROP CONSTRAINT IF EXISTS fk_order_items_product")

	// Create services
	suite.authService = auth.NewService(suite.db, suite.config)

	userRepo := repositories.NewUserRepository(suite.db)
	productRepo := repositories.NewProductRepository(suite.db)
	orderRepo := repositories.NewOrderRepository(suite.db)

	suite.userService = services.NewUserService(userRepo)
	suite.productService = services.NewProductService(productRepo)

	// Create order service with proper dependencies (mocked notification and ws services for simplicity)
	suite.orderService = services.NewOrderService(
		orderRepo,
		userRepo,
		productRepo,
		nil, // notification service
		suite.config,
		nil, // websocket hub
	)
}

// SetupTest runs before each test
func (suite *OrderServiceRoleTestSuite) SetupTest() {
	// Clean database
	suite.db.Exec("TRUNCATE TABLE order_items, orders, products, users RESTART IDENTITY CASCADE")

	// Also clean any seed data that might have been inserted by migrations
	suite.db.Exec("DELETE FROM products WHERE name LIKE 'Balón de Gas%'")
	suite.db.Exec("DELETE FROM users WHERE email = 'admin@exactogas.com'")

	// Create test users for each role
	suite.createTestUsers()
	suite.createTestProduct()
}

// TearDownSuite runs once after the test suite
func (suite *OrderServiceRoleTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Exec("TRUNCATE TABLE order_items, orders, products, users RESTART IDENTITY CASCADE")
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
}

func (suite *OrderServiceRoleTestSuite) createTestUsers() {
	var err error

	// Use timestamp to make emails unique across test runs
	timestamp := time.Now().UnixNano()

	// Create CLIENT user
	suite.clientUser, err = suite.authService.RegisterUser(
		fmt.Sprintf("client%d@test.com", timestamp),
		"password123",
		"Test Client",
		fmt.Sprintf("+5199999%04d", timestamp%10000),
		models.UserRoleClient,
	)
	require.NoError(suite.T(), err)

	// Create REPARTIDOR user
	suite.repartidorUser, err = suite.authService.RegisterUser(
		fmt.Sprintf("repartidor%d@test.com", timestamp),
		"password123",
		"Test Repartidor",
		fmt.Sprintf("+5199999%04d", (timestamp+1)%10000),
		models.UserRoleRepartidor,
	)
	require.NoError(suite.T(), err)

	// Create ADMIN user
	suite.adminUser, err = suite.authService.RegisterUser(
		fmt.Sprintf("admin%d@test.com", timestamp),
		"password123",
		"Test Admin",
		fmt.Sprintf("+5199999%04d", (timestamp+2)%10000),
		models.UserRoleAdmin,
	)
	require.NoError(suite.T(), err)
}

func (suite *OrderServiceRoleTestSuite) createTestProduct() {
	suite.testProduct = &models.Product{
		Name:        "Balón 10kg",
		Description: "Balón de gas de 10 kilogramos",
		Price:       45.50,
		IsActive:    true,
	}

	err := suite.productService.Create(suite.testProduct)
	require.NoError(suite.T(), err)
}

// Helper function to create order for tests
func (suite *OrderServiceRoleTestSuite) createTestOrder(clientID string, items []CreateOrderItemRequest) (*models.Order, error) {
	createReq := &CreateOrderRequest{
		ClientID:            clientID,
		Items:               items,
		Latitude:            -12.046374,
		Longitude:           -77.042793,
		DeliveryAddressText: "Av. Test 123, Lima",
		PaymentNote:         "Billete de 50 soles",
	}

	orderModel, itemsModel, err := createReq.ToModels()
	if err != nil {
		return nil, err
	}

	return suite.orderService.CreateOrder(orderModel, itemsModel)
}

func (suite *OrderServiceRoleTestSuite) TestCreateOrder_ClientRole() {
	// Only CLIENT should be able to create orders for themselves
	orderItems := []CreateOrderItemRequest{
		{
			ProductID: suite.testProduct.ProductID.String(),
			Quantity:  2,
		},
	}

	// Client creating their own order - should succeed
	createdOrder, err := suite.createTestOrder(suite.clientUser.UserID.String(), orderItems)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), createdOrder)
	assert.Equal(suite.T(), suite.clientUser.UserID, createdOrder.ClientID)
	assert.Equal(suite.T(), models.OrderStatusPending, createdOrder.OrderStatus)

	// Verify order items were created correctly
	assert.Len(suite.T(), createdOrder.OrderItems, 1)
	assert.Equal(suite.T(), 2, createdOrder.OrderItems[0].Quantity)
	assert.Equal(suite.T(), 91.0, createdOrder.TotalAmount) // 2 * 45.50
}

func (suite *OrderServiceRoleTestSuite) TestOrderStatusTransitions_ByRole() {
	// First create an order as client
	orderItems := []CreateOrderItemRequest{
		{
			ProductID: suite.testProduct.ProductID.String(),
			Quantity:  1,
		},
	}

	order, err := suite.createTestOrder(suite.clientUser.UserID.String(), orderItems)
	require.NoError(suite.T(), err)
	orderID := order.OrderID.String()

	// Test 1: CLIENT cannot confirm orders (only cancel their own pending orders)
	_, err = suite.orderService.UpdateOrderStatus(
		orderID,
		models.OrderStatusConfirmed,
		suite.clientUser.UserID.String(),
		suite.clientUser.UserRole,
	)
	assert.Error(suite.T(), err, "Client should not be able to confirm orders")

	// Test 2: CLIENT can cancel their own pending orders
	cancelledOrder, err := suite.orderService.UpdateOrderStatus(
		orderID,
		models.OrderStatusCancelled,
		suite.clientUser.UserID.String(),
		suite.clientUser.UserRole,
	)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusCancelled, cancelledOrder.OrderStatus)
	assert.NotNil(suite.T(), cancelledOrder.CancelledAt)

	// Create a new order for the rest of the tests
	order, err = suite.createTestOrder(suite.clientUser.UserID.String(), orderItems)
	require.NoError(suite.T(), err)
	orderID = order.OrderID.String()

	// Test 3: ADMIN can confirm orders
	confirmedOrder, err := suite.orderService.UpdateOrderStatus(
		orderID,
		models.OrderStatusConfirmed,
		suite.adminUser.UserID.String(),
		suite.adminUser.UserRole,
	)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusConfirmed, confirmedOrder.OrderStatus)
	assert.NotNil(suite.T(), confirmedOrder.ConfirmedAt)

	// Test 4: ADMIN can assign repartidor
	assignedOrder, err := suite.orderService.AssignRepartidor(orderID, suite.repartidorUser.UserID.String())
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusAssigned, assignedOrder.OrderStatus)
	assert.Equal(suite.T(), suite.repartidorUser.UserID, *assignedOrder.AssignedRepartidorID)
	assert.NotNil(suite.T(), assignedOrder.AssignedAt)

	// Test 5: Only assigned REPARTIDOR can start transit
	inTransitOrder, err := suite.orderService.UpdateOrderStatus(
		orderID,
		models.OrderStatusInTransit,
		suite.repartidorUser.UserID.String(),
		suite.repartidorUser.UserRole,
	)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusInTransit, inTransitOrder.OrderStatus)

	// Test 6: Different REPARTIDOR cannot update the order
	// Create another repartidor
	otherRepartidor, err := suite.authService.RegisterUser(
		"other_repartidor@test.com",
		"password123",
		"Other Repartidor",
		"+51999999004",
		models.UserRoleRepartidor,
	)
	require.NoError(suite.T(), err)

	_, err = suite.orderService.UpdateOrderStatus(
		orderID,
		models.OrderStatusDelivered,
		otherRepartidor.UserID.String(),
		otherRepartidor.UserRole,
	)
	assert.Error(suite.T(), err, "Different repartidor should not be able to update assigned order")

	// Test 7: Only assigned REPARTIDOR can mark as delivered
	deliveredOrder, err := suite.orderService.UpdateOrderStatus(
		orderID,
		models.OrderStatusDelivered,
		suite.repartidorUser.UserID.String(),
		suite.repartidorUser.UserRole,
	)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusDelivered, deliveredOrder.OrderStatus)
	assert.NotNil(suite.T(), deliveredOrder.DeliveredAt)
}

func (suite *OrderServiceRoleTestSuite) TestRepartidorAutoAssignment() {
	// Create order as client
	orderItems := []CreateOrderItemRequest{
		{
			ProductID: suite.testProduct.ProductID.String(),
			Quantity:  1,
		},
	}

	order, err := suite.createTestOrder(suite.clientUser.UserID.String(), orderItems)
	require.NoError(suite.T(), err)

	// Verify order is initially unassigned
	assert.Nil(suite.T(), order.AssignedRepartidorID, "Order should initially be unassigned")
	assert.Equal(suite.T(), models.OrderStatusPending, order.OrderStatus)

	// REPARTIDOR confirms order (should auto-assign to themselves)
	confirmedOrder, err := suite.orderService.UpdateOrderStatus(
		order.OrderID.String(),
		models.OrderStatusConfirmed,
		suite.repartidorUser.UserID.String(),
		suite.repartidorUser.UserRole,
	)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusConfirmed, confirmedOrder.OrderStatus)

	// CRITICAL: Verify auto-assignment occurred
	assert.NotNil(suite.T(), confirmedOrder.AssignedRepartidorID,
		"Repartidor should be auto-assigned when confirming order")
	assert.Equal(suite.T(), suite.repartidorUser.UserID, *confirmedOrder.AssignedRepartidorID,
		"The repartidor who confirmed should be auto-assigned")
	assert.NotNil(suite.T(), confirmedOrder.AssignedAt, "AssignedAt timestamp should be set")
	assert.NotNil(suite.T(), confirmedOrder.ConfirmedAt, "ConfirmedAt timestamp should be set")
}

func (suite *OrderServiceRoleTestSuite) TestAdminNoAutoAssignment() {
	// Create order as client
	orderItems := []CreateOrderItemRequest{
		{
			ProductID: suite.testProduct.ProductID.String(),
			Quantity:  1,
		},
	}

	order, err := suite.createTestOrder(suite.clientUser.UserID.String(), orderItems)
	require.NoError(suite.T(), err)

	// Verify order is initially unassigned
	assert.Nil(suite.T(), order.AssignedRepartidorID, "Order should initially be unassigned")
	assert.Equal(suite.T(), models.OrderStatusPending, order.OrderStatus)

	// ADMIN confirms order (should NOT auto-assign to themselves)
	confirmedOrder, err := suite.orderService.UpdateOrderStatus(
		order.OrderID.String(),
		models.OrderStatusConfirmed,
		suite.adminUser.UserID.String(),
		suite.adminUser.UserRole,
	)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusConfirmed, confirmedOrder.OrderStatus)

	// CRITICAL: Verify NO auto-assignment occurred for admin
	assert.Nil(suite.T(), confirmedOrder.AssignedRepartidorID,
		"Admin should NOT be auto-assigned when confirming order")
	assert.Nil(suite.T(), confirmedOrder.AssignedAt, "AssignedAt should be nil for admin confirmation")
	assert.NotNil(suite.T(), confirmedOrder.ConfirmedAt, "ConfirmedAt timestamp should be set")
}

func (suite *OrderServiceRoleTestSuite) TestSetEstimatedArrivalTime_Permissions() {
	// Create and assign an order
	orderItems := []CreateOrderItemRequest{
		{
			ProductID: suite.testProduct.ProductID.String(),
			Quantity:  1,
		},
	}

	order, err := suite.createTestOrder(suite.clientUser.UserID.String(), orderItems)
	require.NoError(suite.T(), err)

	// Confirm and assign order
	_, err = suite.orderService.UpdateOrderStatus(
		order.OrderID.String(),
		models.OrderStatusConfirmed,
		suite.adminUser.UserID.String(),
		suite.adminUser.UserRole,
	)
	require.NoError(suite.T(), err)

	assignedOrder, err := suite.orderService.AssignRepartidor(order.OrderID.String(), suite.repartidorUser.UserID.String())
	require.NoError(suite.T(), err)

	// Test ETA setting by assigned repartidor
	eta := time.Now().Add(30 * time.Minute)
	updatedOrder, err := suite.orderService.SetEstimatedArrivalTime(assignedOrder.OrderID.String(), eta)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), updatedOrder.EstimatedArrivalTime)
	assert.True(suite.T(), updatedOrder.EstimatedArrivalTime.Equal(eta))
}

func (suite *OrderServiceRoleTestSuite) TestOrderPermissionsMatrix() {
	// Create test order
	orderItems := []CreateOrderItemRequest{
		{
			ProductID: suite.testProduct.ProductID.String(),
			Quantity:  1,
		},
	}

	// Permission matrix for PENDING state
	testCases := []struct {
		name          string
		userRole      models.UserRole
		userID        string
		targetStatus  models.OrderStatus
		shouldSucceed bool
		description   string
	}{
		{
			name:          "Client cancel own pending order",
			userRole:      models.UserRoleClient,
			userID:        suite.clientUser.UserID.String(),
			targetStatus:  models.OrderStatusCancelled,
			shouldSucceed: true,
			description:   "Clients should be able to cancel their own pending orders",
		},
		{
			name:          "Client confirm order (invalid)",
			userRole:      models.UserRoleClient,
			userID:        suite.clientUser.UserID.String(),
			targetStatus:  models.OrderStatusConfirmed,
			shouldSucceed: false,
			description:   "Clients should not be able to confirm orders",
		},
		{
			name:          "Repartidor confirm order",
			userRole:      models.UserRoleRepartidor,
			userID:        suite.repartidorUser.UserID.String(),
			targetStatus:  models.OrderStatusConfirmed,
			shouldSucceed: true,
			description:   "Repartidores should be able to confirm orders",
		},
		{
			name:          "Admin confirm order",
			userRole:      models.UserRoleAdmin,
			userID:        suite.adminUser.UserID.String(),
			targetStatus:  models.OrderStatusConfirmed,
			shouldSucceed: true,
			description:   "Admins should be able to confirm orders",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Create fresh order for each test
			newOrder, err := suite.createTestOrder(suite.clientUser.UserID.String(), orderItems)
			require.NoError(t, err)

			_, err = suite.orderService.UpdateOrderStatus(
				newOrder.OrderID.String(),
				tc.targetStatus,
				tc.userID,
				tc.userRole,
			)

			if tc.shouldSucceed {
				assert.NoError(t, err, tc.description)
			} else {
				assert.Error(t, err, tc.description)
			}
		})
	}
}

func (suite *OrderServiceRoleTestSuite) TestOrderOperationsSecurity() {
	// Create orders for different clients
	client2, err := suite.authService.RegisterUser(
		"client2@test.com",
		"password123",
		"Test Client 2",
		"+51999999005",
		models.UserRoleClient,
	)
	require.NoError(suite.T(), err)

	orderItems := []CreateOrderItemRequest{
		{
			ProductID: suite.testProduct.ProductID.String(),
			Quantity:  1,
		},
	}

	// Client 1's order
	order1, err := suite.createTestOrder(suite.clientUser.UserID.String(), orderItems)
	require.NoError(suite.T(), err)

	// Client 2's order
	order2, err := suite.createTestOrder(client2.UserID.String(), orderItems)
	require.NoError(suite.T(), err)

	// Test: Client 1 should NOT be able to cancel Client 2's order
	_, err = suite.orderService.UpdateOrderStatus(
		order2.OrderID.String(),
		models.OrderStatusCancelled,
		suite.clientUser.UserID.String(),
		suite.clientUser.UserRole,
	)
	assert.Error(suite.T(), err, "Client should not be able to cancel other client's orders")

	// Test: Client 1 CAN cancel their own order
	_, err = suite.orderService.UpdateOrderStatus(
		order1.OrderID.String(),
		models.OrderStatusCancelled,
		suite.clientUser.UserID.String(),
		suite.clientUser.UserRole,
	)
	assert.NoError(suite.T(), err, "Client should be able to cancel their own orders")
}

func (suite *OrderServiceRoleTestSuite) TestCompleteOrderWorkflow() {
	// Test complete workflow with proper role-based progression
	orderItems := []CreateOrderItemRequest{
		{
			ProductID: suite.testProduct.ProductID.String(),
			Quantity:  2,
		},
	}

	// Step 1: Client creates order
	order, err := suite.createTestOrder(suite.clientUser.UserID.String(), orderItems)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusPending, order.OrderStatus)

	orderID := order.OrderID.String()

	// Step 2: Admin confirms order
	order, err = suite.orderService.UpdateOrderStatus(
		orderID,
		models.OrderStatusConfirmed,
		suite.adminUser.UserID.String(),
		suite.adminUser.UserRole,
	)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusConfirmed, order.OrderStatus)

	// Step 3: Admin assigns repartidor
	order, err = suite.orderService.AssignRepartidor(orderID, suite.repartidorUser.UserID.String())
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusAssigned, order.OrderStatus)
	assert.Equal(suite.T(), suite.repartidorUser.UserID, *order.AssignedRepartidorID)

	// Step 4: Repartidor starts delivery
	order, err = suite.orderService.UpdateOrderStatus(
		orderID,
		models.OrderStatusInTransit,
		suite.repartidorUser.UserID.String(),
		suite.repartidorUser.UserRole,
	)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusInTransit, order.OrderStatus)

	// Step 5: Repartidor sets ETA
	eta := time.Now().Add(25 * time.Minute)
	order, err = suite.orderService.SetEstimatedArrivalTime(orderID, eta)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), order.EstimatedArrivalTime)

	// Step 6: Repartidor delivers order
	order, err = suite.orderService.UpdateOrderStatus(
		orderID,
		models.OrderStatusDelivered,
		suite.repartidorUser.UserID.String(),
		suite.repartidorUser.UserRole,
	)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusDelivered, order.OrderStatus)
	assert.NotNil(suite.T(), order.DeliveredAt)

	// Verify final state
	finalOrder, err := suite.orderService.GetOrderByID(orderID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.OrderStatusDelivered, finalOrder.OrderStatus)
	assert.NotNil(suite.T(), finalOrder.ConfirmedAt)
	assert.NotNil(suite.T(), finalOrder.AssignedAt)
	assert.NotNil(suite.T(), finalOrder.EstimatedArrivalTime)
	assert.NotNil(suite.T(), finalOrder.DeliveredAt)
}

// TestOrderServiceRoleTestSuite runs the test suite
func TestOrderServiceRoleTestSuite(t *testing.T) {
	suite.Run(t, new(OrderServiceRoleTestSuite))
}
