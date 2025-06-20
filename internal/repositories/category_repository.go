package repositories

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

type CategoryRepository interface {
	Create(category *models.Category) error
	FindByID(id string) (*models.Category, error)
	FindAll() ([]*models.Category, error)
	FindActive() ([]*models.Category, error)
	FindWithProductCount() ([]*models.CategoryWithProductCount, error)
	Update(category *models.Category) error
	Delete(id string) error
}

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{
		db: db,
	}
}

func (r *categoryRepository) Create(category *models.Category) error {
	// Special handling for inactive categories due to GORM zero-value omission issue
	if !category.IsActive {
		// Use raw SQL for inactive categories to ensure false value is preserved
		return r.db.Exec(`
			INSERT INTO categories (category_id, name, description, icon_name, color_hex, is_active, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			category.CategoryID, category.Name, category.Description, category.IconName, category.ColorHex, false).Error
	}
	
	// For active categories, use standard GORM create
	return r.db.Create(category).Error
}

func (r *categoryRepository) FindByID(id string) (*models.Category, error) {
	var category models.Category

	if err := r.db.Preload("Products").Where("category_id = ?", id).First(&category).Error; err != nil {
		return nil, err
	}

	return &category, nil
}

func (r *categoryRepository) FindAll() ([]*models.Category, error) {
	var categories []*models.Category

	if err := r.db.Preload("Products").Find(&categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *categoryRepository) FindActive() ([]*models.Category, error) {
	var categories []*models.Category

	if err := r.db.Preload("Products").Where("is_active = ?", true).Find(&categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *categoryRepository) FindWithProductCount() ([]*models.CategoryWithProductCount, error) {
	var categories []*models.CategoryWithProductCount

	query := `
		SELECT 
			c.category_id,
			c.name,
			c.description,
			c.icon_name,
			c.color_hex,
			c.is_active,
			c.created_at,
			c.updated_at,
			COUNT(p.product_id) AS product_count
		FROM categories c
		LEFT JOIN products p ON c.category_id = p.category_id AND p.is_active = true
		WHERE c.is_active = true
		GROUP BY c.category_id, c.name, c.description, c.icon_name, c.color_hex, c.is_active, c.created_at, c.updated_at
		ORDER BY c.name
	`

	if err := r.db.Raw(query).Scan(&categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *categoryRepository) Update(category *models.Category) error {
	return r.db.Save(category).Error
}

func (r *categoryRepository) Delete(id string) error {
	return r.db.Delete(&models.Category{}, "category_id = ?", id).Error
}