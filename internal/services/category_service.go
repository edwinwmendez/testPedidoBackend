package services

import (
	"errors"

	"backend/internal/models"
	"backend/internal/repositories"
)

var (
	ErrCategoryNotFoundService = errors.New("categoría no encontrada")
	ErrCategoryNameExists      = errors.New("ya existe una categoría con ese nombre")
)

// CategoryService maneja la lógica de negocio relacionada con categorías
type CategoryService struct {
	repo repositories.CategoryRepository
}

// NewCategoryService crea un nuevo servicio de categorías
func NewCategoryService(repo repositories.CategoryRepository) *CategoryService {
	return &CategoryService{
		repo: repo,
	}
}

// Create crea una nueva categoría
func (s *CategoryService) Create(category *models.Category) error {
	return s.repo.Create(category)
}

// GetByID obtiene una categoría por su ID
func (s *CategoryService) GetByID(id string) (*models.Category, error) {
	return s.repo.FindByID(id)
}

// GetAll obtiene todas las categorías
func (s *CategoryService) GetAll() ([]*models.Category, error) {
	return s.repo.FindAll()
}

// GetActive obtiene todas las categorías activas
func (s *CategoryService) GetActive() ([]*models.Category, error) {
	return s.repo.FindActive()
}

// GetWithProductCount obtiene todas las categorías activas con conteo de productos
func (s *CategoryService) GetWithProductCount() ([]*models.CategoryWithProductCount, error) {
	return s.repo.FindWithProductCount()
}

// Update actualiza una categoría existente
func (s *CategoryService) Update(category *models.Category) error {
	return s.repo.Update(category)
}

// Delete elimina una categoría por su ID
func (s *CategoryService) Delete(id string) error {
	return s.repo.Delete(id)
}