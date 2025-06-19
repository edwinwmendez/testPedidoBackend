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
	"fmt"
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

// UserHandlerTestSuite defines the test suite for user endpoints
type UserHandlerTestSuite struct {
	suite.Suite
	app            *fiber.App
	db             *gorm.DB
	authService    auth.Service
	userService    *services.UserService
	productService *services.ProductService
	orderService   *services.OrderService
	config         *config.Config
	// Test users
	clientUser     *models.User
	repartidorUser *models.User
	adminUser      *models.User
	// Test tokens
	clientToken    string
	repartidorToken string
	adminToken     string
}

// SetupSuite runs once before the test suite
func (suite *UserHandlerTestSuite) SetupSuite() {
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
	
	// Drop any incorrect foreign key constraints that GORM might have created
	suite.db.Exec("ALTER TABLE products DROP CONSTRAINT IF EXISTS fk_order_items_product")
	suite.db.Exec("ALTER TABLE order_items DROP CONSTRAINT IF EXISTS fk_order_items_product")
	
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
func (suite *UserHandlerTestSuite) SetupTest() {
	// Clean database before each test
	suite.db.Where("1 = 1").Delete(&models.User{})
	
	// Also clean any seed data that might have been inserted by migrations
	suite.db.Exec("DELETE FROM products WHERE name LIKE 'Balón de Gas%'")
	suite.db.Exec("DELETE FROM users WHERE email = 'admin@exactogas.com'")
	
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
	
	// Generate tokens for test users
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
func (suite *UserHandlerTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Where("1 = 1").Delete(&models.User{})
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
}

func (suite *UserHandlerTestSuite) TestGetCurrentUser_Success() {
	// Test getting current user profile
	req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+suite.clientToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	
	// Verify user information
	assert.Equal(suite.T(), suite.clientUser.UserID.String(), response["user_id"])
	assert.Equal(suite.T(), suite.clientUser.Email, response["email"])
	assert.Equal(suite.T(), suite.clientUser.FullName, response["full_name"])
	assert.Equal(suite.T(), suite.clientUser.PhoneNumber, response["phone_number"])
	assert.Equal(suite.T(), string(suite.clientUser.UserRole), response["user_role"])
}

func (suite *UserHandlerTestSuite) TestGetCurrentUser_WithoutAuthentication() {
	// Test getting current user without authentication
	req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	// No Authorization header
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (suite *UserHandlerTestSuite) TestUpdateCurrentUser_Success() {
	// Test updating user profile
	updatePayload := map[string]interface{}{
		"full_name":    "Updated Client Name",
		"phone_number": "+51999888777",
	}
	
	body, err := json.Marshal(updatePayload)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest("PUT", "/api/v1/users/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.clientToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	
	// Verify updated information
	assert.Equal(suite.T(), suite.clientUser.UserID.String(), response["user_id"])
	assert.Equal(suite.T(), suite.clientUser.Email, response["email"]) // Email should not change
	assert.Equal(suite.T(), "Updated Client Name", response["full_name"])
	assert.Equal(suite.T(), "+51999888777", response["phone_number"])
	assert.Equal(suite.T(), string(suite.clientUser.UserRole), response["user_role"])
}

func (suite *UserHandlerTestSuite) TestUpdateCurrentUser_PartialUpdate() {
	// Test updating only one field
	updatePayload := map[string]interface{}{
		"full_name": "New Name Only",
	}
	
	body, err := json.Marshal(updatePayload)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest("PUT", "/api/v1/users/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.repartidorToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	
	// Verify only full_name was updated, phone_number remains the same
	assert.Equal(suite.T(), "New Name Only", response["full_name"])
	assert.Equal(suite.T(), suite.repartidorUser.PhoneNumber, response["phone_number"])
}

func (suite *UserHandlerTestSuite) TestUpdateCurrentUser_WithoutAuthentication() {
	// Test updating profile without authentication
	updatePayload := map[string]interface{}{
		"full_name": "Unauthorized Update",
	}
	
	body, err := json.Marshal(updatePayload)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest("PUT", "/api/v1/users/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (suite *UserHandlerTestSuite) TestUpdateCurrentUser_InvalidJSON() {
	// Test updating with invalid JSON
	req := httptest.NewRequest("PUT", "/api/v1/users/me", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.clientToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	
	assert.Contains(suite.T(), response["error"], "Datos inválidos")
}

func (suite *UserHandlerTestSuite) TestGetAllUsers_AdminAccess() {
	// Test that admin can get all users
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var users []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&users)
	require.NoError(suite.T(), err)
	
	// Should have 3 users (client, repartidor, admin)
	assert.Len(suite.T(), users, 3)
}

func (suite *UserHandlerTestSuite) TestGetAllUsers_WithRoleFilter() {
	// Test getting users filtered by role
	req := httptest.NewRequest("GET", "/api/v1/users?role=CLIENT", nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var users []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&users)
	require.NoError(suite.T(), err)
	
	// Should have 1 client user
	assert.Len(suite.T(), users, 1)
	assert.Equal(suite.T(), "CLIENT", users[0]["user_role"])
}

func (suite *UserHandlerTestSuite) TestGetAllUsers_NonAdminAccess() {
	// Test that non-admin users cannot get all users
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+suite.clientToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)
}

func (suite *UserHandlerTestSuite) TestGetUserByID_AdminAccess() {
	// Test that admin can get specific user by ID
	req := httptest.NewRequest("GET", "/api/v1/users/"+suite.clientUser.UserID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var user map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&user)
	require.NoError(suite.T(), err)
	
	// Verify it's the correct user
	assert.Equal(suite.T(), suite.clientUser.UserID.String(), user["user_id"])
	assert.Equal(suite.T(), suite.clientUser.Email, user["email"])
}

func (suite *UserHandlerTestSuite) TestGetUserByID_NonAdminAccess() {
	// Test that non-admin users cannot get other users by ID
	req := httptest.NewRequest("GET", "/api/v1/users/"+suite.clientUser.UserID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+suite.repartidorToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)
}

func (suite *UserHandlerTestSuite) TestGetUserByID_InvalidID() {
	// Test getting user with invalid UUID
	req := httptest.NewRequest("GET", "/api/v1/users/invalid-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)
	
	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
}

func (suite *UserHandlerTestSuite) TestProfileUpdateWorkflow_AllRoles() {
	// Test profile update workflow for all user roles
	testCases := []struct {
		name  string
		token string
		user  *models.User
	}{
		{"Client Profile Update", suite.clientToken, suite.clientUser},
		{"Repartidor Profile Update", suite.repartidorToken, suite.repartidorUser},
		{"Admin Profile Update", suite.adminToken, suite.adminUser},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Step 1: Get current profile
			getReq := httptest.NewRequest("GET", "/api/v1/users/me", nil)
			getReq.Header.Set("Authorization", "Bearer "+tc.token)
			
			getResp, err := suite.app.Test(getReq)
			require.NoError(t, err)
			defer getResp.Body.Close()
			
			assert.Equal(t, http.StatusOK, getResp.StatusCode)
			
			var originalProfile map[string]interface{}
			err = json.NewDecoder(getResp.Body).Decode(&originalProfile)
			require.NoError(t, err)
			
			// Step 2: Update profile with unique values per test to avoid conflicts
			timestamp := time.Now().UnixNano()
			updatePayload := map[string]interface{}{
				"full_name":    fmt.Sprintf("Updated %s %d", tc.user.FullName, timestamp),
				"phone_number": fmt.Sprintf("+518%010d", timestamp%10000000000),
			}
			
			updateBody, err := json.Marshal(updatePayload)
			require.NoError(t, err)
			
			updateReq := httptest.NewRequest("PUT", "/api/v1/users/me", bytes.NewReader(updateBody))
			updateReq.Header.Set("Content-Type", "application/json")
			updateReq.Header.Set("Authorization", "Bearer "+tc.token)
			
			updateResp, err := suite.app.Test(updateReq)
			require.NoError(t, err)
			defer updateResp.Body.Close()
			
			// Check for error first to debug
			if updateResp.StatusCode != http.StatusOK {
				var errorResponse map[string]interface{}
				json.NewDecoder(updateResp.Body).Decode(&errorResponse)
				t.Logf("Update failed with status %d: %+v", updateResp.StatusCode, errorResponse)
			}
			
			assert.Equal(t, http.StatusOK, updateResp.StatusCode)
			
			var updatedProfile map[string]interface{}
			err = json.NewDecoder(updateResp.Body).Decode(&updatedProfile)
			require.NoError(t, err)
			
			// Step 3: Verify changes
			expectedFullName := fmt.Sprintf("Updated %s %d", tc.user.FullName, timestamp)
			expectedPhone := fmt.Sprintf("+518%010d", timestamp%10000000000)
			
			assert.Equal(t, expectedFullName, updatedProfile["full_name"])
			assert.Equal(t, expectedPhone, updatedProfile["phone_number"])
			assert.Equal(t, originalProfile["email"], updatedProfile["email"]) // Email should not change
			assert.Equal(t, originalProfile["user_role"], updatedProfile["user_role"]) // Role should not change
			
			// Step 4: Wait a moment and verify persistence by getting profile again
			time.Sleep(100 * time.Millisecond) // Small delay to ensure database consistency
			
			finalGetReq := httptest.NewRequest("GET", "/api/v1/users/me", nil)
			finalGetReq.Header.Set("Authorization", "Bearer "+tc.token)
			
			finalGetResp, err := suite.app.Test(finalGetReq)
			require.NoError(t, err)
			defer finalGetResp.Body.Close()
			
			assert.Equal(t, http.StatusOK, finalGetResp.StatusCode)
			
			var finalProfile map[string]interface{}
			err = json.NewDecoder(finalGetResp.Body).Decode(&finalProfile)
			require.NoError(t, err)
			
			// Verify changes were persisted
			assert.Equal(t, expectedFullName, finalProfile["full_name"])
			assert.Equal(t, expectedPhone, finalProfile["phone_number"])
		})
	}
}

// TestUserHandlerTestSuite runs the test suite
func TestUserHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UserHandlerTestSuite))
}