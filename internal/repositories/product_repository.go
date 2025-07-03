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
	IncrementViewCount(id string) error
	IncrementPurchaseCount(id string) error
	Update(product *models.Product) error
	Delete(id string) error
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
