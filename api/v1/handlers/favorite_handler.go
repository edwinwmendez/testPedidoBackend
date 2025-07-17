package handlers

import (
	"net/http"

	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// FavoriteHandler maneja las peticiones HTTP relacionadas con favoritos
type FavoriteHandler struct {
	favoriteService *services.FavoriteService
}

// NewFavoriteHandler crea una nueva instancia del handler de favoritos
func NewFavoriteHandler(favoriteService *services.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{
		favoriteService: favoriteService,
	}
}

// AddFavorite agrega un producto a favoritos del usuario
// @Summary Agregar producto a favoritos
// @Description Agrega un producto a la lista de favoritos del usuario autenticado
// @Tags Favoritos
// @Accept json
// @Produce json
// @Param request body models.FavoriteActionRequest true "Información del producto a agregar"
// @Success 200 {object} models.FavoriteActionResponse "Producto agregado exitosamente"
// @Failure 400 {object} map[string]string "Petición inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/favorites [post]
// @Security Bearer
func (h *FavoriteHandler) AddFavorite(c *fiber.Ctx) error {
	// Obtener ID del usuario desde el contexto
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID

	// Parsear petición
	var request models.FavoriteActionRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos de petición inválidos",
		})
	}

	// Validar que el product_id no esté vacío
	if request.ProductID == uuid.Nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "product_id es requerido",
		})
	}

	// Agregar a favoritos
	response, err := h.favoriteService.AddFavorite(userID, request.ProductID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// RemoveFavorite quita un producto de favoritos del usuario
// @Summary Quitar producto de favoritos
// @Description Quita un producto de la lista de favoritos del usuario autenticado
// @Tags Favoritos
// @Accept json
// @Produce json
// @Param request body models.FavoriteActionRequest true "Información del producto a quitar"
// @Success 200 {object} models.FavoriteActionResponse "Producto quitado exitosamente"
// @Failure 400 {object} map[string]string "Petición inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/favorites [delete]
// @Security Bearer
func (h *FavoriteHandler) RemoveFavorite(c *fiber.Ctx) error {
	// Obtener ID del usuario desde el contexto
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID

	// Parsear petición
	var request models.FavoriteActionRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos de petición inválidos",
		})
	}

	// Validar que el product_id no esté vacío
	if request.ProductID == uuid.Nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "product_id es requerido",
		})
	}

	// Quitar de favoritos
	response, err := h.favoriteService.RemoveFavorite(userID, request.ProductID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// ToggleFavorite cambia el estado de favorito de un producto
// @Summary Cambiar estado de favorito
// @Description Cambia el estado de favorito de un producto (agregar si no está, quitar si está)
// @Tags Favoritos
// @Accept json
// @Produce json
// @Param request body models.FavoriteActionRequest true "Información del producto"
// @Success 200 {object} models.FavoriteActionResponse "Estado cambiado exitosamente"
// @Failure 400 {object} map[string]string "Petición inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/favorites/toggle [post]
// @Security Bearer
func (h *FavoriteHandler) ToggleFavorite(c *fiber.Ctx) error {
	// Obtener ID del usuario desde el contexto
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID

	// Parsear petición
	var request models.FavoriteActionRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos de petición inválidos",
		})
	}

	// Validar que el product_id no esté vacío
	if request.ProductID == uuid.Nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "product_id es requerido",
		})
	}

	// Cambiar estado
	response, err := h.favoriteService.ToggleFavorite(userID, request.ProductID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// GetFavoriteStatus obtiene el estado de favorito de un producto
// @Summary Obtener estado de favorito
// @Description Obtiene el estado de favorito de un producto específico para el usuario autenticado
// @Tags Favoritos
// @Produce json
// @Param product_id path string true "ID del producto"
// @Success 200 {object} models.FavoriteStatusResponse "Estado de favorito obtenido exitosamente"
// @Failure 400 {object} map[string]string "Petición inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/favorites/status/{product_id} [get]
// @Security Bearer
func (h *FavoriteHandler) GetFavoriteStatus(c *fiber.Ctx) error {
	// Obtener ID del usuario desde el contexto
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID

	// Obtener product_id desde parámetros
	productIDStr := c.Params("product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de producto inválido",
		})
	}

	// Obtener estado
	response, err := h.favoriteService.GetFavoriteStatus(userID, productID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// GetUserFavorites obtiene todos los productos favoritos del usuario con paginación
// @Summary Obtener favoritos del usuario
// @Description Obtiene todos los productos favoritos del usuario autenticado con paginación
// @Tags Favoritos
// @Produce json
// @Param page query int false "Número de página (default: 1)"
// @Param limit query int false "Límite por página (default: 20, max: 100)"
// @Success 200 {object} models.FavoritesListResponse "Favoritos obtenidos exitosamente"
// @Failure 400 {object} map[string]string "Petición inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/favorites [get]
// @Security Bearer
func (h *FavoriteHandler) GetUserFavorites(c *fiber.Ctx) error {
	// Obtener ID del usuario desde el contexto
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID

	// Obtener parámetros de paginación
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	// Validar parámetros
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Obtener favoritos
	response, err := h.favoriteService.GetUserFavorites(userID, page, limit)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// GetFavoriteStats obtiene estadísticas de favoritos del usuario
// @Summary Obtener estadísticas de favoritos
// @Description Obtiene estadísticas de favoritos del usuario autenticado
// @Tags Favoritos
// @Produce json
// @Success 200 {object} map[string]int "Estadísticas de favoritos"
// @Failure 400 {object} map[string]string "Petición inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/favorites/stats [get]
// @Security Bearer
func (h *FavoriteHandler) GetFavoriteStats(c *fiber.Ctx) error {
	// Obtener ID del usuario desde el contexto
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID

	// Obtener estadísticas
	count, err := h.favoriteService.GetFavoriteStats(userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"total_favorites": count,
		"user_id":         userID.String(),
	})
}

// GetMostFavorited obtiene los productos más agregados a favoritos
// @Summary Obtener productos más favoritos
// @Description Obtiene los productos más agregados a favoritos (solo para ADMIN)
// @Tags Favoritos
// @Produce json
// @Param limit query int false "Límite de productos (default: 10, max: 50)"
// @Success 200 {object} []models.Product "Productos más favoritos"
// @Failure 400 {object} map[string]string "Petición inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 403 {object} map[string]string "Acceso denegado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/favorites/most-favorited [get]
// @Security Bearer
func (h *FavoriteHandler) GetMostFavorited(c *fiber.Ctx) error {
	// Verificar que el usuario sea ADMIN
	claims := c.Locals("user").(*auth.Claims)
	if claims.UserRole != models.UserRoleAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Acceso denegado. Solo administradores pueden acceder a esta información",
		})
	}

	// Obtener límite
	limit := c.QueryInt("limit", 10)
	if limit < 1 || limit > 50 {
		limit = 10
	}

	// Obtener productos más favoritos
	products, err := h.favoriteService.GetMostFavorited(limit)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"products": products,
		"count":    len(products),
	})
}

// BulkCheckFavorites verifica el estado de favorito para múltiples productos
// @Summary Verificar múltiples favoritos
// @Description Verifica el estado de favorito para múltiples productos del usuario autenticado
// @Tags Favoritos
// @Accept json
// @Produce json
// @Param request body []string true "Lista de IDs de productos"
// @Success 200 {object} map[string]bool "Estado de favoritos para cada producto"
// @Failure 400 {object} map[string]string "Petición inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/favorites/bulk-check [post]
// @Security Bearer
func (h *FavoriteHandler) BulkCheckFavorites(c *fiber.Ctx) error {
	// Obtener ID del usuario desde el contexto
	claims := c.Locals("user").(*auth.Claims)
	userID := claims.UserID

	// Parsear lista de product_ids
	var productIDStrs []string
	if err := c.BodyParser(&productIDStrs); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Datos de petición inválidos",
		})
	}

	// Validar que la lista no esté vacía
	if len(productIDStrs) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "La lista de productos no puede estar vacía",
		})
	}

	// Limitar cantidad de productos a verificar
	if len(productIDStrs) > 100 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "No se pueden verificar más de 100 productos a la vez",
		})
	}

	// Convertir strings a UUIDs
	var productIDs []uuid.UUID
	for _, idStr := range productIDStrs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "ID de producto inválido: " + idStr,
			})
		}
		productIDs = append(productIDs, id)
	}

	// Verificar favoritos
	result, err := h.favoriteService.BulkCheckFavorites(userID, productIDs)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convertir resultado a map[string]bool para respuesta JSON
	resultMap := make(map[string]bool)
	for productID, isFavorite := range result {
		resultMap[productID.String()] = isFavorite
	}

	return c.Status(http.StatusOK).JSON(resultMap)
}

// RegisterRoutes registra todas las rutas de favoritos
func (h *FavoriteHandler) RegisterRoutes(api fiber.Router, authMiddleware fiber.Handler, adminOnly fiber.Handler) {
	// Grupo de rutas para favoritos
	favorites := api.Group("/favorites")

	// Rutas para todos los usuarios autenticados
	favorites.Get("/", authMiddleware, h.GetUserFavorites)          // GET /api/v1/favorites
	favorites.Post("/", authMiddleware, h.AddFavorite)              // POST /api/v1/favorites
	favorites.Delete("/", authMiddleware, h.RemoveFavorite)         // DELETE /api/v1/favorites
	favorites.Post("/toggle", authMiddleware, h.ToggleFavorite)     // POST /api/v1/favorites/toggle
	favorites.Get("/stats", authMiddleware, h.GetFavoriteStats)     // GET /api/v1/favorites/stats
	favorites.Get("/status/:product_id", authMiddleware, h.GetFavoriteStatus) // GET /api/v1/favorites/status/{product_id}
	favorites.Post("/bulk-check", authMiddleware, h.BulkCheckFavorites) // POST /api/v1/favorites/bulk-check

	// Rutas solo para administradores
	favorites.Get("/most-favorited", authMiddleware, adminOnly, h.GetMostFavorited) // GET /api/v1/favorites/most-favorited
}