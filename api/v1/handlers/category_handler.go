package handlers

import (
	"backend/internal/models"
	"backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CategoryHandler maneja las peticiones relacionadas con categorías
type CategoryHandler struct {
	categoryService *services.CategoryService
}

// NewCategoryHandler crea un nuevo handler de categorías
func NewCategoryHandler(categoryService *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

// RegisterRoutes registra las rutas del handler de categorías
func (h *CategoryHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler, adminOnly fiber.Handler) {
	// Rutas públicas
	categories := router.Group("/categories")
	categories.Get("/", h.GetAllCategories)
	categories.Get("/active", h.GetActiveCategories)
	categories.Get("/with-count", h.GetCategoriesWithProductCount)
	categories.Get("/:id", h.GetCategoryByID)

	// Rutas solo para administradores
	adminCategories := router.Group("/categories", authMiddleware, adminOnly)
	adminCategories.Post("/", h.CreateCategory)
	adminCategories.Put("/:id", h.UpdateCategory)
	adminCategories.Delete("/:id", h.DeleteCategory)
}

// GetAllCategories obtiene todas las categorías
// @Summary Obtener todas las categorías
// @Description Obtiene una lista de todas las categorías del sistema
// @Tags categorías
// @Accept json
// @Produce json
// @Success 200 {array} models.CategoryResponse
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/categories [get]
func (h *CategoryHandler) GetAllCategories(c *fiber.Ctx) error {
	categories, err := h.categoryService.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No se pudieron obtener las categorías",
		})
	}

	response := make([]models.CategoryResponse, len(categories))
	for i, category := range categories {
		response[i] = category.ToResponse()
	}

	return c.JSON(response)
}

// GetActiveCategories obtiene todas las categorías activas
// @Summary Obtener categorías activas
// @Description Obtiene una lista de todas las categorías activas del sistema
// @Tags categorías
// @Accept json
// @Produce json
// @Success 200 {array} models.CategoryResponse
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/categories/active [get]
func (h *CategoryHandler) GetActiveCategories(c *fiber.Ctx) error {
	categories, err := h.categoryService.GetActive()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No se pudieron obtener las categorías activas",
		})
	}

	response := make([]models.CategoryResponse, len(categories))
	for i, category := range categories {
		response[i] = category.ToResponse()
	}

	return c.JSON(response)
}

// GetCategoriesWithProductCount obtiene categorías con conteo de productos
// @Summary Obtener categorías con conteo de productos
// @Description Obtiene una lista de categorías activas con el número de productos de cada una
// @Tags categorías
// @Accept json
// @Produce json
// @Success 200 {array} models.CategoryResponse
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/categories/with-count [get]
func (h *CategoryHandler) GetCategoriesWithProductCount(c *fiber.Ctx) error {
	categories, err := h.categoryService.GetWithProductCount()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No se pudieron obtener las categorías con conteo",
		})
	}

	response := make([]models.CategoryResponse, len(categories))
	for i, category := range categories {
		response[i] = category.ToResponse()
	}

	return c.JSON(response)
}

// GetCategoryByID obtiene una categoría por su ID
// @Summary Obtener categoría por ID
// @Description Obtiene una categoría específica por su ID
// @Tags categorías
// @Accept json
// @Produce json
// @Param id path string true "ID de la categoría"
// @Success 200 {object} models.CategoryResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/categories/{id} [get]
func (h *CategoryHandler) GetCategoryByID(c *fiber.Ctx) error {
	id := c.Params("id")

	// Validar UUID
	if _, err := uuid.Parse(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El ID de la categoría debe ser un UUID válido",
		})
	}

	category, err := h.categoryService.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No se encontró una categoría con el ID especificado",
		})
	}

	return c.JSON(category.ToResponse())
}

// CreateCategory crea una nueva categoría
// @Summary Crear nueva categoría
// @Description Crea una nueva categoría en el sistema (solo administradores)
// @Tags categorías
// @Accept json
// @Produce json
// @Param category body models.CreateCategoryRequest true "Datos de la categoría"
// @Success 201 {object} models.CategoryResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/categories [post]
func (h *CategoryHandler) CreateCategory(c *fiber.Ctx) error {
	var req models.CreateCategoryRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No se pudieron procesar los datos de la categoría",
		})
	}

	// Crear el objeto Category
	category := &models.Category{
		Name:        req.Name,
		Description: req.Description,
		IconName:    req.IconName,
		ColorHex:    req.ColorHex,
		IsActive:    true,
	}

	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}

	if err := h.categoryService.Create(category); err != nil {
		if err == services.ErrCategoryNameExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Ya existe una categoría con ese nombre",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No se pudo crear la categoría",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(category.ToResponse())
}

// UpdateCategory actualiza una categoría existente
// @Summary Actualizar categoría
// @Description Actualiza una categoría existente (solo administradores)
// @Tags categorías
// @Accept json
// @Produce json
// @Param id path string true "ID de la categoría"
// @Param category body models.UpdateCategoryRequest true "Datos actualizados de la categoría"
// @Success 200 {object} models.CategoryResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *fiber.Ctx) error {
	id := c.Params("id")

	// Validar UUID
	if _, err := uuid.Parse(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El ID de la categoría debe ser un UUID válido",
		})
	}

	var req models.UpdateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No se pudieron procesar los datos de la categoría",
		})
	}

	// Obtener la categoría existente
	category, err := h.categoryService.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No se encontró una categoría con el ID especificado",
		})
	}

	// Actualizar campos si se proporcionan
	if req.Name != "" {
		category.Name = req.Name
	}
	if req.Description != "" {
		category.Description = req.Description
	}
	if req.IconName != "" {
		category.IconName = req.IconName
	}
	if req.ColorHex != "" {
		category.ColorHex = req.ColorHex
	}
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}

	if err := h.categoryService.Update(category); err != nil {
		if err == services.ErrCategoryNameExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Ya existe una categoría con ese nombre",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No se pudo actualizar la categoría",
		})
	}

	return c.JSON(category.ToResponse())
}

// DeleteCategory elimina una categoría
// @Summary Eliminar categoría
// @Description Elimina una categoría del sistema (solo administradores)
// @Tags categorías
// @Accept json
// @Produce json
// @Param id path string true "ID de la categoría"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *fiber.Ctx) error {
	id := c.Params("id")

	// Validar UUID
	if _, err := uuid.Parse(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El ID de la categoría debe ser un UUID válido",
		})
	}

	// Verificar que la categoría existe
	_, err := h.categoryService.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No se encontró una categoría con el ID especificado",
		})
	}

	if err := h.categoryService.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No se pudo eliminar la categoría",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Categoría eliminada exitosamente",
	})
}