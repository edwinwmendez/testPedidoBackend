package handlers

import (
	"strconv"
	"time"

	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// OfferHandler maneja los endpoints HTTP para ofertas
type OfferHandler struct {
	offerService services.OfferService
}

// NewOfferHandler crea una nueva instancia del handler
func NewOfferHandler(offerService services.OfferService) *OfferHandler {
	return &OfferHandler{
		offerService: offerService,
	}
}

// CreateOfferRequest estructura para crear ofertas
type CreateOfferRequest struct {
	ProductID     string                   `json:"product_id" validate:"required,uuid"`
	DiscountType  models.OfferDiscountType `json:"discount_type" validate:"required,oneof=percentage fixed_amount fixed_price"`
	DiscountValue float64                  `json:"discount_value" validate:"required,gt=0"`
	StartDate     string                   `json:"start_date" validate:"required"`
	EndDate       string                   `json:"end_date" validate:"required"`
}

// UpdateOfferRequest estructura para actualizar ofertas
type UpdateOfferRequest struct {
	DiscountType  models.OfferDiscountType `json:"discount_type" validate:"required,oneof=percentage fixed_amount fixed_price"`
	DiscountValue float64                  `json:"discount_value" validate:"required,gt=0"`
	StartDate     string                   `json:"start_date" validate:"required"`
	EndDate       string                   `json:"end_date" validate:"required"`
	IsActive      bool                     `json:"is_active"`
}

// CreateOffer crea una nueva oferta
// POST /admin/offers
func (h *OfferHandler) CreateOffer(c *fiber.Ctx) error {
	var req CreateOfferRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos de entrada inválidos",
		})
	}

	// Obtener claims del usuario desde el contexto (middleware de auth)
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID.String()

	// Parsear fechas
	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha de inicio inválido. Use RFC3339",
		})
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha de fin inválido. Use RFC3339",
		})
	}

	// Convertir string a UUID
	productUUID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "product_id inválido",
		})
	}

	// Crear oferta
	offer := &models.ProductOffer{
		ProductID:     productUUID,
		DiscountType:  req.DiscountType,
		DiscountValue: req.DiscountValue,
		StartDate:     startDate,
		EndDate:       endDate,
	}

	if err := h.offerService.CreateOffer(userID, offer); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Oferta creada exitosamente",
		"offer":   offer,
	})
}

// GetOffer obtiene una oferta por ID
// GET /admin/offers/:id
func (h *OfferHandler) GetOffer(c *fiber.Ctx) error {
	offerID := c.Params("id")

	offer, err := h.offerService.GetOffer(offerID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Oferta no encontrada",
		})
	}

	return c.JSON(offer)
}

// UpdateOffer actualiza una oferta existente
// PUT /admin/offers/:id
func (h *OfferHandler) UpdateOffer(c *fiber.Ctx) error {
	offerID := c.Params("id")
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID.String()

	var req UpdateOfferRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos de entrada inválidos",
		})
	}

	// Parsear fechas
	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha de inicio inválido",
		})
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha de fin inválido",
		})
	}

	// Convertir string a UUID
	offerUUID, err := uuid.Parse(offerID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "offer_id inválido",
		})
	}

	// Actualizar oferta
	offer := &models.ProductOffer{
		OfferID:       offerUUID,
		DiscountType:  req.DiscountType,
		DiscountValue: req.DiscountValue,
		StartDate:     startDate,
		EndDate:       endDate,
		IsActive:      req.IsActive,
	}

	if err := h.offerService.UpdateOffer(userID, offerID, offer); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Oferta actualizada exitosamente",
		"offer":   offer,
	})
}

// DeleteOffer elimina una oferta
// DELETE /admin/offers/:id
func (h *OfferHandler) DeleteOffer(c *fiber.Ctx) error {
	offerID := c.Params("id")
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID.String()

	if err := h.offerService.DeleteOffer(userID, offerID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Oferta eliminada exitosamente",
	})
}

// GetProductOffers obtiene productos con ofertas activas (público)
// GET /products/offers
func (h *OfferHandler) GetProductOffers(c *fiber.Ctx) error {
	// Obtener parámetro limit opcional
	limitStr := c.Query("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		limit = 10
	}

	offers, err := h.offerService.GetActiveOffers(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener ofertas",
		})
	}

	// Transformar a formato de respuesta para el frontend
	response := make([]fiber.Map, len(offers))
	for i, offer := range offers {
		if offer.Product != nil {
			finalPrice := offer.CalculateFinalPrice(offer.Product.Price)
			savings := offer.CalculateSavings(offer.Product.Price)
			discountPercentage := offer.GetDiscountPercentageDisplay(offer.Product.Price)

			// Manejar campos que pueden estar vacíos según el modelo Flutter
			var packageSize interface{}
			if offer.Product.PackageSize != "" {
				packageSize = offer.Product.PackageSize
			} else {
				packageSize = nil  // packageSize es nullable en Flutter
			}
			
			var imageURL interface{}
			if offer.Product.ImageURL != "" {
				imageURL = offer.Product.ImageURL
			} else {
				imageURL = nil  // imageUrl es nullable en Flutter
			}

			// unit_of_measure NO puede ser null según el modelo Flutter
			unitOfMeasure := offer.Product.UnitOfMeasure
			if unitOfMeasure == "" {
				unitOfMeasure = "unidad"  // Valor por defecto
			}

			response[i] = fiber.Map{
				"product_id":       offer.Product.ProductID,
				"name":             offer.Product.Name,
				"description":      offer.Product.Description,
				"price":            offer.Product.Price,
				"image_url":        imageURL,
				"unit_of_measure":  unitOfMeasure,  // Siempre string, nunca null
				"package_size":     packageSize,
				"stock_quantity":   offer.Product.StockQuantity,
				"category_id":      offer.Product.CategoryID,
				"is_active":        offer.Product.IsActive,
				"view_count":       offer.Product.ViewCount,
				"purchase_count":   offer.Product.PurchaseCount,
				"rating_average":   offer.Product.RatingAverage,
				"rating_count":     offer.Product.RatingCount,
				"popularity_score": offer.Product.PopularityScore,
				"created_at":       offer.Product.CreatedAt,
				"updated_at":       offer.Product.UpdatedAt,
				"current_offer": fiber.Map{
					"offer_id":       offer.OfferID,
					"discount_type":  offer.DiscountType,
					"discount_value": offer.DiscountValue,
					"start_date":     offer.StartDate,
					"end_date":       offer.EndDate,
				},
				"final_price":         finalPrice,
				"savings":             savings,
				"discount_percentage": discountPercentage,
				"is_on_offer":         true,
			}
		}
	}

	return c.JSON(fiber.Map{
		"products": response,
		"total":    len(response),
	})
}

// SetProductOfferRequest estructura específica para crear ofertas por producto
type SetProductOfferRequest struct {
	DiscountType  models.OfferDiscountType `json:"discount_type" validate:"required,oneof=percentage fixed_amount fixed_price"`
	DiscountValue float64                  `json:"discount_value" validate:"required,gt=0"`
	StartDate     string                   `json:"start_date" validate:"required"`
	EndDate       string                   `json:"end_date" validate:"required"`
}

// SetProductOffer crea/actualiza oferta para un producto específico (método conveniente)
// POST /admin/products/:id/offer
func (h *OfferHandler) SetProductOffer(c *fiber.Ctx) error {
	productID := c.Params("id")
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID.String()

	var req SetProductOfferRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos de entrada inválidos",
		})
	}

	// Parsear fechas
	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha de inicio inválido",
		})
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha de fin inválido",
		})
	}

	// Usar el método conveniente del servicio
	if err := h.offerService.SetProductOffer(
		userID, productID, req.DiscountType, req.DiscountValue, startDate, endDate,
	); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Oferta establecida exitosamente",
	})
}

// RemoveProductOffer elimina la oferta de un producto
// DELETE /admin/products/:id/offer
func (h *OfferHandler) RemoveProductOffer(c *fiber.Ctx) error {
	productID := c.Params("id")
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID.String()

	if err := h.offerService.RemoveProductOffer(userID, productID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Oferta removida exitosamente",
	})
}

// GetProductOffer obtiene la oferta activa de un producto específico
// GET /products/:id/offer
func (h *OfferHandler) GetProductOffer(c *fiber.Ctx) error {
	productID := c.Params("id")

	offer, err := h.offerService.GetProductOffer(productID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No hay ofertas activas para este producto",
		})
	}

	return c.JSON(offer)
}
