package services

import (
	"errors"
	"fmt"
	"time"

	"backend/internal/models"
	"backend/internal/repositories"

	"github.com/google/uuid"
)

// OfferService maneja la lógica de negocio para ofertas
type OfferService interface {
	CreateOffer(userID string, offer *models.ProductOffer) error
	GetOffer(offerID string) (*models.ProductOffer, error)
	UpdateOffer(userID, offerID string, offer *models.ProductOffer) error
	DeleteOffer(userID, offerID string) error
	GetActiveOffers(limit int) ([]*models.ProductOffer, error)
	GetProductOffer(productID string) (*models.ProductOffer, error)
	SetProductOffer(userID, productID string, discountType models.OfferDiscountType, discountValue float64, startDate, endDate time.Time) error
	RemoveProductOffer(userID, productID string) error
	IsUserAuthorized(userID string) error
}

type offerService struct {
	offerRepo   repositories.OfferRepository
	userRepo    repositories.UserRepository
	productRepo repositories.ProductRepository
}

// NewOfferService crea una nueva instancia del servicio
func NewOfferService(
	offerRepo repositories.OfferRepository,
	userRepo repositories.UserRepository,
	productRepo repositories.ProductRepository,
) OfferService {
	return &offerService{
		offerRepo:   offerRepo,
		userRepo:    userRepo,
		productRepo: productRepo,
	}
}

// CreateOffer crea una nueva oferta (solo admins)
func (s *offerService) CreateOffer(userID string, offer *models.ProductOffer) error {
	fmt.Printf("🔍 DEBUG CreateOffer: offer.StartDate=%v, offer.EndDate=%v\n", offer.StartDate, offer.EndDate)

	// Verificar permisos de admin
	if err := s.IsUserAuthorized(userID); err != nil {
		return err
	}

	// Verificar que el producto existe y está activo
	product, err := s.productRepo.FindByID(offer.ProductID.String())
	if err != nil {
		return errors.New("producto no encontrado")
	}
	if !product.IsActive {
		return errors.New("no se pueden crear ofertas para productos inactivos")
	}

	// Desactivar cualquier oferta existente para este producto
	if err := s.offerRepo.DeactivateByProductID(offer.ProductID.String()); err != nil {
		return errors.New("error al desactivar ofertas existentes")
	}

	// Establecer campos requeridos solo si no están ya establecidos
	if offer.CreatedBy == uuid.Nil {
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return errors.New("user_id inválido")
		}
		offer.CreatedBy = userUUID
	}

	if !offer.IsActive {
		offer.IsActive = true
	}

	fmt.Printf("🔍 DEBUG CreateOffer before Validate: offer.StartDate=%v, offer.EndDate=%v\n", offer.StartDate, offer.EndDate)

	// Validar y crear la oferta
	if err := offer.Validate(); err != nil {
		return err
	}

	return s.offerRepo.Create(offer)
}

// GetOffer obtiene una oferta por ID
func (s *offerService) GetOffer(offerID string) (*models.ProductOffer, error) {
	return s.offerRepo.GetByID(offerID)
}

// UpdateOffer actualiza una oferta existente (solo admins)
func (s *offerService) UpdateOffer(userID, offerID string, offer *models.ProductOffer) error {
	// Verificar permisos
	if err := s.IsUserAuthorized(userID); err != nil {
		return err
	}

	// Verificar que la oferta existe
	existingOffer, err := s.offerRepo.GetByID(offerID)
	if err != nil {
		return errors.New("oferta no encontrada")
	}

	// Mantener campos inmutables
	offer.OfferID = existingOffer.OfferID
	offer.CreatedBy = existingOffer.CreatedBy
	offer.CreatedAt = existingOffer.CreatedAt

	// Validar y actualizar
	if err := offer.Validate(); err != nil {
		return err
	}

	return s.offerRepo.Update(offer)
}

// DeleteOffer elimina una oferta (solo admins)
func (s *offerService) DeleteOffer(userID, offerID string) error {
	// Verificar permisos
	if err := s.IsUserAuthorized(userID); err != nil {
		return err
	}

	// Verificar que la oferta existe
	_, err := s.offerRepo.GetByID(offerID)
	if err != nil {
		return errors.New("oferta no encontrada")
	}

	return s.offerRepo.Delete(offerID)
}

// GetActiveOffers obtiene todas las ofertas activas (público)
func (s *offerService) GetActiveOffers(limit int) ([]*models.ProductOffer, error) {
	return s.offerRepo.FindActiveOffers(limit)
}

// GetProductOffer obtiene la oferta activa de un producto específico
func (s *offerService) GetProductOffer(productID string) (*models.ProductOffer, error) {
	return s.offerRepo.GetActiveByProductID(productID)
}

// SetProductOffer crea/actualiza una oferta para un producto (método conveniente)
func (s *offerService) SetProductOffer(
	userID, productID string,
	discountType models.OfferDiscountType,
	discountValue float64,
	startDate, endDate time.Time,
) error {
	fmt.Printf("🔍 DEBUG SetProductOffer: startDate=%v, endDate=%v\n", startDate, endDate)

	// Convertir string a UUID
	productUUID, err := uuid.Parse(productID)
	if err != nil {
		return errors.New("product_id inválido")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("user_id inválido")
	}

	offer := &models.ProductOffer{
		ProductID:     productUUID,
		DiscountType:  discountType,
		DiscountValue: discountValue,
		StartDate:     startDate,
		EndDate:       endDate,
		IsActive:      true,
		CreatedBy:     userUUID,
	}

	fmt.Printf("🔍 DEBUG SetProductOffer: offer.StartDate=%v, offer.EndDate=%v\n", offer.StartDate, offer.EndDate)

	return s.CreateOffer(userID, offer)
}

// RemoveProductOffer elimina la oferta de un producto
func (s *offerService) RemoveProductOffer(userID, productID string) error {
	// Verificar permisos
	if err := s.IsUserAuthorized(userID); err != nil {
		return err
	}

	// Desactivar oferta del producto
	return s.offerRepo.DeactivateByProductID(productID)
}

// IsUserAuthorized verifica si el usuario tiene permisos para gestionar ofertas
func (s *offerService) IsUserAuthorized(userID string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("usuario no encontrado")
	}

	if user.UserRole != models.UserRoleAdmin {
		return errors.New("solo los administradores pueden gestionar ofertas")
	}

	return nil
}

// GetOfferStats obtiene estadísticas de ofertas (para dashboard admin)
type OfferStats struct {
	TotalActiveOffers  int     `json:"total_active_offers"`
	TotalOffersToday   int     `json:"total_offers_today"`
	AverageDiscount    float64 `json:"average_discount"`
	ProductsWithOffers int     `json:"products_with_offers"`
}

// GetStats obtiene estadísticas generales de ofertas
func (s *offerService) GetStats() (*OfferStats, error) {
	// Implementación básica - se puede expandir según necesidades
	activeOffers, err := s.offerRepo.FindActiveOffers(0) // Sin límite
	if err != nil {
		return nil, err
	}

	stats := &OfferStats{
		TotalActiveOffers:  len(activeOffers),
		ProductsWithOffers: len(activeOffers),
	}

	// Calcular descuento promedio
	if len(activeOffers) > 0 {
		var totalDiscount float64
		for _, offer := range activeOffers {
			if offer.DiscountType == models.DiscountTypePercentage {
				totalDiscount += offer.DiscountValue
			}
		}
		stats.AverageDiscount = totalDiscount / float64(len(activeOffers))
	}

	return stats, nil
}
