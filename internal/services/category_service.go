package services

import (
	"errors"
	"log"

	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/ws"
)

var (
	ErrCategoryNotFoundService = errors.New("categoría no encontrada")
	ErrCategoryNameExists      = errors.New("ya existe una categoría con ese nombre")
)

// CategoryService maneja la lógica de negocio relacionada con categorías
type CategoryService struct {
	repo  repositories.CategoryRepository
	wsHub ws.HubInterface
}

// NewCategoryService crea un nuevo servicio de categorías
func NewCategoryService(repo repositories.CategoryRepository, wsHub ws.HubInterface) *CategoryService {
	return &CategoryService{
		repo:  repo,
		wsHub: wsHub,
	}
}

// Create crea una nueva categoría
func (s *CategoryService) Create(category *models.Category) error {
	if err := s.repo.Create(category); err != nil {
		return err
	}

	// Notificar sobre nueva categoría
	s.notifyCategoryUpdate(category, "created")

	return nil
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
	if err := s.repo.Update(category); err != nil {
		return err
	}

	// Notificar sobre categoría actualizada
	s.notifyCategoryUpdate(category, "updated")

	return nil
}

// Delete elimina una categoría por su ID
func (s *CategoryService) Delete(id string) error {
	// Obtener la categoría antes de eliminarla para la notificación
	category, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}

	// Notificar sobre categoría eliminada
	s.notifyCategoryUpdate(category, "deleted")

	return nil
}

// notifyCategoryUpdate envía notificaciones WebSocket sobre cambios en categorías
func (s *CategoryService) notifyCategoryUpdate(category *models.Category, action string) {
	if s.wsHub == nil {
		log.Printf("[WebSocket] Hub not configured, skipping websocket notification for category %s", category.CategoryID.String())
		return
	}

	payload := map[string]interface{}{
		"action":   action, // "created", "updated", "deleted"
		"category": category,
	}

	msg := ws.Message{
		Type:    ws.CategoryUpdate,
		Payload: ws.MustMarshalPayload(payload),
	}

	// Enviar a todos los roles ya que las categorías afectan a todos
	s.wsHub.SendToRole("ADMIN", msg)
	s.wsHub.SendToRole("CLIENT", msg)
	s.wsHub.SendToRole("REPARTIDOR", msg)

	log.Printf("[WebSocket] Category %s notification sent for action: %s", category.CategoryID.String(), action)
}
