package services

import (
	"errors"

	"backend/internal/models"
	"backend/internal/repositories"
)

var (
	ErrProductNotFoundService = errors.New("producto no encontrado")
	ErrProductNameExists      = errors.New("ya existe un producto con ese nombre")
)

// ProductService maneja la lógica de negocio relacionada con productos
type ProductService struct {
	repo repositories.ProductRepository
}

// NewProductService crea un nuevo servicio de productos
func NewProductService(repo repositories.ProductRepository) *ProductService {
	return &ProductService{
		repo: repo,
	}
}

// Create crea un nuevo producto
func (s *ProductService) Create(product *models.Product) error {
	return s.repo.Create(product)
}

// GetByID obtiene un producto por su ID
func (s *ProductService) GetByID(id string) (*models.Product, error) {
	return s.repo.FindByID(id)
}

// GetAll obtiene todos los productos
func (s *ProductService) GetAll() ([]*models.Product, error) {
	return s.repo.FindAll()
}

// GetActive obtiene todos los productos activos
func (s *ProductService) GetActive() ([]*models.Product, error) {
	return s.repo.FindActive()
}

// Update actualiza un producto existente
func (s *ProductService) Update(product *models.Product) error {
	return s.repo.Update(product)
}

// Delete elimina un producto por su ID
func (s *ProductService) Delete(id string) error {
	return s.repo.Delete(id)
}

// GetPopular obtiene productos populares
func (s *ProductService) GetPopular(limit int) ([]*models.Product, error) {
	if limit <= 0 {
		limit = 5 // Default limit
	}
	if limit > 20 {
		limit = 20 // Maximum limit to prevent abuse
	}

	return s.repo.FindPopular(limit)
}

// GetRecent obtiene productos recientes
func (s *ProductService) GetRecent(limit int) ([]*models.Product, error) {
	if limit <= 0 {
		limit = 5 // Default limit
	}
	if limit > 20 {
		limit = 20 // Maximum limit to prevent abuse
	}

	return s.repo.FindRecent(limit)
}

// IncrementViewCount incrementa el contador de vistas
func (s *ProductService) IncrementViewCount(id string) error {
	// Verificar que el producto existe
	_, err := s.repo.FindByID(id)
	if err != nil {
		return ErrProductNotFoundService
	}

	return s.repo.IncrementViewCount(id)
}

// IncrementPurchaseCount incrementa el contador de compras
func (s *ProductService) IncrementPurchaseCount(id string) error {
	// Verificar que el producto existe
	_, err := s.repo.FindByID(id)
	if err != nil {
		return ErrProductNotFoundService
	}

	return s.repo.IncrementPurchaseCount(id)
}
