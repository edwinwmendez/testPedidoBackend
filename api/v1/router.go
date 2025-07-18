package v1

import (
	"backend/api/v1/handlers"
	"backend/api/v1/middlewares"
	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configura todas las rutas de la API v1
func SetupRoutes(app *fiber.App, authService auth.Service, userService *services.UserService, productService *services.ProductService, categoryService *services.CategoryService, orderService *services.OrderService, productRatingService *services.ProductRatingService, favoriteService *services.FavoriteService, offerService services.OfferService) {
	// Crear grupo de rutas para API v1
	api := app.Group("/api/v1")

	// Middlewares de autenticación
	authMiddleware := middlewares.AuthMiddleware(authService)
	adminOnly := middlewares.RequireRole(models.UserRoleAdmin)
	repartidorOrAdmin := middlewares.RequireRole(models.UserRoleRepartidor, models.UserRoleAdmin)

	// Rutas de autenticación
	authHandler := handlers.NewAuthHandler(authService)
	authHandler.RegisterRoutes(api, authMiddleware, adminOnly)

	// Rutas de usuarios
	userHandler := handlers.NewUserHandler(userService)
	userHandler.RegisterRoutes(api, authMiddleware, adminOnly)

	// Rutas de calificaciones de productos (DEBE ir ANTES que las rutas de productos para evitar conflictos)
	productRatingHandler := handlers.NewProductRatingHandler(productRatingService)
	productRatingHandler.RegisterRoutes(api, authMiddleware)

	// Rutas de ofertas (DEBE ir ANTES que las rutas de productos para evitar conflictos de rutas)
	offerHandler := handlers.NewOfferHandler(offerService)
	setupOfferRoutes(api, offerHandler, authMiddleware, adminOnly)

	// Rutas de productos (DEBE ir DESPUÉS de las ofertas para evitar conflictos)
	productHandler := handlers.NewProductHandler(productService)
	productHandler.RegisterRoutes(api, authMiddleware, adminOnly)

	// Rutas de categorías
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	categoryHandler.RegisterRoutes(api, authMiddleware, adminOnly)

	// Rutas de pedidos
	orderHandler := handlers.NewOrderHandler(orderService, authService)
	orderHandler.RegisterRoutes(api, authMiddleware, adminOnly, repartidorOrAdmin)

	// Rutas de favoritos
	favoriteHandler := handlers.NewFavoriteHandler(favoriteService)
	favoriteHandler.RegisterRoutes(api, authMiddleware, adminOnly)

	// Ruta de salud
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "API funcionando correctamente",
		})
	})

}

// setupOfferRoutes configura las rutas específicas para ofertas
func setupOfferRoutes(api fiber.Router, offerHandler *handlers.OfferHandler, authMiddleware, adminOnly fiber.Handler) {
	// Rutas públicas de ofertas (sin autenticación)
	api.Get("/products/offers", offerHandler.GetProductOffers)   // GET /products/offers
	api.Get("/products/:id/offer", offerHandler.GetProductOffer) // GET /products/:id/offer

	// Rutas administrativas de ofertas (requieren autenticación de admin)
	adminOffers := api.Group("/admin", authMiddleware, adminOnly)

	// CRUD básico de ofertas
	adminOffers.Post("/offers", offerHandler.CreateOffer)       // POST /admin/offers
	adminOffers.Get("/offers/:id", offerHandler.GetOffer)       // GET /admin/offers/:id
	adminOffers.Put("/offers/:id", offerHandler.UpdateOffer)    // PUT /admin/offers/:id
	adminOffers.Delete("/offers/:id", offerHandler.DeleteOffer) // DELETE /admin/offers/:id

	// Rutas convenientes para gestionar ofertas por producto
	adminOffers.Post("/products/:id/offer", offerHandler.SetProductOffer)      // POST /admin/products/:id/offer
	adminOffers.Delete("/products/:id/offer", offerHandler.RemoveProductOffer) // DELETE /admin/products/:id/offer
}
