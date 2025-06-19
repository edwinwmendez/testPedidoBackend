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
func SetupRoutes(app *fiber.App, authService auth.Service, userService *services.UserService, productService *services.ProductService, orderService *services.OrderService) {
	// Crear grupo de rutas para API v1
	api := app.Group("/api/v1")

	// Middlewares de autenticación
	authMiddleware := middlewares.AuthMiddleware(authService)
	adminOnly := middlewares.RequireRole(models.UserRoleAdmin)
	repartidorOrAdmin := middlewares.RequireRole(models.UserRoleRepartidor, models.UserRoleAdmin)

	// Rutas de autenticación
	authHandler := handlers.NewAuthHandler(authService)
	authHandler.RegisterRoutes(api)

	// Rutas de usuarios
	userHandler := handlers.NewUserHandler(userService)
	userHandler.RegisterRoutes(api, authMiddleware, adminOnly)

	// Rutas de productos
	productHandler := handlers.NewProductHandler(productService)
	productHandler.RegisterRoutes(api, authMiddleware, adminOnly)

	// Rutas de pedidos
	orderHandler := handlers.NewOrderHandler(orderService, authService)
	orderHandler.RegisterRoutes(api, authMiddleware, adminOnly, repartidorOrAdmin)

	// Ruta de salud
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "API funcionando correctamente",
		})
	})
}
