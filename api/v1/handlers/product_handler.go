package handlers

import (
	"backend/internal/models"
	"backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ProductHandler maneja las peticiones HTTP relacionadas con productos
type ProductHandler struct {
	productService *services.ProductService
}

// NewProductHandler crea un nuevo handler de productos
func NewProductHandler(productService *services.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

// GetAllProducts obtiene todos los productos
// @Summary Obtener todos los productos
// @Description Obtiene la lista de todos los productos disponibles
// @Tags productos
// @Accept json
// @Produce json
// @Param active query boolean false "Solo productos activos"
// @Success 200 {array} models.Product
// @Failure 500 {object} map[string]interface{}
// @Router /products [get]
func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	// Obtener parámetros de consulta opcionales
	onlyActive := c.Query("active") == "true"

	var products []*models.Product
	var err error

	if onlyActive {
		products, err = h.productService.GetActive()
	} else {
		products, err = h.productService.GetAll()
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los productos",
		})
	}

	return c.JSON(products)
}

// GetProductByID obtiene un producto por su ID
// @Summary Obtener un producto por ID
// @Description Obtiene los detalles de un producto específico por su ID
// @Tags productos
// @Accept json
// @Produce json
// @Param id path string true "ID del producto"
// @Success 200 {object} models.Product
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /products/{id} [get]
func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	// Obtener el ID del producto de los parámetros
	productID := c.Params("id")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de producto requerido",
		})
	}

	// Obtener el producto
	product, err := h.productService.GetByID(productID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Producto no encontrado",
		})
	}

	return c.JSON(product)
}

// CreateProductRequest estructura para crear un producto
type CreateProductRequest struct {
	Name          string  `json:"name" validate:"required"`
	Description   string  `json:"description"`
	Price         float64 `json:"price" validate:"required,min=0"`
	ImageURL      string  `json:"image_url"`
	StockQuantity int     `json:"stock_quantity" validate:"min=0"`
	IsActive      bool    `json:"is_active"`
}

// CreateProduct crea un nuevo producto (solo para administradores)
// @Summary Crear un nuevo producto
// @Description Crea un nuevo producto (solo para administradores)
// @Tags productos
// @Accept json
// @Produce json
// @Param product body CreateProductRequest true "Datos del producto"
// @Success 201 {object} models.Product
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	// Parsear el cuerpo de la petición
	var req CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos de producto inválidos",
		})
	}

	// Validar los datos
	if req.Name == "" || req.Price <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nombre y precio son campos requeridos y el precio debe ser mayor a 0",
		})
	}

	// Crear el producto en el modelo
	product := &models.Product{
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		ImageURL:      req.ImageURL,
		StockQuantity: req.StockQuantity,
		IsActive:      req.IsActive,
	}

	// Crear el producto usando el servicio
	if err := h.productService.Create(product); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al crear el producto",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(product)
}

// UpdateProductRequest estructura para actualizar un producto
type UpdateProductRequest struct {
	Name          string  `json:"name,omitempty"`
	Description   string  `json:"description,omitempty"`
	Price         float64 `json:"price,omitempty" validate:"omitempty,min=0"`
	CategoryID    string  `json:"category_id,omitempty"`
	ImageURL      string  `json:"image_url,omitempty"`
	StockQuantity *int    `json:"stock_quantity,omitempty" validate:"omitempty,min=0"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

// UpdateProduct actualiza un producto existente (solo para administradores)
// @Summary Actualizar un producto
// @Description Actualiza un producto existente (solo para administradores)
// @Tags productos
// @Accept json
// @Produce json
// @Param id path string true "ID del producto"
// @Param product body UpdateProductRequest true "Datos del producto"
// @Success 200 {object} models.Product
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *fiber.Ctx) error {
	// Obtener el ID del producto de los parámetros
	productID := c.Params("id")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de producto requerido",
		})
	}

	// Parsear el cuerpo de la petición
	var req UpdateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Obtener el producto actual
	product, err := h.productService.GetByID(productID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Producto no encontrado",
		})
	}

	// Actualizar los campos del producto
	if req.Name != "" {
		product.Name = req.Name
	}

	if req.Description != "" {
		product.Description = req.Description
	}

	if req.Price > 0 {
		product.Price = req.Price
	}

	if req.CategoryID != "" {
		categoryUUID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "ID de categoría inválido",
			})
		}
		product.CategoryID = &categoryUUID
	}

	if req.ImageURL != "" {
		product.ImageURL = req.ImageURL
	}

	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	if req.StockQuantity != nil {
		product.StockQuantity = *req.StockQuantity
	}

	// Guardar los cambios
	if err := h.productService.Update(product); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al actualizar el producto",
		})
	}

	return c.JSON(product)
}

// DeleteProduct elimina un producto (solo para administradores)
// @Summary Eliminar un producto
// @Description Elimina un producto existente (solo para administradores)
// @Tags productos
// @Accept json
// @Produce json
// @Param id path string true "ID del producto"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *fiber.Ctx) error {
	// Obtener el ID del producto de los parámetros
	productID := c.Params("id")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de producto requerido",
		})
	}

	// Eliminar el producto
	if err := h.productService.Delete(productID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al eliminar el producto",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RegisterRoutes registra las rutas del handler en el router
func (h *ProductHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler, adminOnly fiber.Handler) {
	products := router.Group("/products")

	// Rutas públicas
	products.Get("/", h.GetAllProducts)
	products.Get("/:id", h.GetProductByID)

	// Rutas solo para administradores
	adminProducts := router.Group("/products", authMiddleware, adminOnly)
	adminProducts.Post("/", h.CreateProduct)
	adminProducts.Put("/:id", h.UpdateProduct)
	adminProducts.Delete("/:id", h.DeleteProduct)
}
