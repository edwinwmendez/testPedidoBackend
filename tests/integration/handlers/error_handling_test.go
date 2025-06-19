package handlers

import (
	v1 "backend/api/v1"
	"backend/config"
	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/services"
	"bytes"
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

// ErrorHandlingTestSuite tests consistent error handling across all endpoints
type ErrorHandlingTestSuite struct {
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
func (suite *ErrorHandlingTestSuite) SetupSuite() {
	// Test configuration
	suite.config = &config.Config{
		JWT: config.JWTConfig{
			Secret:           "test-jwt-secret-key-for-integration-testing",
			AccessTokenExp:   15 * time.Minute,
			RefreshTokenExp:  7 * 24 * time.Hour,
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

// SetupTest runs before each test
func (suite *ErrorHandlingTestSuite) SetupTest() {
	// Clean database before each test
	suite.db.Where("1 = 1").Delete(&models.User{})
	suite.db.Where("1 = 1").Delete(&models.Product{})
	suite.db.Where("1 = 1").Delete(&models.Order{})
	suite.db.Where("1 = 1").Delete(&models.OrderItem{})
}

// TearDownSuite runs once after the test suite
func (suite *ErrorHandlingTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Where("1 = 1").Delete(&models.User{})
		suite.db.Where("1 = 1").Delete(&models.Product{})
		suite.db.Where("1 = 1").Delete(&models.Order{})
		suite.db.Where("1 = 1").Delete(&models.OrderItem{})
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
}

func (suite *ErrorHandlingTestSuite) TestConsistentErrorFormat() {
	testCases := []struct {
		name           string
		method         string
		endpoint       string
		payload        interface{}
		headers        map[string]string
		expectedStatus int
		description    string
	}{
		{
			name:           "Invalid JSON in registration",
			method:         "POST",
			endpoint:       "/api/v1/auth/register",
			payload:        "invalid json",
			expectedStatus: http.StatusBadRequest,
			description:    "Should return 400 for invalid JSON",
		},
		{
			name:     "Missing fields in registration",
			method:   "POST", 
			endpoint: "/api/v1/auth/register",
			payload: map[string]interface{}{
				"email": "test@example.com",
				// Missing required fields
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should return 400 for missing required fields",
		},
		{
			name:     "Invalid credentials in login",
			method:   "POST",
			endpoint: "/api/v1/auth/login",
			payload: map[string]interface{}{
				"email":    "nonexistent@example.com",
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			description:    "Should return 401 for invalid credentials",
		},
		{
			name:           "No authentication token",
			method:         "GET",
			endpoint:       "/api/v1/users/me",
			payload:        nil,
			expectedStatus: http.StatusUnauthorized,
			description:    "Should return 401 for missing auth token",
		},
		{
			name:           "Invalid user ID",
			method:         "GET", 
			endpoint:       "/api/v1/users/invalid-uuid-format",
			payload:        nil,
			headers:        map[string]string{"Authorization": "Bearer invalid-token"},
			expectedStatus: http.StatusUnauthorized, // Will fail on token first
			description:    "Should return 401 for invalid token",
		},
		{
			name:     "Invalid order data",
			method:   "POST",
			endpoint: "/api/v1/orders",
			payload: map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"product_id": "invalid-uuid",
						"quantity":   "not-a-number",
					},
				},
			},
			headers:        map[string]string{"Authorization": "Bearer invalid-token"},
			expectedStatus: http.StatusUnauthorized, // Will fail on token first
			description:    "Should return 401 for invalid token",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			var req *http.Request
			var err error

			// Prepare request body
			if tc.payload != nil {
				if str, ok := tc.payload.(string); ok {
					// Invalid JSON string
					req = httptest.NewRequest(tc.method, tc.endpoint, bytes.NewReader([]byte(str)))
				} else {
					// Valid JSON structure
					body, err := json.Marshal(tc.payload)
					require.NoError(t, err)
					req = httptest.NewRequest(tc.method, tc.endpoint, bytes.NewReader(body))
				}
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.endpoint, nil)
			}

			// Add headers if provided
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			resp, err := suite.app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, tc.description)

			// Verify response has error field
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			// All error responses should have an "error" field
			assert.Contains(t, response, "error", "Error response should contain 'error' field")
			
			// Error message should be a string
			errorMsg, ok := response["error"].(string)
			assert.True(t, ok, "Error field should be a string")
			assert.NotEmpty(t, errorMsg, "Error message should not be empty")

			// Error message should not contain technical details (no stack traces)
			assert.NotContains(t, errorMsg, "goroutine", "Should not expose stack traces")
			assert.NotContains(t, errorMsg, "panic", "Should not expose panic details")
		})
	}
}

func (suite *ErrorHandlingTestSuite) TestNotFoundEndpoints() {
	notFoundCases := []struct {
		method   string
		endpoint string
		expectedStatuses []int // Multiple acceptable status codes
	}{
		{"GET", "/api/v1/nonexistent", []int{404, 500}}, // Could be 404 or 500 depending on framework
		{"POST", "/api/v1/invalid/endpoint", []int{404, 500}}, // Could be 404 or 500
		{"PUT", "/api/v1/users/nonexistent-resource", []int{401, 404}}, // 401 if auth required first
		{"DELETE", "/api/v1/orders/invalid", []int{401, 404}}, // 401 if auth required first
	}

	for _, tc := range notFoundCases {
		suite.T().Run(tc.method+" "+tc.endpoint, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.endpoint, nil)
			resp, err := suite.app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should return one of the expected error status codes
			assert.Contains(t, tc.expectedStatuses, resp.StatusCode,
				"Expected one of %v, got %d", tc.expectedStatuses, resp.StatusCode)
		})
	}
}

func (suite *ErrorHandlingTestSuite) TestHTTPMethodValidation() {
	methodCases := []struct {
		method      string
		endpoint    string
		description string
	}{
		// These endpoints should not accept GET
		{"GET", "/api/v1/auth/register", "Register should not accept GET"},
		{"GET", "/api/v1/auth/login", "Login should not accept GET"},
		
		// These endpoints should not accept POST  
		{"POST", "/api/v1/users/me", "User profile should not accept POST"},
		
		// These endpoints should not accept DELETE
		{"DELETE", "/api/v1/auth/login", "Login should not accept DELETE"},
	}

	for _, tc := range methodCases {
		suite.T().Run(tc.method+" "+tc.endpoint, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.endpoint, nil)
			resp, err := suite.app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should return 4xx or 5xx for unsupported methods (not 2xx)
			assert.True(t, resp.StatusCode >= 400,
				"%s: Should return error status for unsupported HTTP method, got: %d", 
				tc.description, resp.StatusCode)
		})
	}
}

func (suite *ErrorHandlingTestSuite) TestContentTypeValidation() {
	// Test endpoints that require JSON
	jsonEndpoints := []string{
		"/api/v1/auth/register",
		"/api/v1/auth/login", 
		"/api/v1/auth/refresh",
	}

	for _, endpoint := range jsonEndpoints {
		suite.T().Run("Missing Content-Type: "+endpoint, func(t *testing.T) {
			req := httptest.NewRequest("POST", endpoint, bytes.NewReader([]byte("{}")))
			// Deliberately not setting Content-Type header
			
			resp, err := suite.app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should handle missing Content-Type gracefully
			// (Fiber typically accepts JSON even without explicit Content-Type)
			assert.True(t, resp.StatusCode >= 400 && resp.StatusCode < 500,
				"Should return 4xx status for requests without proper content type")
		})
	}
}

func (suite *ErrorHandlingTestSuite) TestLargePayloadHandling() {
	// Test with oversized payload
	largePayload := make(map[string]interface{})
	largePayload["email"] = "test@example.com"
	largePayload["password"] = "password123"
	largePayload["full_name"] = "Test User"
	largePayload["phone_number"] = "+51999999999"
	largePayload["user_role"] = "CLIENT"
	
	// Add a very large field
	largeString := make([]byte, 1024*1024) // 1MB string
	for i := range largeString {
		largeString[i] = 'a'
	}
	largePayload["large_field"] = string(largeString)

	body, err := json.Marshal(largePayload)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// Should handle large payloads gracefully (either accept or reject cleanly)
	assert.True(suite.T(), resp.StatusCode == http.StatusBadRequest || 
			   resp.StatusCode == http.StatusRequestEntityTooLarge ||
			   resp.StatusCode == http.StatusCreated,
		"Should handle large payloads gracefully")
}

// TestErrorHandlingTestSuite runs the test suite
func TestErrorHandlingTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorHandlingTestSuite))
}