package repositories

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

type ProductRatingRepository interface {
	Create(rating *models.ProductRating) error
	FindByID(id string) (*models.ProductRating, error)
	FindByProductAndUser(productID, userID string) (*models.ProductRating, error)
	FindByProduct(productID string) ([]*models.ProductRating, error)
	FindByUser(userID string) ([]*models.ProductRating, error)
	Update(rating *models.ProductRating) error
	Delete(id string) error
}

type productRatingRepository struct {
	db *gorm.DB
}

func NewProductRatingRepository(db *gorm.DB) ProductRatingRepository {
	return &productRatingRepository{
		db: db,
	}
}

func (r *productRatingRepository) Create(rating *models.ProductRating) error {
	return r.db.Create(rating).Error
}

func (r *productRatingRepository) FindByID(id string) (*models.ProductRating, error) {
	var rating models.ProductRating
	err := r.db.Preload("Product").Preload("User").
		Where("rating_id = ?", id).
		First(&rating).Error
	if err != nil {
		return nil, err
	}
	return &rating, nil
}

func (r *productRatingRepository) FindByProductAndUser(productID, userID string) (*models.ProductRating, error) {
	var rating models.ProductRating
	err := r.db.Where("product_id = ? AND user_id = ?", productID, userID).
		First(&rating).Error
	if err != nil {
		return nil, err
	}
	return &rating, nil
}

func (r *productRatingRepository) FindByProduct(productID string) ([]*models.ProductRating, error) {
	var ratings []*models.ProductRating
	err := r.db.Preload("User").
		Where("product_id = ?", productID).
		Order("created_at DESC").
		Find(&ratings).Error
	return ratings, err
}

func (r *productRatingRepository) FindByUser(userID string) ([]*models.ProductRating, error) {
	var ratings []*models.ProductRating
	err := r.db.Preload("Product").Preload("Product.Category").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&ratings).Error
	return ratings, err
}

func (r *productRatingRepository) Update(rating *models.ProductRating) error {
	return r.db.Model(rating).
		Where("rating_id = ?", rating.RatingID).
		Updates(map[string]interface{}{
			"rating":      rating.Rating,
			"review_text": rating.ReviewText,
			"updated_at":  "NOW()",
		}).Error
}

func (r *productRatingRepository) Delete(id string) error {
	return r.db.Delete(&models.ProductRating{}, "rating_id = ?", id).Error
}