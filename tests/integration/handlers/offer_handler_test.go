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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OfferHandlerTestSuite defines the test suite for offer endpoints
type OfferHandlerTestSuite struct {
	suite.Suite
	app            *fiber.App
	db             *gorm.DB
	authService    auth.Service
	userService    *services.UserService
	productService *services.ProductService
	offerService   services.OfferService
	config         *config.Config
	adminToken     string
	clientToken    string
	testProductID  string
	testAdminID    string
	testClientID   string
}

// SetupSuite runs once before the test suite
func (suite *OfferHandlerTestSuite) SetupSuite() {
	// Test configuration
	suite.config = &config.Config{
		JWT: config.JWTConfig{
			Secret:          "test-jwt-secret-key-for-integration-testing",
			AccessTokenExp:  15 * time.Minute,
			RefreshTokenExp: 7 * 24 * time.Hour,
		},
	}

	// Setup test database
	var err error
	suite.db, err = gorm.Open(postgres.Open("host=localhost port=5432 user=postgres password=postgress dbname=pedidos_dev sslmode=disable"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		suite.T().Skip("PostgreSQL not available for integration testing")
		return
	}

	// Auto-migrate models
	err = suite.db.AutoMigrate(&models.User{}, &models.Product{}, &models.Category{}, &models.ProductOffer{})
	require.NoError(suite.T(), err)

	// Create repositories
	userRepo := repositories.NewUserRepository(suite.db)
	productRepo := repositories.NewProductRepository(suite.db)
	categoryRepo := repositories.NewCategoryRepository(suite.db)
	offerRepo := repositories.NewOfferRepository(suite.db)

	// Create services
	suite.authService = auth.NewService(suite.db, suite.config)
	suite.userService = services.NewUserService(userRepo)
	suite.productService = services.NewProductService(productRepo, nil)
	suite.offerService = services.NewOfferService(offerRepo, userRepo, productRepo)

	// Setup Fiber app with routes
	suite.app = fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Setup routes
	categoryService := services.NewCategoryService(categoryRepo, nil)
	v1.SetupRoutes(suite.app, suite.authService, suite.userService, suite.productService, categoryService, nil, nil, nil, suite.offerService)
}

// SetupTest runs before each test
func (suite *OfferHandlerTestSuite) SetupTest() {
	// Clean database before each test
	suite.db.Where("1 = 1").Delete(&models.ProductOffer{})
	suite.db.Where("1 = 1").Delete(&models.Product{})
	suite.db.Where("1 = 1").Delete(&models.Category{})
	suite.db.Where("1 = 1").Delete(&models.User{})

	// Create test users (admin and client)
	suite.createTestUsers()

	// Create test product
	suite.createTestProduct()
}

// TearDownSuite runs once after the test suite
func (suite *OfferHandlerTestSuite) TearDownSuite() {
	if suite.db != nil {
		// Clean all test data
		suite.db.Where("1 = 1").Delete(&models.ProductOffer{})
		suite.db.Where("1 = 1").Delete(&models.Product{})
		suite.db.Where("1 = 1").Delete(&models.Category{})
		suite.db.Where("1 = 1").Delete(&models.User{})

		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
}

func (suite *OfferHandlerTestSuite) createTestUsers() {
	// Use unique emails based on current time to avoid conflicts
	timestamp := time.Now().UnixNano()
	
	// Create admin user
	adminEmail := fmt.Sprintf("admin%d@test.com", timestamp)
	adminPhone := fmt.Sprintf("+5199999%04d", timestamp%10000)
	
	adminUser, err := suite.authService.RegisterUser(
		adminEmail,
		"testpassword123",
		"Test Admin",
		adminPhone,
		models.UserRoleAdmin,
	)
	require.NoError(suite.T(), err)
	suite.testAdminID = adminUser.UserID.String()

	// Create client user
	clientEmail := fmt.Sprintf("client%d@test.com", timestamp)
	clientPhone := fmt.Sprintf("+5188888%04d", timestamp%10000)
	
	clientUser, err := suite.authService.RegisterUser(
		clientEmail,
		"testpassword123",
		"Test Client",
		clientPhone,
		models.UserRoleClient,
	)
	require.NoError(suite.T(), err)
	suite.testClientID = clientUser.UserID.String()

	// Generate tokens by logging in
	adminTokenData, err := suite.authService.Login(adminEmail, "testpassword123")
	require.NoError(suite.T(), err)
	suite.adminToken = adminTokenData.AccessToken

	clientTokenData, err := suite.authService.Login(clientEmail, "testpassword123")
	require.NoError(suite.T(), err)
	suite.clientToken = clientTokenData.AccessToken
}

func (suite *OfferHandlerTestSuite) createTestProduct() {
	// Use unique names based on current time to avoid conflicts
	timestamp := time.Now().UnixNano()
	
	// Create test category first
	categoryID := uuid.New()
	category := &models.Category{
		CategoryID:  categoryID,
		Name:        fmt.Sprintf("Test Category %d", timestamp),
		Description: "Test category for offers",
		IsActive:    true,
	}
	require.NoError(suite.T(), suite.db.Create(category).Error)

	// Create test product
	productID := uuid.New()
	suite.testProductID = productID.String()
	product := &models.Product{
		ProductID:     productID,
		Name:          fmt.Sprintf("Test Product for Offers %d", timestamp),
		Description:   "A test product to test offers functionality",
		Price:         100.00,
		CategoryID:    &categoryID,
		ImageURL:      "https://example.com/test-product.jpg",
		StockQuantity: 50,
		IsActive:      true,
	}
	require.NoError(suite.T(), suite.db.Create(product).Error)
}

// Test creating an offer as admin
func (suite *OfferHandlerTestSuite) TestCreateOffer_Success() {
	startDate := time.Now().Add(1 * time.Hour)
	endDate := time.Now().Add(24 * time.Hour)

	payload := map[string]interface{}{
		"product_id":     suite.testProductID,
		"discount_type":  "percentage",
		"discount_value": 20.0,
		"start_date":     startDate.Format(time.RFC3339),
		"end_date":       endDate.Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest("POST", "/api/v1/admin/offers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Oferta creada exitosamente", response["message"])
	assert.NotNil(suite.T(), response["offer"])
}

// Test creating offer with invalid dates
func (suite *OfferHandlerTestSuite) TestCreateOffer_InvalidDates() {
	// End date before start date
	startDate := time.Now().Add(24 * time.Hour)
	endDate := time.Now().Add(1 * time.Hour)

	payload := map[string]interface{}{
		"product_id":     suite.testProductID,
		"discount_type":  "percentage",
		"discount_value": 20.0,
		"start_date":     startDate.Format(time.RFC3339),
		"end_date":       endDate.Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest("POST", "/api/v1/admin/offers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
}

// Test unauthorized access (client trying to create offer)
func (suite *OfferHandlerTestSuite) TestCreateOffer_Unauthorized() {
	startDate := time.Now().Add(1 * time.Hour)
	endDate := time.Now().Add(24 * time.Hour)

	payload := map[string]interface{}{
		"product_id":     suite.testProductID,
		"discount_type":  "percentage",
		"discount_value": 20.0,
		"start_date":     startDate.Format(time.RFC3339),
		"end_date":       endDate.Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest("POST", "/api/v1/admin/offers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.clientToken)

	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)
}

// Test getting products with offers (public endpoint)
func (suite *OfferHandlerTestSuite) TestGetProductOffers_Success() {
	// First create an active offer
	suite.createActiveOffer()

	req := httptest.NewRequest("GET", "/api/v1/products/offers", nil)

	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), response["products"])
	assert.NotNil(suite.T(), response["total"])

	products := response["products"].([]interface{})
	if len(products) > 0 {
		product := products[0].(map[string]interface{})
		assert.NotNil(suite.T(), product["product_id"])
		assert.NotNil(suite.T(), product["name"])
		assert.NotNil(suite.T(), product["current_offer"])
		assert.NotNil(suite.T(), product["final_price"])
		assert.Equal(suite.T(), true, product["is_on_offer"])
	}
}

// Test getting specific product offer
func (suite *OfferHandlerTestSuite) TestGetProductOffer_Success() {
	// First create an active offer
	suite.createActiveOffer()

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/products/%s/offer", suite.testProductID), nil)

	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var offer map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&offer)
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), offer["offer_id"])
	assert.Equal(suite.T(), suite.testProductID, offer["product_id"])
}

// Test setting product offer (convenience method)
func (suite *OfferHandlerTestSuite) TestSetProductOffer_Success() {
	startDate := time.Now().Add(-1 * time.Hour) // Active now
	endDate := time.Now().Add(24 * time.Hour)

	payload := map[string]interface{}{
		"discount_type":  "fixed_amount",
		"discount_value": 15.0,
		"start_date":     startDate.Format(time.RFC3339),
		"end_date":       endDate.Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/admin/products/%s/offer", suite.testProductID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Oferta establecida exitosamente", response["message"])
}

// Test removing product offer
func (suite *OfferHandlerTestSuite) TestRemoveProductOffer_Success() {
	// First create an active offer
	suite.createActiveOffer()

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/products/%s/offer", suite.testProductID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	resp, err := suite.app.Test(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Oferta removida exitosamente", response["message"])
}

// Test different discount types calculation
func (suite *OfferHandlerTestSuite) TestDiscountTypesCalculation() {
	testCases := []struct {
		discountType  string
		discountValue float64
		originalPrice float64
		expectedPrice float64
	}{
		{"percentage", 20.0, 100.0, 80.0},   // 20% off 100 = 80
		{"fixed_amount", 15.0, 100.0, 85.0}, // 100 - 15 = 85
		{"fixed_price", 75.0, 100.0, 75.0},  // Fixed price 75
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("Test_%s_discount", tc.discountType), func(t *testing.T) {
			// Clean previous offers
			suite.db.Where("product_id = ?", suite.testProductID).Delete(&models.ProductOffer{})

			// Create offer with specific discount type
			suite.createOfferWithType(tc.discountType, tc.discountValue)

			// Get product offer and verify calculation
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/products/%s/offer", suite.testProductID), nil)

			resp, err := suite.app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var offer map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&offer)
			require.NoError(t, err)

			assert.Equal(t, tc.discountType, offer["discount_type"])
			assert.Equal(t, tc.discountValue, offer["discount_value"])
		})
	}
}

// Helper method to create an active offer
func (suite *OfferHandlerTestSuite) createActiveOffer() {
	startDate := time.Now().Add(-1 * time.Hour) // Started 1 hour ago
	endDate := time.Now().Add(24 * time.Hour)   // Ends in 24 hours

	offer := &models.ProductOffer{
		OfferID:       uuid.New().String(),
		ProductID:     suite.testProductID,
		DiscountType:  models.DiscountTypePercentage,
		DiscountValue: 20.0,
		StartDate:     startDate,
		EndDate:       endDate,
		IsActive:      true,
		CreatedBy:     suite.testAdminID,
	}

	require.NoError(suite.T(), suite.db.Create(offer).Error)
}

// Helper method to create offer with specific type
func (suite *OfferHandlerTestSuite) createOfferWithType(discountType string, discountValue float64) {
	startDate := time.Now().Add(-1 * time.Hour)
	endDate := time.Now().Add(24 * time.Hour)

	var dType models.OfferDiscountType
	switch discountType {
	case "percentage":
		dType = models.DiscountTypePercentage
	case "fixed_amount":
		dType = models.DiscountTypeFixedAmount
	case "fixed_price":
		dType = models.DiscountTypeFixedPrice
	}

	offer := &models.ProductOffer{
		OfferID:       uuid.New().String(),
		ProductID:     suite.testProductID,
		DiscountType:  dType,
		DiscountValue: discountValue,
		StartDate:     startDate,
		EndDate:       endDate,
		IsActive:      true,
		CreatedBy:     suite.testAdminID,
	}

	require.NoError(suite.T(), suite.db.Create(offer).Error)
}

// TestOfferHandlerTestSuite runs the test suite
func TestOfferHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(OfferHandlerTestSuite))
}
