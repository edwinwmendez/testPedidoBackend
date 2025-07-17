package repositories

import (
	"math"

	"backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FavoriteRepository interfaz para las operaciones de favoritos
type FavoriteRepository interface {
	AddFavorite(userID, productID uuid.UUID) error
	RemoveFavorite(userID, productID uuid.UUID) error
	IsFavorite(userID, productID uuid.UUID) (bool, error)
	GetFavoriteInfo(userID, productID uuid.UUID) (*models.UserFavorite, error)
	GetUserFavorites(userID uuid.UUID, page, limit int) ([]models.FavoriteResponse, int, error)
	GetFavoritesByProduct(productID uuid.UUID) ([]uuid.UUID, error)
	GetFavoriteStats(userID uuid.UUID) (int, error)
	GetMostFavorited(limit int) ([]models.Product, error)
	ToggleFavorite(userID, productID uuid.UUID) (bool, error)
	BuildFavoritesResponse(favorites []models.FavoriteResponse, totalCount, page, limit int) *models.FavoritesListResponse
	CleanupInactiveFavorites() error
	BulkCheckFavorites(userID uuid.UUID, productIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

// favoriteRepository implementación concreta del repositorio de favoritos
type favoriteRepository struct {
	db *gorm.DB
}

// NewFavoriteRepository crea una nueva instancia del repositorio de favoritos
func NewFavoriteRepository(db *gorm.DB) FavoriteRepository {
	return &favoriteRepository{db: db}
}

// AddFavorite agrega un producto a favoritos del usuario
func (r *favoriteRepository) AddFavorite(userID, productID uuid.UUID) error {
	favorite := models.UserFavorite{
		UserID:    userID,
		ProductID: productID,
	}

	// Usar GORM para insertar o ignorar si ya existe
	result := r.db.Create(&favorite)
	if result.Error != nil {
		// Si es error de clave duplicada, no es realmente un error
		if result.Error.Error() == "UNIQUE constraint failed" ||
			result.Error.Error() == "duplicate key value violates unique constraint" {
			return nil
		}
		return result.Error
	}

	return nil
}

// RemoveFavorite quita un producto de favoritos del usuario
func (r *favoriteRepository) RemoveFavorite(userID, productID uuid.UUID) error {
	result := r.db.Where("user_id = ? AND product_id = ?", userID, productID).
		Delete(&models.UserFavorite{})

	if result.Error != nil {
		return result.Error
	}

	return nil
}

// IsFavorite verifica si un producto es favorito del usuario
func (r *favoriteRepository) IsFavorite(userID, productID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.UserFavorite{}).
		Where("user_id = ? AND product_id = ?", userID, productID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetFavoriteInfo obtiene información sobre cuándo se agregó un producto a favoritos
func (r *favoriteRepository) GetFavoriteInfo(userID, productID uuid.UUID) (*models.UserFavorite, error) {
	var favorite models.UserFavorite
	err := r.db.Where("user_id = ? AND product_id = ?", userID, productID).
		First(&favorite).Error

	if err != nil {
		return nil, err
	}

	return &favorite, nil
}

// GetUserFavorites obtiene todos los productos favoritos de un usuario con paginación
func (r *favoriteRepository) GetUserFavorites(userID uuid.UUID, page, limit int) ([]models.FavoriteResponse, int, error) {
	var favorites []models.FavoriteResponse
	var totalCount int64

	// Calcular offset para paginación
	offset := (page - 1) * limit

	// Contar total de favoritos
	err := r.db.Model(&models.UserFavorite{}).
		Where("user_id = ?", userID).
		Count(&totalCount).Error
	if err != nil {
		return nil, 0, err
	}

	// Obtener favoritos con información del producto
	err = r.db.Table("user_favorites uf").
		Select(`
			p.product_id,
			p.name,
			p.description,
			p.price,
			p.image_url,
			p.unit_of_measure,
			p.package_size,
			p.stock_quantity,
			p.category_id,
			p.is_active,
			p.created_at,
			p.updated_at,
			uf.created_at as added_at
		`).
		Joins("INNER JOIN products p ON uf.product_id = p.product_id").
		Where("uf.user_id = ? AND p.is_active = true", userID).
		Order("uf.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&favorites).Error

	if err != nil {
		return nil, 0, err
	}

	return favorites, int(totalCount), nil
}

// GetFavoritesByProduct obtiene todos los usuarios que tienen un producto como favorito
func (r *favoriteRepository) GetFavoritesByProduct(productID uuid.UUID) ([]uuid.UUID, error) {
	var userIDs []uuid.UUID

	err := r.db.Model(&models.UserFavorite{}).
		Where("product_id = ?", productID).
		Pluck("user_id", &userIDs).Error

	if err != nil {
		return nil, err
	}

	return userIDs, nil
}

// GetFavoriteStats obtiene estadísticas de favoritos de un usuario
func (r *favoriteRepository) GetFavoriteStats(userID uuid.UUID) (int, error) {
	var count int64
	err := r.db.Model(&models.UserFavorite{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// GetMostFavorited obtiene los productos más agregados a favoritos
func (r *favoriteRepository) GetMostFavorited(limit int) ([]models.Product, error) {
	var products []models.Product

	err := r.db.Table("products p").
		Select("p.*, COUNT(uf.product_id) as favorite_count").
		Joins("LEFT JOIN user_favorites uf ON p.product_id = uf.product_id").
		Where("p.is_active = true").
		Group("p.product_id").
		Order("favorite_count DESC").
		Limit(limit).
		Find(&products).Error

	if err != nil {
		return nil, err
	}

	return products, nil
}

// ToggleFavorite cambia el estado de favorito de un producto
func (r *favoriteRepository) ToggleFavorite(userID, productID uuid.UUID) (bool, error) {
	isFavorite, err := r.IsFavorite(userID, productID)
	if err != nil {
		return false, err
	}

	if isFavorite {
		err = r.RemoveFavorite(userID, productID)
		if err != nil {
			return false, err
		}
		return false, nil
	} else {
		err = r.AddFavorite(userID, productID)
		if err != nil {
			return false, err
		}
		return true, nil
	}
}

// BuildFavoritesResponse construye la respuesta paginada de favoritos
func (r *favoriteRepository) BuildFavoritesResponse(favorites []models.FavoriteResponse, totalCount, page, limit int) *models.FavoritesListResponse {
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
	
	// Asegurar que favorites nunca sea nil para evitar null en JSON
	if favorites == nil {
		favorites = []models.FavoriteResponse{}
	}

	return &models.FavoritesListResponse{
		Favorites:   favorites,
		TotalCount:  totalCount,
		CurrentPage: page,
		TotalPages:  totalPages,
		PageSize:    limit,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
}

// CleanupInactiveFavorites limpia favoritos de productos inactivos
func (r *favoriteRepository) CleanupInactiveFavorites() error {
	return r.db.Where("product_id NOT IN (SELECT product_id FROM products WHERE is_active = true)").
		Delete(&models.UserFavorite{}).Error
}

// BulkCheckFavorites verifica el estado de favorito para múltiples productos
func (r *favoriteRepository) BulkCheckFavorites(userID uuid.UUID, productIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := make(map[uuid.UUID]bool)
	
	// Inicializar todos los productos como no favoritos
	for _, productID := range productIDs {
		result[productID] = false
	}
	
	// Obtener productos que SÍ están en favoritos
	var favorites []models.UserFavorite
	err := r.db.Where("user_id = ? AND product_id IN ?", userID, productIDs).
		Find(&favorites).Error
	
	if err != nil {
		return nil, err
	}
	
	// Marcar como favoritos los que existen
	for _, favorite := range favorites {
		result[favorite.ProductID] = true
	}
	
	return result, nil
}