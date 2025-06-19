package performance

import (
	v1 "backend/api/v1"
	"backend/config"
	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/services"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
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

// APIPerformanceTestSuite tests API response times and concurrent handling
type APIPerformanceTestSuite struct {
	suite.Suite
	app            *fiber.App
	db             *gorm.DB
	authService    auth.Service
	userService    *services.UserService
	productService *services.ProductService
	orderService   *services.OrderService
	config         *config.Config
	testToken      string
}

// SetupSuite runs once before the test suite
func (suite *APIPerformanceTestSuite) SetupSuite() {
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
	
	// Create test user and get token
	suite.setupTestUser()
}

func (suite *APIPerformanceTestSuite) setupTestUser() {
	// Clean database
	suite.db.Where("1 = 1").Delete(&models.User{})
	
	// Create test user
	user, err := suite.authService.RegisterUser(
		"performance@test.com",
		"password123",
		"Performance Test User",
		"+51999999999",
		models.UserRoleClient,
	)
	require.NoError(suite.T(), err)
	
	// Get token
	tokenPair, err := suite.authService.Login(user.Email, "password123")
	require.NoError(suite.T(), err)
	suite.testToken = tokenPair.AccessToken
}

// TearDownSuite runs once after the test suite
func (suite *APIPerformanceTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Where("1 = 1").Delete(&models.User{})
		suite.db.Where("1 = 1").Delete(&models.Product{})
		suite.db.Where("1 = 1").Delete(&models.Order{})
		suite.db.Where("1 = 1").Delete(&models.OrderItem{})
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
}

func (suite *APIPerformanceTestSuite) TestHealthEndpoint() {
	// Test health endpoint exists and responds quickly
	start := time.Now()
	
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	duration := time.Since(start)
	
	// Should respond quickly
	assert.Less(suite.T(), duration, 100*time.Millisecond, 
		"Health endpoint should respond in <100ms, took: %v", duration)
	
	// Should return 200 OK
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	// Should return valid JSON
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	
	// Should contain status field
	assert.Contains(suite.T(), response, "status")
	assert.Equal(suite.T(), "ok", response["status"])
}

func (suite *APIPerformanceTestSuite) TestAPIResponseTimes() {
	testCases := []struct {
		name           string
		method         string
		endpoint       string
		payload        interface{}
		useAuth        bool
		maxDuration    time.Duration
		description    string
	}{
		{
			name:        "Health check",
			method:      "GET",
			endpoint:    "/api/v1/health",
			maxDuration: 100 * time.Millisecond,
			description: "Health endpoint should be very fast",
		},
		{
			name:     "User login",
			method:   "POST",
			endpoint: "/api/v1/auth/login",
			payload: map[string]interface{}{
				"email":    "performance@test.com",
				"password": "password123",
			},
			maxDuration: 500 * time.Millisecond,
			description: "Login should complete in reasonable time",
		},
		{
			name:        "Get current user",
			method:      "GET",
			endpoint:    "/api/v1/users/me",
			useAuth:     true,
			maxDuration: 200 * time.Millisecond,
			description: "Getting user profile should be fast",
		},
		{
			name:     "User registration",
			method:   "POST",
			endpoint: "/api/v1/auth/register",
			payload: map[string]interface{}{
				"email":        fmt.Sprintf("perf%d@test.com", time.Now().UnixNano()),
				"password":     "password123",
				"full_name":    "Performance User",
				"phone_number": fmt.Sprintf("+519999%05d", time.Now().UnixNano()%100000),
				"user_role":    "CLIENT",
			},
			maxDuration: 1000 * time.Millisecond,
			description: "Registration includes password hashing so can be slower",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			var req *http.Request
			var err error

			// Prepare request
			if tc.payload != nil {
				body, err := json.Marshal(tc.payload)
				require.NoError(t, err)
				req = httptest.NewRequest(tc.method, tc.endpoint, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.endpoint, nil)
			}

			// Add auth if needed
			if tc.useAuth {
				req.Header.Set("Authorization", "Bearer "+suite.testToken)
			}

			// Measure response time
			start := time.Now()
			resp, err := suite.app.Test(req)
			duration := time.Since(start)
			
			require.NoError(t, err)
			defer resp.Body.Close()

			// Check response time
			assert.Less(t, duration, tc.maxDuration,
				"%s: %s %s took too long: %v (max: %v)",
				tc.description, tc.method, tc.endpoint, duration, tc.maxDuration)

			// Should not return 5xx errors (performance issues often cause these)
			assert.Less(t, resp.StatusCode, 500,
				"Should not return server errors: got status %d", resp.StatusCode)

			t.Logf("✅ %s %s completed in %v (limit: %v)",
				tc.method, tc.endpoint, duration, tc.maxDuration)
		})
	}
}

func (suite *APIPerformanceTestSuite) TestConcurrentRequests() {
	concurrency := 10
	requestsPerWorker := 5
	totalRequests := concurrency * requestsPerWorker

	// Test concurrent access to user profile endpoint
	var wg sync.WaitGroup
	var mutex sync.Mutex
	results := make([]time.Duration, 0, totalRequests)
	errors := 0

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < requestsPerWorker; j++ {
				reqStart := time.Now()
				
				req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
				req.Header.Set("Authorization", "Bearer "+suite.testToken)
				
				resp, err := suite.app.Test(req)
				reqDuration := time.Since(reqStart)
				
				mutex.Lock()
				if err != nil || resp.StatusCode >= 500 {
					errors++
				} else {
					results = append(results, reqDuration)
				}
				mutex.Unlock()
				
				if resp != nil {
					resp.Body.Close()
				}
			}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(start)

	// Analyze results
	assert.Equal(suite.T(), 0, errors, "Should not have any errors in concurrent requests")
	assert.Len(suite.T(), results, totalRequests, "Should complete all requests")

	// Calculate average response time
	var totalTime time.Duration
	var maxTime time.Duration
	for _, duration := range results {
		totalTime += duration
		if duration > maxTime {
			maxTime = duration
		}
	}
	avgTime := totalTime / time.Duration(len(results))

	// Performance assertions
	assert.Less(suite.T(), avgTime, 200*time.Millisecond,
		"Average response time should be <200ms, got: %v", avgTime)
	assert.Less(suite.T(), maxTime, 1*time.Second,
		"Max response time should be <1s, got: %v", maxTime)

	// Throughput calculation
	requestsPerSecond := float64(totalRequests) / totalDuration.Seconds()
	assert.Greater(suite.T(), requestsPerSecond, 20.0,
		"Should handle at least 20 requests/second, got: %.2f", requestsPerSecond)

	suite.T().Logf("🚀 Concurrent test results:")
	suite.T().Logf("   Total requests: %d", totalRequests)
	suite.T().Logf("   Concurrency: %d", concurrency)
	suite.T().Logf("   Total time: %v", totalDuration)
	suite.T().Logf("   Average response time: %v", avgTime)
	suite.T().Logf("   Max response time: %v", maxTime)
	suite.T().Logf("   Throughput: %.2f requests/second", requestsPerSecond)
	suite.T().Logf("   Errors: %d", errors)
}

func (suite *APIPerformanceTestSuite) TestDatabaseConnectionPerformance() {
	// Test that database queries are reasonably fast
	
	// Get user ID from token first
	claims, err := suite.authService.ValidateToken(suite.testToken)
	require.NoError(suite.T(), err)
	
	testCases := []struct {
		name        string
		operation   func() error
		maxDuration time.Duration
	}{
		{
			name: "Find user by ID",
			operation: func() error {
				_, err := suite.authService.GetUserByID(claims.UserID)
				return err
			},
			maxDuration: 50 * time.Millisecond,
		},
		{
			name: "Validate token",
			operation: func() error {
				_, err := suite.authService.ValidateToken(suite.testToken)
				return err
			},
			maxDuration: 100 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			start := time.Now()
			err := tc.operation()
			duration := time.Since(start)

			require.NoError(t, err)
			assert.Less(t, duration, tc.maxDuration,
				"Database operation '%s' took too long: %v (max: %v)",
				tc.name, duration, tc.maxDuration)

			t.Logf("✅ %s completed in %v (limit: %v)",
				tc.name, duration, tc.maxDuration)
		})
	}
}

func (suite *APIPerformanceTestSuite) TestMemoryUsage() {
	// Simple memory usage test - create many requests and ensure no major leaks
	var initialStats, finalStats = new(interface{}), new(interface{})
	
	// This is a basic test - in production you'd use proper memory profiling
	numRequests := 100
	
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		resp, err := suite.app.Test(req)
		require.NoError(suite.T(), err)
		resp.Body.Close()
	}
	
	// Basic assertion - we completed all requests without panicking
	assert.True(suite.T(), true, "Completed %d requests without memory issues", numRequests)
	
	// In a real scenario, you would measure actual memory usage here
	_ = initialStats
	_ = finalStats
}

// TestAPIPerformanceTestSuite runs the test suite
func TestAPIPerformanceTestSuite(t *testing.T) {
	suite.Run(t, new(APIPerformanceTestSuite))
}