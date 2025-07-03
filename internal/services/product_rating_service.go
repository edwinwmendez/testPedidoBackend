package services

import (
	"errors"

	"backend/internal/models"
	"backend/internal/repositories"

	"gorm.io/gorm"
)

var (
	ErrRatingNotFound           = errors.New("calificación no encontrada")
	ErrRatingAlreadyExists      = errors.New("ya existe una calificación para este producto")
	ErrUnauthorizedRatingAccess = errors.New("no tienes permisos para modificar esta calificación")
)

// ProductRatingService maneja la lógica de negocio relacionada con calificaciones
type ProductRatingService struct {
	ratingRepo  repositories.ProductRatingRepository
	productRepo repositories.ProductRepository
}

// NewProductRatingService crea un nuevo servicio de calificaciones
func NewProductRatingService(
	ratingRepo repositories.ProductRatingRepository,
	productRepo repositories.ProductRepository,
) *ProductRatingService {
	return &ProductRatingService{
		ratingRepo:  ratingRepo,
		productRepo: productRepo,
	}
}

// Create crea una nueva calificación
func (s *ProductRatingService) Create(req *models.CreateRatingRequest, userID string) (*models.RatingResponse, error) {
	// Verificar que el producto existe
	_, err := s.productRepo.FindByID(req.ProductID.String())
	if err != nil {
		return nil, ErrProductNotFoundService
	}

	// Verificar que el usuario no haya calificado ya este producto
	existingRating, err := s.ratingRepo.FindByProductAndUser(req.ProductID.String(), userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existingRating != nil {
		return nil, ErrRatingAlreadyExists
	}

	// Crear la calificación
	rating := &models.ProductRating{
		ProductID:  req.ProductID,
		UserID:     mustParseUUID(userID),
		Rating:     req.Rating,
		ReviewText: req.ReviewText,
	}

	if err := s.ratingRepo.Create(rating); err != nil {
		return nil, err
	}

	// Crear respuesta
	response := &models.RatingResponse{
		RatingID:   rating.RatingID,
		ProductID:  rating.ProductID,
		UserID:     rating.UserID,
		Rating:     rating.Rating,
		ReviewText: rating.ReviewText,
		CreatedAt:  rating.CreatedAt,
		UpdatedAt:  rating.UpdatedAt,
	}

	return response, nil
}

// GetByProduct obtiene todas las calificaciones de un producto
func (s *ProductRatingService) GetByProduct(productID string) ([]*models.RatingResponse, error) {
	// Verificar que el producto existe
	_, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, ErrProductNotFoundService
	}

	ratings, err := s.ratingRepo.FindByProduct(productID)
	if err != nil {
		return nil, err
	}

	// Convertir a response format
	responses := make([]*models.RatingResponse, len(ratings))
	for i, rating := range ratings {
		responses[i] = &models.RatingResponse{
			RatingID:   rating.RatingID,
			ProductID:  rating.ProductID,
			UserID:     rating.UserID,
			Rating:     rating.Rating,
			ReviewText: rating.ReviewText,
			CreatedAt:  rating.CreatedAt,
			UpdatedAt:  rating.UpdatedAt,
		}
		// Agregar nombre de usuario si está disponible
		if rating.User != nil {
			responses[i].UserName = rating.User.FullName
		}
	}

	return responses, nil
}

// GetUserRatingForProduct obtiene la calificación de un usuario para un producto específico
func (s *ProductRatingService) GetUserRatingForProduct(productID, userID string) (*models.RatingResponse, error) {
	rating, err := s.ratingRepo.FindByProductAndUser(productID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No error, simplemente no existe
		}
		return nil, err
	}

	response := &models.RatingResponse{
		RatingID:   rating.RatingID,
		ProductID:  rating.ProductID,
		UserID:     rating.UserID,
		Rating:     rating.Rating,
		ReviewText: rating.ReviewText,
		CreatedAt:  rating.CreatedAt,
		UpdatedAt:  rating.UpdatedAt,
	}

	return response, nil
}