package handlers

import (
	v1 "backend/api/v1"
	"backend/config"
	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/services"
	"backend/internal/ws"
	"backend/tests/integration/mocks"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OrderHandlerTestSuite defines the test suite for order endpoints
type OrderHandlerTestSuite struct {
	suite.Suite
	app               *fiber.App
	db                *gorm.DB
	authService       auth.Service
	userService       *services.UserService
	productService    *services.ProductService
	orderService      *services.OrderService
	config            *config.Config
	mockWebSocketHub  *mocks.MockWebSocketHub
	// Test users
	clientUser        *models.User
	repartidorUser    *models.User
	adminUser         *models.User
	// Test products
	testProduct       *models.Product
	inactiveProduct   *models.Product
	// Test tokens
	clientToken       string
	repartidorToken   string
	adminToken        string
}

// SetupSuite runs once before the test suite
func (suite *OrderHandlerTestSuite) SetupSuite() {
	// Test configuration
	suite.config = &config.Config{
		JWT: config.JWTConfig{
			Secret:           "test-jwt-secret-key-for-integration-testing",
			AccessTokenExp:   15 * time.Minute,
			RefreshTokenExp:  7 * 24 * time.Hour,
		},
		App: config.AppConfig{
			BusinessHoursStart: 8 * time.Hour,  // 8:00 AM
			BusinessHoursEnd:   22 * time.Hour, // 10:00 PM
			TimeZone:           "America/Lima",
		},
	}
	
	// Setup test database with foreign key constraint disabled
	var err error
	suite.db, err = gorm.Open(postgres.Open("host=localhost port=5433 user=postgres password=postgres dbname=exactogas_test sslmode=disable"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		suite.T().Skip("PostgreSQL not available for integration testing")
		return
	}
	
	// Auto-migrate
	err = suite.db.AutoMigrate(&models.User{}, &models.Product{}, &models.Order{}, &models.OrderItem{})
	require.NoError(suite.T(), err)
	
	// Drop any incorrect foreign key constraints that GORM might have created
	suite.db.Exec("ALTER TABLE products DROP CONSTRAINT IF EXISTS fk_order_items_product")
	suite.db.Exec("ALTER TABLE order_items DROP CONSTRAINT IF EXISTS fk_order_items_product")
	
	// Create repositories
	userRepo := repositories.NewUserRepository(suite.db)
	productRepo := repositories.NewProductRepository(suite.db)
	orderRepo := repositories.NewOrderRepository(suite.db)
	
	// Create mock WebSocket hub
	suite.mockWebSocketHub = mocks.NewMockWebSocketHub()
	
	// Create services
	suite.authService = auth.NewService(suite.db, suite.config)
	suite.userService = services.NewUserService(userRepo)
	suite.productService = services.NewProductService(productRepo)
	suite.orderService = services.NewOrderService(
		orderRepo,
		userRepo,
		productRepo,
		nil, // notification service
		suite.config,
		suite.mockWebSocketHub, // Use mock WebSocket hub
	)
	
	// Setup Fiber app with routes
	suite.app = fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})
	
	// Setup routes with proper services
	v1.SetupRoutes(suite.app, suite.authService, suite.userService, suite.productService, suite.orderService)
}

// SetupTest runs before each test
func (suite *OrderHandlerTestSuite) SetupTest() {
	// Clean database before each test
	suite.db.Where("1 = 1").Delete(&models.OrderItem{})
	suite.db.Where("1 = 1").Delete(&models.Order{})
	suite.db.Where("1 = 1").Delete(&models.Product{})
	suite.db.Where("1 = 1").Delete(&models.User{})
	
	// Also clean any seed data that might have been inserted by migrations
	suite.db.Exec("DELETE FROM products WHERE name LIKE 'Balón de Gas%'")
	suite.db.Exec("DELETE FROM users WHERE email = 'admin@exactogas.com'")
	
	// Reset mock WebSocket hub
	suite.mockWebSocketHub.Reset()
	
	// Create test users with unique emails using timestamp
	var err error
	timestamp := time.Now().UnixNano()
	
	suite.clientUser, err = suite.authService.RegisterUser(
		fmt.Sprintf("client%d@test.com", timestamp),
		"password123",
		"Test Client",
		fmt.Sprintf("+5199999%04d", timestamp%10000),
		models.UserRoleClient,
	)
	require.NoError(suite.T(), err)
	
	suite.repartidorUser, err = suite.authService.RegisterUser(
		fmt.Sprintf("repartidor%d@test.com", timestamp),
		"password123",
		"Test Repartidor",
		fmt.Sprintf("+5199999%04d", (timestamp+1)%10000),
		models.UserRoleRepartidor,
	)
	require.NoError(suite.T(), err)
	
	suite.adminUser, err = suite.authService.RegisterUser(
		fmt.Sprintf("admin%d@test.com", timestamp),
		"password123",
		"Test Admin",
		fmt.Sprintf("+5199999%04d", (timestamp+2)%10000),
		models.UserRoleAdmin,
	)
	require.NoError(suite.T(), err)
	
	// Create test products
	suite.testProduct = &models.Product{
		ProductID:   uuid.New(),
		Name:        "Balón de Gas 10kg",
		Description: "Balón de gas doméstico 10kg",
		Price:       45.50,
		ImageURL:    "https://example.com/gas10kg.jpg",
		IsActive:    true,
	}
	err = suite.productService.Create(suite.testProduct)
	require.NoError(suite.T(), err)
	
	suite.inactiveProduct = &models.Product{
		ProductID:   uuid.New(),
		Name:        "Producto Inactivo",
		Description: "Producto que no está disponible",
		Price:       25.00,
		ImageURL:    "https://example.com/inactive.jpg",
		IsActive:    false,
	}
	err = suite.productService.Create(suite.inactiveProduct)
	require.NoError(suite.T(), err)
	
	// Generate tokens for test users using their actual emails
	clientTokenPair, err := suite.authService.Login(suite.clientUser.Email, "password123")
	require.NoError(suite.T(), err)
	suite.clientToken = clientTokenPair.AccessToken
	
	repartidorTokenPair, err := suite.authService.Login(suite.repartidorUser.Email, "password123")
	require.NoError(suite.T(), err)
	suite.repartidorToken = repartidorTokenPair.AccessToken
	
	adminTokenPair, err := suite.authService.Login(suite.adminUser.Email, "password123")
	require.NoError(suite.T(), err)
	suite.adminToken = adminTokenPair.AccessToken
}

// TearDownSuite runs once after the test suite
func (suite *OrderHandlerTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Where("1 = 1").Delete(&models.OrderItem{})
		suite.db.Where("1 = 1").Delete(&models.Order{})
		suite.db.Where("1 = 1").Delete(&models.Product{})
		suite.db.Where("1 = 1").Delete(&models.User{})
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
}

func (suite *OrderHandlerTestSuite) TestCreateOrder_Success_WithRealTimeNotifications() {
	// Test data for order creation
	orderPayload := map[string]interface{}{
		"delivery_address_text": "Av. Javier Prado 123, San Isidro, Lima",
		"latitude":              -12.0970,
		"longitude":             -77.0283,
		"items": []map[string]interface{}{
			{
				"product_id": suite.testProduct.ProductID.String(),
				"quantity":   2,
			},
		},
	}
	
	body, err := json.Marshal(orderPayload)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.clientToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Verify HTTP response
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	
	// The handler returns the order directly, not wrapped in an "order" field
	order := response
	require.NotNil(suite.T(), order)
	
	assert.Equal(suite.T(), suite.clientUser.UserID.String(), order["client_id"])
	assert.Equal(suite.T(), orderPayload["delivery_address_text"], order["delivery_address_text"])
	assert.Equal(suite.T(), 91.0, order["total_amount"]) // 45.50 * 2 = 91.0
	assert.NotEmpty(suite.T(), order["order_id"])
	
	// Get the order ID for verification
	orderID := order["order_id"].(string)
	
	// **CRITICAL TEST: Verify WebSocket notifications were sent**
	
	// 1. Verify notification was sent to REPARTIDOR role
	repartidorMessages := suite.mockWebSocketHub.GetMessagesForRole("REPARTIDOR")
	assert.Len(suite.T(), repartidorMessages, 1, "Should send exactly 1 message to REPARTIDOR role")
	
	if len(repartidorMessages) > 0 {
		repartidorMsg := repartidorMessages[0]
		assert.Equal(suite.T(), ws.NewOrderAvailable, repartidorMsg.Type)
		
		// Verify message payload structure
		var payload map[string]interface{}
		err = json.Unmarshal(repartidorMsg.Payload, &payload)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), orderID, payload["order_id"])
		assert.Equal(suite.T(), suite.clientUser.UserID.String(), payload["client_id"])
		assert.Equal(suite.T(), suite.clientUser.FullName, payload["client_name"])
		assert.Equal(suite.T(), orderPayload["delivery_address_text"], payload["address"])
		assert.Equal(suite.T(), 91.0, payload["total_amount"])
		assert.NotEmpty(suite.T(), payload["order_time"])
	}
	
	// 2. Verify notification was sent to ADMIN role
	adminMessages := suite.mockWebSocketHub.GetMessagesForRole("ADMIN")
	assert.Len(suite.T(), adminMessages, 1, "Should send exactly 1 message to ADMIN role")
	
	if len(adminMessages) > 0 {
		adminMsg := adminMessages[0]
		assert.Equal(suite.T(), ws.NewOrderAvailable, adminMsg.Type)
		
		// Verify message payload structure (should be identical to repartidor message)
		var payload map[string]interface{}
		err = json.Unmarshal(adminMsg.Payload, &payload)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), orderID, payload["order_id"])
		assert.Equal(suite.T(), suite.clientUser.UserID.String(), payload["client_id"])
		assert.Equal(suite.T(), suite.clientUser.FullName, payload["client_name"])
		assert.Equal(suite.T(), orderPayload["delivery_address_text"], payload["address"])
		assert.Equal(suite.T(), 91.0, payload["total_amount"])
		assert.NotEmpty(suite.T(), payload["order_time"])
	}
	
	// 3. Verify no messages were sent to CLIENT role (they shouldn't get new order notifications)
	clientMessages := suite.mockWebSocketHub.GetMessagesForRole("CLIENT")
	assert.Len(suite.T(), clientMessages, 0, "CLIENT role should not receive new order notifications")
	
	// 4. Verify no direct user messages were sent (notifications go to roles, not specific users)
	clientUserMessages := suite.mockWebSocketHub.GetMessagesForUser(suite.clientUser.UserID.String())
	assert.Len(suite.T(), clientUserMessages, 0, "No direct user messages should be sent during order creation")
	
	// 5. Verify total message count
	totalMessages := suite.mockWebSocketHub.GetMessageCount()
	assert.Equal(suite.T(), 2, totalMessages, "Should send exactly 2 messages total (1 to REPARTIDOR, 1 to ADMIN)")
	
	// Verify order status is set correctly based on business hours
	orderStatus := order["order_status"].(string)
	// During business hours, order should be PENDING
	// Outside business hours, order should be PENDING_OUT_OF_HOURS
	assert.Contains(suite.T(), []string{"PENDING", "PENDING_OUT_OF_HOURS"}, orderStatus)
}

func (suite *OrderHandlerTestSuite) TestCreateOrder_WithInactiveProduct_ShouldFail() {
	// Test creating order with inactive product
	orderPayload := map[string]interface{}{
		"delivery_address_text": "Av. Javier Prado 123, San Isidro, Lima",
		"latitude":              -12.0970,
		"longitude":             -77.0283,
		"items": []map[string]interface{}{
			{
				"product_id": suite.inactiveProduct.ProductID.String(),
				"quantity":   1,
			},
		},
	}
	
	body, err := json.Marshal(orderPayload)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.clientToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Should fail with bad request
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
	
	// Verify no WebSocket notifications were sent since order creation failed
	repartidorMessages := suite.mockWebSocketHub.GetMessagesForRole("REPARTIDOR")
	assert.Len(suite.T(), repartidorMessages, 0, "No notifications should be sent for failed order")
	
	adminMessages := suite.mockWebSocketHub.GetMessagesForRole("ADMIN")
	assert.Len(suite.T(), adminMessages, 0, "No notifications should be sent for failed order")
	
	totalMessages := suite.mockWebSocketHub.GetMessageCount()
	assert.Equal(suite.T(), 0, totalMessages, "No messages should be sent for failed order creation")
}

func (suite *OrderHandlerTestSuite) TestCreateOrder_WithoutAuthentication_ShouldFail() {
	// Test creating order without authentication
	orderPayload := map[string]interface{}{
		"delivery_address_text": "Av. Javier Prado 123, San Isidro, Lima",
		"latitude":              -12.0970,
		"longitude":             -77.0283,
		"items": []map[string]interface{}{
			{
				"product_id": suite.testProduct.ProductID.String(),
				"quantity":   1,
			},
		},
	}
	
	body, err := json.Marshal(orderPayload)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Should fail with unauthorized
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
	
	// Verify no WebSocket notifications were sent
	totalMessages := suite.mockWebSocketHub.GetMessageCount()
	assert.Equal(suite.T(), 0, totalMessages, "No messages should be sent for unauthorized request")
}

func (suite *OrderHandlerTestSuite) TestCreateOrder_NonClientRole_ShouldFail() {
	// Test creating order with REPARTIDOR role (should fail)
	orderPayload := map[string]interface{}{
		"delivery_address_text": "Av. Javier Prado 123, San Isidro, Lima",
		"latitude":              -12.0970,
		"longitude":             -77.0283,
		"items": []map[string]interface{}{
			{
				"product_id": suite.testProduct.ProductID.String(),
				"quantity":   1,
			},
		},
	}
	
	body, err := json.Marshal(orderPayload)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.repartidorToken) // Using repartidor token
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Should fail - only clients can create orders (returns 403 Forbidden)
	assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)
	
	// Verify no WebSocket notifications were sent
	totalMessages := suite.mockWebSocketHub.GetMessageCount()
	assert.Equal(suite.T(), 0, totalMessages, "No messages should be sent for invalid role request")
}

func (suite *OrderHandlerTestSuite) TestCreateOrder_MultipleProducts_Success() {
	// Create another test product
	secondProduct := &models.Product{
		ProductID:   uuid.New(),
		Name:        "Balón de Gas 5kg",
		Description: "Balón de gas doméstico 5kg",
		Price:       25.75,
		ImageURL:    "https://example.com/gas5kg.jpg",
		IsActive:    true,
	}
	err := suite.productService.Create(secondProduct)
	require.NoError(suite.T(), err)
	
	// Test creating order with multiple products
	orderPayload := map[string]interface{}{
		"delivery_address_text": "Av. Arequipa 456, Miraflores, Lima",
		"latitude":              -12.1207,
		"longitude":             -77.0282,
		"items": []map[string]interface{}{
			{
				"product_id": suite.testProduct.ProductID.String(),
				"quantity":   1,
			},
			{
				"product_id": secondProduct.ProductID.String(),
				"quantity":   3,
			},
		},
	}
	
	body, err := json.Marshal(orderPayload)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.clientToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Verify HTTP response
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	
	// Verify order total calculation: (45.50 * 1) + (25.75 * 3) = 45.50 + 77.25 = 122.75
	// The handler returns the order directly, not wrapped in an "order" field
	order := response
	require.NotNil(suite.T(), order)
	assert.Equal(suite.T(), 122.75, order["total_amount"])
	
	// Verify WebSocket notifications were sent for this multi-product order
	repartidorMessages := suite.mockWebSocketHub.GetMessagesForRole("REPARTIDOR")
	assert.Len(suite.T(), repartidorMessages, 1, "Should send notification for multi-product order")
	
	adminMessages := suite.mockWebSocketHub.GetMessagesForRole("ADMIN")
	assert.Len(suite.T(), adminMessages, 1, "Should send notification for multi-product order")
	
	// Verify payload contains correct total amount
	if len(repartidorMessages) > 0 {
		var payload map[string]interface{}
		err = json.Unmarshal(repartidorMessages[0].Payload, &payload)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 122.75, payload["total_amount"])
	}
}

// TestOrderHandlerTestSuite runs the test suite
func TestOrderHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(OrderHandlerTestSuite))
}