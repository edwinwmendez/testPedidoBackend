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

	// Actualizar el promedio de calificación del producto
	if err := s.updateProductRatingAverage(req.ProductID.String()); err != nil {
		// Log el error pero no fallar la creación
		// En producción usaríamos un logger apropiado
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

// Update actualiza una calificación existente
func (s *ProductRatingService) Update(req *models.CreateRatingRequest, userID string) (*models.RatingResponse, error) {
	// Verificar que el producto existe
	_, err := s.productRepo.FindByID(req.ProductID.String())
	if err != nil {
		return nil, ErrProductNotFoundService
	}

	// Buscar la calificación existente del usuario
	existingRating, err := s.ratingRepo.FindByProductAndUser(req.ProductID.String(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRatingNotFound
		}
		return nil, err
	}

	// Actualizar la calificación
	existingRating.Rating = req.Rating
	existingRating.ReviewText = req.ReviewText

	// Guardar en la base de datos
	if err := s.ratingRepo.Update(existingRating); err != nil {
		return nil, err
	}

	// Actualizar promedio del producto
	if err := s.updateProductRatingAverage(req.ProductID.String()); err != nil {
		return nil, err
	}

	// Crear respuesta
	response := &models.RatingResponse{
		RatingID:   existingRating.RatingID,
		ProductID:  existingRating.ProductID,
		UserID:     existingRating.UserID,
		Rating:     existingRating.Rating,
		ReviewText: existingRating.ReviewText,
		CreatedAt:  existingRating.CreatedAt,
		UpdatedAt:  existingRating.UpdatedAt,
	}

	return response, nil
}

// updateProductRatingAverage actualiza el promedio de calificación de un producto
func (s *ProductRatingService) updateProductRatingAverage(productID string) error {
	ratings, err := s.ratingRepo.FindByProduct(productID)
	if err != nil {
		return err
	}

	if len(ratings) == 0 {
		return nil
	}

	// Calcular promedio
	sum := 0
	for _, rating := range ratings {
		sum += rating.Rating
	}
	average := float64(sum) / float64(len(ratings))

	// Actualizar producto
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return err
	}

	product.RatingAverage = average
	product.RatingCount = len(ratings)

	return s.productRepo.Update(product)
}
