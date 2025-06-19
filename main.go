package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	v1 "backend/api/v1"
	"backend/config"
	"backend/database"
	_ "backend/docs" // Import para Swagger (solo si existe)
	"backend/internal/auth"
	"backend/internal/repositories"
	"backend/internal/services"
	"backend/internal/ws"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"github.com/swaggo/swag"
)

// setupSwagger configura la documentación Swagger de forma segura
func setupSwagger() {
	// Configurar información básica de Swagger
	swag.Register("swagger", &swag.Spec{
		Version:          "1.0",
		Host:             getSwaggerHost(),
		BasePath:         "/api/v1",
		Schemes:          getSwaggerSchemes(),
		Title:            "ExactoGas API",
		Description:      "API para la aplicación de gestión de pedidos de gas a domicilio ExactoGas",
		InfoInstanceName: "swagger",
		SwaggerTemplate:  "", // Se carga automáticamente si existe
	})
}

// getSwaggerHost retorna el host apropiado para Swagger
func getSwaggerHost() string {
	if os.Getenv("RENDER") == "true" {
		return "" // Render maneja esto automáticamente
	}
	return "localhost:8080"
}

// getSwaggerSchemes retorna los esquemas apropiados para Swagger
func getSwaggerSchemes() []string {
	if os.Getenv("RENDER") == "true" {
		return []string{"https"}
	}
	return []string{"http"}
}

// ... comentarios Swagger ...
func main() {
	// Inicializar configuración de Swagger (sin dependencia de docs package)
	setupSwagger()

	// Cargar configuración
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error al cargar la configuración: %v", err)
	}

	// Conectar a la base de datos
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Error al conectar a la base de datos: %v", err)
	}
	log.Println("Conexión a la base de datos establecida correctamente")

	// Inicializar repositorios
	userRepo := repositories.NewUserRepository(db)
	productRepo := repositories.NewProductRepository(db)
	orderRepo := repositories.NewOrderRepository(db)

	// Inicializar servicios
	authService := auth.NewService(db, cfg)
	userService := services.NewUserService(userRepo)
	productService := services.NewProductService(productRepo)

	// Inicializar servicio de notificaciones (opcional para el MVP)
	var notificationService *services.NotificationService
	// notificationService, err = services.NewNotificationService(userRepo, cfg.Firebase.CredentialsFile)
	// if err != nil {
	//     log.Printf("Advertencia: No se pudo inicializar el servicio de notificaciones: %v", err)
	//     log.Println("Las notificaciones estarán desactivadas")
	// }

	// Inicializar servicio de pedidos
	hub := ws.NewHub()
	go hub.Run()
	orderService := services.NewOrderService(orderRepo, userRepo, productRepo, notificationService, cfg, hub)

	// Crear la aplicación Fiber
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	})

	// Registrar middlewares globales
	app.Use(logger.New())
	app.Use(recover.New())

	// Configurar CORS para permitir conexiones desde aplicaciones móviles
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		AllowCredentials: false,
		ExposeHeaders:    "Content-Length, Content-Type",
	}))

	// Configurar rutas de la API
	v1.SetupRoutes(app, authService, userService, productService, orderService)

	// Endpoint de salud para verificar que el servidor está funcionando
	app.Get("/api/v1/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "ExactoGas API está funcionando correctamente",
		})
	})

	// --- INICIO WEBSOCKET ---
	app.Get("/ws/notifications", ws.WebSocketHandler(hub, cfg))
	// --- FIN WEBSOCKET ---

	// ... Swagger y archivos estáticos ...
	// Obtener la ruta absoluta del directorio de documentación
	docsDir, err := filepath.Abs("./docs")
	if err != nil {
		log.Printf("Error al obtener la ruta absoluta de la documentación: %v", err)
		docsDir = "./docs"
	}

	app.Get("/swagger/*", fiberSwagger.FiberWrapHandler())
	app.Get("/swagger", func(c *fiber.Ctx) error {
		return c.Redirect("/swagger/index.html")
	})
	app.Get("/swagger/doc.json", func(c *fiber.Ctx) error {
		// Intentar leer la documentación Swagger desde archivo si existe
		if _, err := os.Stat("./docs/swagger.json"); err == nil {
			return c.SendFile("./docs/swagger.json")
		}
		// Fallback a respuesta básica
		return c.JSON(map[string]interface{}{
			"swagger": "2.0",
			"info": map[string]interface{}{
				"title":       "ExactoGas API",
				"description": "API para la aplicación de gestión de pedidos de gas a domicilio ExactoGas",
				"version":     "1.0",
			},
			"host":     getSwaggerHost(),
			"basePath": "/api/v1",
			"schemes":  getSwaggerSchemes(),
		})
	})
	app.Get("/swagger/index.html", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.SendFile(filepath.Join(docsDir, "index.html"))
	})
	app.Get("/swagger/swagger-ui.css", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/css")
		return c.SendFile(filepath.Join(docsDir, "swagger-ui.css"))
	})
	app.Get("/swagger/swagger-ui-bundle.js", func(c *fiber.Ctx) error {
		return c.Redirect("https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/swagger-ui-bundle.js")
	})
	app.Get("/swagger/swagger-ui-standalone-preset.js", func(c *fiber.Ctx) error {
		return c.Redirect("https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/swagger-ui-standalone-preset.js")
	})
	app.Get("/swagger/favicon-32x32.png", func(c *fiber.Ctx) error {
		return c.Redirect("https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/favicon-32x32.png")
	})
	app.Get("/swagger/favicon-16x16.png", func(c *fiber.Ctx) error {
		return c.Redirect("https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/favicon-16x16.png")
	})

	// Manejar señales de apagado
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Apagando servidor...")
		if err := app.Shutdown(); err != nil {
			log.Fatalf("Error al apagar servidor: %v", err)
		}
	}()

	// Iniciar servidor
	port := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Servidor iniciado en http://localhost%s", port)
	log.Printf("Documentación Swagger disponible en http://localhost%s/swagger", port)
	if err := app.Listen(port); err != nil {
		log.Fatalf("Error al iniciar servidor: %v", err)
	}
}
