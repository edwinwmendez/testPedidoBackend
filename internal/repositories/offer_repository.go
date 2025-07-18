package repositories

import (
	"fmt"
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
)

// OfferRepository maneja las operaciones de base de datos para ofertas
type OfferRepository interface {
	Create(offer *models.ProductOffer) error
	GetByID(offerID string) (*models.ProductOffer, error)
	GetByProductID(productID string) (*models.ProductOffer, error)
	GetActiveByProductID(productID string) (*models.ProductOffer, error)
	Update(offer *models.ProductOffer) error
	Delete(offerID string) error
	DeactivateByProductID(productID string) error
	FindActiveOffers(limit int) ([]*models.ProductOffer, error)
	FindOffersByDateRange(startDate, endDate time.Time) ([]*models.ProductOffer, error)
	FindByCreator(createdBy string) ([]*models.ProductOffer, error)
}

type offerRepository struct {
	db *gorm.DB
}

// NewOfferRepository crea una nueva instancia del repositorio
func NewOfferRepository(db *gorm.DB) OfferRepository {
	return &offerRepository{db: db}
}

// Create crea una nueva oferta
func (r *offerRepository) Create(offer *models.ProductOffer) error {
	fmt.Printf("🔍 DEBUG Repository Create: StartDate=%v, EndDate=%v, OfferID=%v\n", offer.StartDate, offer.EndDate, offer.OfferID)
	result := r.db.Create(offer)
	fmt.Printf("🔍 DEBUG Repository Create result: RowsAffected=%d, Error=%v\n", result.RowsAffected, result.Error)
	return result.Error
}

// GetByID obtiene una oferta por su ID
func (r *offerRepository) GetByID(offerID string) (*models.ProductOffer, error) {
	var offer models.ProductOffer
	err := r.db.Where("offer_id = ?", offerID).First(&offer).Error
	if err != nil {
		return nil, err
	}
	return &offer, nil
}

// GetByProductID obtiene la oferta (activa o inactiva) de un producto
func (r *offerRepository) GetByProductID(productID string) (*models.ProductOffer, error) {
	var offer models.ProductOffer
	err := r.db.Where("product_id = ?", productID).
		Order("created_at DESC").
		First(&offer).Error
	if err != nil {
		return nil, err
	}
	return &offer, nil
}

// GetActiveByProductID obtiene la oferta activa de un producto específico
func (r *offerRepository) GetActiveByProductID(productID string) (*models.ProductOffer, error) {
	var offer models.ProductOffer
	now := time.Now()

	err := r.db.Where("product_id = ? AND is_active = ? AND start_date <= ? AND end_date >= ?",
		productID, true, now, now).
		First(&offer).Error

	if err != nil {
		return nil, err
	}
	return &offer, nil
}

// Update actualiza una oferta existente
func (r *offerRepository) Update(offer *models.ProductOffer) error {
	return r.db.Save(offer).Error
}

// Delete elimina una oferta por ID
func (r *offerRepository) Delete(offerID string) error {
	return r.db.Where("offer_id = ?", offerID).Delete(&models.ProductOffer{}).Error
}

// DeactivateByProductID desactiva cualquier oferta activa de un producto
func (r *offerRepository) DeactivateByProductID(productID string) error {
	fmt.Printf("🔍 DEBUG: DeactivateByProductID called with productID: %s\n", productID)

	result := r.db.Model(&models.ProductOffer{}).
		Where("product_id = ? AND is_active = ?", productID, true).
		Update("is_active", false)

	fmt.Printf("🔍 DEBUG: DeactivateByProductID result: RowsAffected=%d, Error=%v\n", result.RowsAffected, result.Error)

	return result.Error
}

// FindActiveOffers encuentra todas las ofertas activas (para /products/offers)
func (r *offerRepository) FindActiveOffers(limit int) ([]*models.ProductOffer, error) {
	var offers []*models.ProductOffer
	now := time.Now()

	query := r.db.Preload("Product").
		Where("is_active = ? AND start_date <= ? AND end_date >= ?", true, now, now).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&offers).Error
	return offers, err
}

// FindOffersByDateRange encuentra ofertas en un rango de fechas
func (r *offerRepository) FindOffersByDateRange(startDate, endDate time.Time) ([]*models.ProductOffer, error) {
	var offers []*models.ProductOffer
	err := r.db.Preload("Product").
		Where("(start_date BETWEEN ? AND ?) OR (end_date BETWEEN ? AND ?) OR (start_date <= ? AND end_date >= ?)",
			startDate, endDate, startDate, endDate, startDate, endDate).
		Order("start_date DESC").
		Find(&offers).Error
	return offers, err
}

// FindByCreator encuentra ofertas creadas por un usuario específico
func (r *offerRepository) FindByCreator(createdBy string) ([]*models.ProductOffer, error) {
	var offers []*models.ProductOffer
	err := r.db.Preload("Product").
		Where("created_by = ?", createdBy).
		Order("created_at DESC").
		Find(&offers).Error
	return offers, err
}
