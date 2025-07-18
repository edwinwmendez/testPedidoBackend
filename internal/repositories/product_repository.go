package repositories

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

type ProductRepository interface {
	Create(product *models.Product) error
	FindByID(id string) (*models.Product, error)
	FindAll() ([]*models.Product, error)
	FindActive() ([]*models.Product, error)
	FindPopular(limit int) ([]*models.Product, error)
	FindRecent(limit int) ([]*models.Product, error)
	FindWithOffers() ([]*models.Product, error)
	FindActiveWithOffers(limit int) ([]*models.Product, error)
	IncrementViewCount(id string) error
	IncrementPurchaseCount(id string) error
	Update(product *models.Product) error
	Delete(id string) error
	LoadCurrentOffer(product *models.Product) error
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{
		db: db,
	}
}

func (r *productRepository) Create(product *models.Product) error {
	// Special handling for inactive products due to GORM zero-value omission issue
	if !product.IsActive {
		// Use raw SQL for inactive products to ensure false value is preserved
		return r.db.Exec(`
			INSERT INTO products (product_id, name, description, price, category_id, image_url, stock_quantity, is_active, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			product.ProductID, product.Name, product.Description, product.Price, product.CategoryID, product.ImageURL, product.StockQuantity, false).Error
	}

	// For active products, use standard GORM create
	return r.db.Create(product).Error
}

func (r *productRepository) FindByID(id string) (*models.Product, error) {
	var product models.Product

	if err := r.db.Preload("Category").Where("product_id = ?", id).First(&product).Error; err != nil {
		return nil, err
	}

	return &product, nil
}

func (r *productRepository) FindAll() ([]*models.Product, error) {
	var products []*models.Product

	if err := r.db.Preload("Category").Find(&products).Error; err != nil {
		return nil, err
	}

	return products, nil
}

func (r *productRepository) FindActive() ([]*models.Product, error) {
	var products []*models.Product

	if err := r.db.Preload("Category").Where("is_active = ?", true).Find(&products).Error; err != nil {
		return nil, err
	}

	return products, nil
}

func (r *productRepository) Update(product *models.Product) error {
	// Use Updates to handle pointer fields correctly
	return r.db.Model(product).Where("product_id = ?", product.ProductID).Updates(map[string]interface{}{
		"name":           product.Name,
		"description":    product.Description,
		"price":          product.Price,
		"category_id":    product.CategoryID,
		"image_url":      product.ImageURL,
		"stock_quantity": product.StockQuantity,
		"is_active":      product.IsActive,
		"updated_at":     "NOW()",
	}).Error
}

func (r *productRepository) Delete(id string) error {
	return r.db.Delete(&models.Product{}, "product_id = ?", id).Error
}

// FindPopular obtiene productos populares basados en popularity_score
func (r *productRepository) FindPopular(limit int) ([]*models.Product, error) {
	var products []*models.Product

	err := r.db.Preload("Category").
		Where("is_active = ? AND stock_quantity > 0", true).
		Order("popularity_score DESC, purchase_count DESC, rating_average DESC").
		Limit(limit).
		Find(&products).Error

	return products, err
}

// FindRecent obtiene los productos más recientes
func (r *productRepository) FindRecent(limit int) ([]*models.Product, error) {
	var products []*models.Product

	err := r.db.Preload("Category").
		Where("is_active = ? AND stock_quantity > 0", true).
		Order("created_at DESC").
		Limit(limit).
		Find(&products).Error

	return products, err
}

// IncrementViewCount incrementa el contador de vistas de un producto
func (r *productRepository) IncrementViewCount(id string) error {
	return r.db.Model(&models.Product{}).
		Where("product_id = ?", id).
		Update("view_count", gorm.Expr("view_count + 1")).Error
}

// IncrementPurchaseCount incrementa el contador de compras de un producto
func (r *productRepository) IncrementPurchaseCount(id string) error {
	return r.db.Model(&models.Product{}).
		Where("product_id = ?", id).
		Update("purchase_count", gorm.Expr("purchase_count + 1")).Error
}

// FindWithOffers encuentra todos los productos con sus ofertas activas
func (r *productRepository) FindWithOffers() ([]*models.Product, error) {
	var products []*models.Product
	
	subquery := r.db.Table("product_offers").
		Select("DISTINCT product_id").
		Where("is_active = ? AND start_date <= NOW() AND end_date >= NOW()", true)
	
	err := r.db.Preload("Category").
		Preload("CurrentOffer", "is_active = ? AND start_date <= NOW() AND end_date >= NOW()", true).
		Where("is_active = ? AND product_id IN (?)", true, subquery).
		Find(&products).Error
	
	return products, err
}

// FindActiveWithOffers encuentra productos activos con ofertas, con límite
func (r *productRepository) FindActiveWithOffers(limit int) ([]*models.Product, error) {
	var products []*models.Product
	
	query := r.db.Preload("Category").
		Preload("CurrentOffer", "is_active = ? AND start_date <= NOW() AND end_date >= NOW()", true).
		Joins("JOIN product_offers ON products.product_id = product_offers.product_id").
		Where("products.is_active = ? AND product_offers.is_active = ? AND product_offers.start_date <= NOW() AND product_offers.end_date >= NOW()", 
			true, true).
		Group("products.product_id")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&products).Error
	return products, err
}

// LoadCurrentOffer carga la oferta activa actual para un producto
func (r *productRepository) LoadCurrentOffer(product *models.Product) error {
	return r.db.Preload("CurrentOffer", "is_active = ? AND start_date <= NOW() AND end_date >= NOW()", true).
		Where("product_id = ?", product.ProductID).
		First(product).Error
}

