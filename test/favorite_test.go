package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFavoriteEndpoints tests all favorite-related endpoints
func TestFavoriteEndpoints(t *testing.T) {
	// Setup test environment
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create test user and product
	user := createTestUser(t, app, models.UserRoleClient)
	product := createTestProduct(t, app, user.Token)

	t.Run("Add favorite", func(t *testing.T) {
		testAddFavorite(t, app, user.Token, product.ProductID)
	})

	t.Run("Get favorite status", func(t *testing.T) {
		testGetFavoriteStatus(t, app, user.Token, product.ProductID, true)
	})

	t.Run("Get user favorites", func(t *testing.T) {
		testGetUserFavorites(t, app, user.Token)
	})

	t.Run("Toggle favorite (remove)", func(t *testing.T) {
		testToggleFavorite(t, app, user.Token, product.ProductID, false)
	})

	t.Run("Toggle favorite (add again)", func(t *testing.T) {
		testToggleFavorite(t, app, user.Token, product.ProductID, true)
	})

	t.Run("Remove favorite", func(t *testing.T) {
		testRemoveFavorite(t, app, user.Token, product.ProductID)
	})

	t.Run("Get favorite stats", func(t *testing.T) {
		testGetFavoriteStats(t, app, user.Token)
	})

	t.Run("Bulk check favorites", func(t *testing.T) {
		testBulkCheckFavorites(t, app, user.Token, []string{product.ProductID.String()})
	})
}

// TestFavoritePermissions tests role-based access control
func TestFavoritePermissions(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create users with different roles
	client := createTestUser(t, app, models.UserRoleClient)
	admin := createTestUser(t, app, models.UserRoleAdmin)
	product := createTestProduct(t, app, admin.Token)

	t.Run("Client can access basic endpoints", func(t *testing.T) {
		testAddFavorite(t, app, client.Token, product.ProductID)
		testGetFavoriteStatus(t, app, client.Token, product.ProductID, true)
		testGetUserFavorites(t, app, client.Token)
		testGetFavoriteStats(t, app, client.Token)
	})

	t.Run("Only admin can access most favorited", func(t *testing.T) {
		// Client should be forbidden
		req := httptest.NewRequest("GET", "/api/v1/favorites/most-favorited", nil)
		req.Header.Set("Authorization", "Bearer "+client.Token)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// Admin should succeed
		req = httptest.NewRequest("GET", "/api/v1/favorites/most-favorited", nil)
		req.Header.Set("Authorization", "Bearer "+admin.Token)
		
		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestFavoriteValidation tests input validation
func TestFavoriteValidation(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	user := createTestUser(t, app, models.UserRoleClient)

	t.Run("Invalid product ID", func(t *testing.T) {
		request := map[string]interface{}{
			"product_id": "invalid-uuid",
		}
		
		body, _ := json.Marshal(request)
		req := httptest.NewRequest("POST", "/api/v1/favorites", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+user.Token)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Empty product ID", func(t *testing.T) {
		request := map[string]interface{}{
			"product_id": "",
		}
		
		body, _ := json.Marshal(request)
		req := httptest.NewRequest("POST", "/api/v1/favorites", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+user.Token)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Unauthorized access", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/favorites", nil)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// Helper functions

func testAddFavorite(t *testing.T, app *fiber.App, token string, productID uuid.UUID) {
	request := map[string]interface{}{
		"product_id": productID.String(),
	}
	
	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/v1/favorites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var response models.FavoriteActionResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.True(t, response.IsFavorite)
	assert.Equal(t, productID.String(), response.ProductID)
}

func testRemoveFavorite(t *testing.T, app *fiber.App, token string, productID uuid.UUID) {
	request := map[string]interface{}{
		"product_id": productID.String(),
	}
	
	body, _ := json.Marshal(request)
	req := httptest.NewRequest("DELETE", "/api/v1/favorites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var response models.FavoriteActionResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.IsFavorite)
	assert.Equal(t, productID.String(), response.ProductID)
}

func testToggleFavorite(t *testing.T, app *fiber.App, token string, productID uuid.UUID, expectedState bool) {
	request := map[string]interface{}{
		"product_id": productID.String(),
	}
	
	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/v1/favorites/toggle", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var response models.FavoriteActionResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, expectedState, response.IsFavorite)
	assert.Equal(t, productID.String(), response.ProductID)
}

func testGetFavoriteStatus(t *testing.T, app *fiber.App, token string, productID uuid.UUID, expectedStatus bool) {
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/favorites/status/%s", productID.String()), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var response models.FavoriteStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, expectedStatus, response.IsFavorite)
	assert.Equal(t, productID.String(), response.ProductID)
}

func testGetUserFavorites(t *testing.T, app *fiber.App, token string) {
	req := httptest.NewRequest("GET", "/api/v1/favorites?page=1&limit=20", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var response models.FavoritesListResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, response.Total, 0)
	assert.GreaterOrEqual(t, response.Page, 1)
	assert.GreaterOrEqual(t, response.Limit, 1)
}

func testGetFavoriteStats(t *testing.T, app *fiber.App, token string) {
	req := httptest.NewRequest("GET", "/api/v1/favorites/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response, "total_favorites")
	assert.Contains(t, response, "user_id")
}

func testBulkCheckFavorites(t *testing.T, app *fiber.App, token string, productIDs []string) {
	body, _ := json.Marshal(productIDs)
	req := httptest.NewRequest("POST", "/api/v1/favorites/bulk-check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var response map[string]bool
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	
	for _, productID := range productIDs {
		_, exists := response[productID]
		assert.True(t, exists, "Product ID %s should be in response", productID)
	}
}