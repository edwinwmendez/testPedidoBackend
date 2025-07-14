package services

import (
	"errors"
	"log"

	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/ws"
)

var (
	ErrProductNotFoundService = errors.New("producto no encontrado")
	ErrProductNameExists      = errors.New("ya existe un producto con ese nombre")
)

// ProductService maneja la lógica de negocio relacionada con productos
type ProductService struct {
	repo  repositories.ProductRepository
	wsHub ws.HubInterface
}

// NewProductService crea un nuevo servicio de productos
func NewProductService(repo repositories.ProductRepository, wsHub ws.HubInterface) *ProductService {
	return &ProductService{
		repo:  repo,
		wsHub: wsHub,
	}
}

// Create crea un nuevo producto
func (s *ProductService) Create(product *models.Product) error {
	err := s.repo.Create(product)
	if err != nil {
		return err
	}

	// Enviar notificación WebSocket
	s.notifyProductUpdate(product, "created")
	
	return nil
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
	err := s.repo.Update(product)
	if err != nil {
		return err
	}

	// Enviar notificación WebSocket
	s.notifyProductUpdate(product, "updated")
	
	return nil
}

// Delete elimina un producto por su ID
func (s *ProductService) Delete(id string) error {
	// Primero obtener el producto para la notificación
	product, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	err = s.repo.Delete(id)
	if err != nil {
		return err
	}

	// Enviar notificación WebSocket
	s.notifyProductUpdate(product, "deleted")
	
	return nil
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

// notifyProductUpdate envía notificación WebSocket cuando se modifica un producto
func (s *ProductService) notifyProductUpdate(product *models.Product, action string) {
	// Guard clause: if websocket hub is not available, skip websocket notification
	if s.wsHub == nil {
		log.Printf("[WebSocket] Hub not configured, skipping websocket notification for product %s", product.ProductID.String())
		return
	}

	// Crear payload para WebSocket
	payload := map[string]interface{}{
		"action":  action, // "created", "updated", "deleted"
		"product": product,
	}
	
	// Crear mensaje WebSocket
	msg := ws.Message{
		Type:    ws.ProductUpdate,
		Payload: ws.MustMarshalPayload(payload),
	}

	// Enviar a todos los usuarios conectados (admins ven cambios inmediatamente)
	log.Printf("[WebSocket] Enviando notificación de producto %s: %s", action, product.Name)
	s.wsHub.SendToRole("ADMIN", msg)
	
	// Los clientes también necesitan ver productos nuevos/actualizados
	if action == "created" || action == "updated" {
		s.wsHub.SendToRole("CLIENT", msg)
	}
	
	// Los repartidores también pueden necesitar ver productos actualizados
	s.wsHub.SendToRole("REPARTIDOR", msg)
	
	log.Printf("[WebSocket] Notificación de producto enviada al hub WebSocket")
}
