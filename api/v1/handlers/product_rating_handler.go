package handlers

import (
	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ProductRatingHandler maneja las peticiones HTTP relacionadas con calificaciones
type ProductRatingHandler struct {
	ratingService *services.ProductRatingService
}

// NewProductRatingHandler crea un nuevo handler de calificaciones
func NewProductRatingHandler(ratingService *services.ProductRatingService) *ProductRatingHandler {
	return &ProductRatingHandler{
		ratingService: ratingService,
	}
}

// CreateRating crea una nueva calificación
// @Summary Crear calificación
// @Description Permite a un usuario calificar un producto
// @Tags calificaciones
// @Accept json
// @Produce json
// @Param rating body models.CreateRatingRequest true "Datos de la calificación"
// @Security BearerAuth
// @Success 201 {object} models.RatingResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ratings [post]
func (h *ProductRatingHandler) CreateRating(c *fiber.Ctx) error {
	// Obtener claims del usuario del contexto (agregado por middleware de auth)
	claims, ok := c.Locals("user").(*auth.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No se encontró información de autenticación",
		})
	}
	userID := claims.UserID.String()

	var req models.CreateRatingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de datos inválido",
		})
	}

	// Validar request
	if req.Rating < 1 || req.Rating > 5 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "La calificación debe estar entre 1 y 5",
		})
	}

	response, err := h.ratingService.Create(&req, userID)
	if err != nil {
		switch err {
		case services.ErrProductNotFoundService:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Producto no encontrado",
			})
		case services.ErrRatingAlreadyExists:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Ya has calificado este producto",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al crear la calificación",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// GetProductRatings obtiene todas las calificaciones de un producto
// @Summary Obtener calificaciones de producto
// @Description Obtiene todas las calificaciones de un producto específico
// @Tags calificaciones
// @Accept json
// @Produce json
// @Param id path string true "ID del producto"
// @Success 200 {array} models.RatingResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{id}/ratings [get]
func (h *ProductRatingHandler) GetProductRatings(c *fiber.Ctx) error {
	productID := c.Params("id")

	// Validar UUID
	if _, err := uuid.Parse(productID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de producto inválido",
		})
	}

	ratings, err := h.ratingService.GetByProduct(productID)
	if err != nil {
		if err == services.ErrProductNotFoundService {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Producto no encontrado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener calificaciones",
		})
	}

	return c.JSON(ratings)
}

// GetUserRating obtiene la calificación del usuario para un producto
// @Summary Obtener calificación del usuario
// @Description Obtiene la calificación del usuario autenticado para un producto específico
// @Tags calificaciones
// @Accept json
// @Produce json
// @Param id path string true "ID del producto"
// @Security BearerAuth
// @Success 200 {object} models.RatingResponse
// @Success 204 "Usuario no ha calificado este producto"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{id}/ratings/me [get]
func (h *ProductRatingHandler) GetUserRating(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*auth.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No se encontró información de autenticación",
		})
	}
	userID := claims.UserID.String()
	productID := c.Params("id")

	// Validar UUID
	if _, err := uuid.Parse(productID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de producto inválido",
		})
	}

	rating, err := h.ratingService.GetUserRatingForProduct(productID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener calificación",
		})
	}

	if rating == nil {
		return c.SendStatus(fiber.StatusNoContent)
	}

	return c.JSON(rating)
}

// UpdateRating actualiza una calificación existente
// @Summary Actualizar calificación
// @Description Permite a un usuario actualizar su calificación de un producto
// @Tags calificaciones
// @Accept json
// @Produce json
// @Param rating body models.CreateRatingRequest true "Datos de la calificación actualizada"
// @Security BearerAuth
// @Success 200 {object} models.RatingResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ratings [put]
func (h *ProductRatingHandler) UpdateRating(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*auth.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No se encontró información de autenticación",
		})
	}
	userID := claims.UserID.String()

	var req models.CreateRatingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de datos inválido",
		})
	}

	// Validar request
	if req.Rating < 1 || req.Rating > 5 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "La calificación debe estar entre 1 y 5",
		})
	}

	response, err := h.ratingService.Update(&req, userID)
	if err != nil {
		switch err {
		case services.ErrProductNotFoundService:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Producto no encontrado",
			})
		case services.ErrRatingNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "No tienes una calificación para este producto",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al actualizar la calificación",
			})
		}
	}

	return c.JSON(response)
}

// RegisterRoutes registra las rutas del handler en el router
func (h *ProductRatingHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	// Rutas que requieren autenticación SOLO (sin restricciones de rol)
	ratings := router.Group("/ratings", authMiddleware)
	ratings.Post("/", h.CreateRating)
	ratings.Put("/", h.UpdateRating)

	// Rutas públicas para calificaciones de productos (sin autenticación)
	router.Get("/products/:id/ratings", h.GetProductRatings)

	// Rutas autenticadas para calificaciones de productos (solo auth, sin roles)
	router.Get("/products/:id/ratings/me", authMiddleware, h.GetUserRating)
}
