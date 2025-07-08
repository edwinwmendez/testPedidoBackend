package handlers

import (
	v1 "backend/api/v1"
	"backend/config"
	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/services"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// HealthTestSuite tests the health endpoint
type HealthTestSuite struct {
	suite.Suite
	app            *fiber.App
	db             *gorm.DB
	authService    auth.Service
	userService    *services.UserService
	productService *services.ProductService
	orderService   *services.OrderService
	config         *config.Config
}

// SetupSuite runs once before the test suite
func (suite *HealthTestSuite) SetupSuite() {
	// Test configuration
	suite.config = &config.Config{
		JWT: config.JWTConfig{
			Secret:          "test-jwt-secret-key-for-integration-testing",
			AccessTokenExp:  15 * time.Minute,
			RefreshTokenExp: 7 * 24 * time.Hour,
		},
	}

	// Setup test database with foreign key constraint disabled
	var err error
	suite.db, err = gorm.Open(postgres.Open("host=localhost port=5433 user=postgres password=postgres dbname=exactogas_test sslmode=disable"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		suite.T().Skip("PostgreSQL not available for integration testing")
		return
	}

	// Auto-migrate
	err = suite.db.AutoMigrate(&models.User{}, &models.Product{}, &models.Order{}, &models.OrderItem{})
	require.NoError(suite.T(), err)

	// Create repositories
	userRepo := repositories.NewUserRepository(suite.db)
	productRepo := repositories.NewProductRepository(suite.db)
	orderRepo := repositories.NewOrderRepository(suite.db)

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
		nil, // websocket hub
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

// TearDownSuite runs once after the test suite
func (suite *HealthTestSuite) TearDownSuite() {
	if suite.db != nil {
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
}

func (suite *HealthTestSuite) TestHealthEndpoint() {
	// Test health endpoint response
	req := httptest.NewRequest("GET", "/api/v1/health", nil)

	start := time.Now()
	resp, err := suite.app.Test(req)
	duration := time.Since(start)

	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// Should respond quickly
	assert.Less(suite.T(), duration, 100*time.Millisecond,
		"Health endpoint should respond in <100ms, took: %v", duration)

	// Should return 200 OK
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Should return valid JSON
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	// Should contain required fields
	assert.Contains(suite.T(), response, "status")
	assert.Equal(suite.T(), "ok", response["status"])
	assert.Contains(suite.T(), response, "message")
	assert.Equal(suite.T(), "API funcionando correctamente", response["message"])
}

func (suite *HealthTestSuite) TestHealthEndpointMultipleRequests() {
	// Test multiple concurrent requests to health endpoint
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)

		resp, err := suite.app.Test(req)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	}
}

// TestHealthTestSuite runs the test suite
func TestHealthTestSuite(t *testing.T) {
	suite.Run(t, new(HealthTestSuite))
}
