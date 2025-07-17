package services

import (
	"fmt"
	"log"

	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/ws"

	"github.com/google/uuid"
)

// FavoriteService maneja la lógica de negocio para favoritos
type FavoriteService struct {
	favoriteRepo repositories.FavoriteRepository
	productRepo  repositories.ProductRepository
	userRepo     repositories.UserRepository
	hub          ws.HubInterface
}

// NewFavoriteService crea una nueva instancia del servicio de favoritos
func NewFavoriteService(
	favoriteRepo repositories.FavoriteRepository,
	productRepo repositories.ProductRepository,
	userRepo repositories.UserRepository,
	hub ws.HubInterface,
) *FavoriteService {
	return &FavoriteService{
		favoriteRepo: favoriteRepo,
		productRepo:  productRepo,
		userRepo:     userRepo,
		hub:          hub,
	}
}

// AddFavorite agrega un producto a favoritos del usuario
func (s *FavoriteService) AddFavorite(userID, productID uuid.UUID) (*models.FavoriteActionResponse, error) {
	// Validar que el producto existe y está activo
	product, err := s.productRepo.FindByID(productID.String())
	if err != nil {
		return nil, fmt.Errorf("producto no encontrado: %v", err)
	}

	if !product.IsActive {
		return nil, fmt.Errorf("no se puede agregar un producto inactivo a favoritos")
	}

	// Verificar si ya es favorito
	isFavorite, err := s.favoriteRepo.IsFavorite(userID, productID)
	if err != nil {
		return nil, fmt.Errorf("error al verificar favorito: %v", err)
	}

	if isFavorite {
		return &models.FavoriteActionResponse{
			Success:    false,
			Message:    "El producto ya está en favoritos",
			IsFavorite: true,
			ProductID:  productID,
		}, nil
	}

	// Agregar a favoritos
	err = s.favoriteRepo.AddFavorite(userID, productID)
	if err != nil {
		return nil, fmt.Errorf("error al agregar favorito: %v", err)
	}

	// Notificar via WebSocket
	s.notifyFavoriteChange(userID, productID, true, product.Name)

	return &models.FavoriteActionResponse{
		Success:    true,
		Message:    "Producto agregado a favoritos exitosamente",
		IsFavorite: true,
		ProductID:  productID,
	}, nil
}

// RemoveFavorite quita un producto de favoritos del usuario
func (s *FavoriteService) RemoveFavorite(userID, productID uuid.UUID) (*models.FavoriteActionResponse, error) {
	// Obtener información del producto para notificación
	product, err := s.productRepo.FindByID(productID.String())
	if err != nil {
		return nil, fmt.Errorf("producto no encontrado: %v", err)
	}

	// Verificar si es favorito
	isFavorite, err := s.favoriteRepo.IsFavorite(userID, productID)
	if err != nil {
		return nil, fmt.Errorf("error al verificar favorito: %v", err)
	}

	if !isFavorite {
		return &models.FavoriteActionResponse{
			Success:    false,
			Message:    "El producto no está en favoritos",
			IsFavorite: false,
			ProductID:  productID,
		}, nil
	}

	// Quitar de favoritos
	err = s.favoriteRepo.RemoveFavorite(userID, productID)
	if err != nil {
		return nil, fmt.Errorf("error al quitar favorito: %v", err)
	}

	// Notificar via WebSocket
	s.notifyFavoriteChange(userID, productID, false, product.Name)

	return &models.FavoriteActionResponse{
		Success:    true,
		Message:    "Producto quitado de favoritos exitosamente",
		IsFavorite: false,
		ProductID:  productID,
	}, nil
}

// ToggleFavorite cambia el estado de favorito de un producto
func (s *FavoriteService) ToggleFavorite(userID, productID uuid.UUID) (*models.FavoriteActionResponse, error) {
	// Validar que el producto existe
	product, err := s.productRepo.FindByID(productID.String())
	if err != nil {
		return nil, fmt.Errorf("producto no encontrado: %v", err)
	}

	if !product.IsActive {
		return nil, fmt.Errorf("no se puede marcar como favorito un producto inactivo")
	}

	// Cambiar estado
	isFavorite, err := s.favoriteRepo.ToggleFavorite(userID, productID)
	if err != nil {
		return nil, fmt.Errorf("error al cambiar estado de favorito: %v", err)
	}

	// Notificar via WebSocket
	s.notifyFavoriteChange(userID, productID, isFavorite, product.Name)

	message := "Producto quitado de favoritos exitosamente"
	if isFavorite {
		message = "Producto agregado a favoritos exitosamente"
	}

	return &models.FavoriteActionResponse{
		Success:    true,
		Message:    message,
		IsFavorite: isFavorite,
		ProductID:  productID,
	}, nil
}

// GetUserFavorites obtiene todos los productos favoritos de un usuario con paginación
func (s *FavoriteService) GetUserFavorites(userID uuid.UUID, page, limit int) (*models.FavoritesListResponse, error) {
	// Validar parámetros de paginación
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20 // Límite predeterminado
	}

	// Obtener favoritos
	favorites, totalCount, err := s.favoriteRepo.GetUserFavorites(userID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("error al obtener favoritos: %v", err)
	}

	// Construir respuesta
	response := s.favoriteRepo.BuildFavoritesResponse(favorites, totalCount, page, limit)

	return response, nil
}

// BulkCheckFavorites verifica el estado de favorito para múltiples productos
func (s *FavoriteService) BulkCheckFavorites(userID uuid.UUID, productIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	return s.favoriteRepo.BulkCheckFavorites(userID, productIDs)
}

// GetFavoriteStatus obtiene el estado de favorito de un producto
func (s *FavoriteService) GetFavoriteStatus(userID, productID uuid.UUID) (*models.FavoriteStatusResponse, error) {
	isFavorite, err := s.favoriteRepo.IsFavorite(userID, productID)
	if err != nil {
		return nil, fmt.Errorf("error al verificar favorito: %v", err)
	}

	response := &models.FavoriteStatusResponse{
		IsFavorite: isFavorite,
		ProductID:  productID,
	}

	// Si es favorito, obtener información adicional
	if isFavorite {
		favoriteInfo, err := s.favoriteRepo.GetFavoriteInfo(userID, productID)
		if err == nil {
			response.AddedAt = &favoriteInfo.CreatedAt
		}
	}

	return response, nil
}

// GetFavoriteStats obtiene estadísticas de favoritos de un usuario
func (s *FavoriteService) GetFavoriteStats(userID uuid.UUID) (int, error) {
	return s.favoriteRepo.GetFavoriteStats(userID)
}

// GetMostFavorited obtiene los productos más agregados a favoritos
func (s *FavoriteService) GetMostFavorited(limit int) ([]models.Product, error) {
	if limit < 1 || limit > 50 {
		limit = 10 // Límite predeterminado
	}

	return s.favoriteRepo.GetMostFavorited(limit)
}

// notifyFavoriteChange envía notificación WebSocket sobre cambio de favorito
func (s *FavoriteService) notifyFavoriteChange(userID, productID uuid.UUID, isFavorite bool, productName string) {
	if s.hub == nil {
		return
	}

	action := "removed"
	if isFavorite {
		action = "added"
	}

	// Crear payload para el WebSocket siguiendo el patrón existente
	payload := map[string]interface{}{
		"user_id":      userID.String(),
		"product_id":   productID.String(),
		"product_name": productName,
		"action":       action,
		"is_favorite":  isFavorite,
		"timestamp":    getCurrentTimeString(),
	}

	// Crear mensaje WebSocket siguiendo el patrón existente
	message := ws.Message{
		Type:    "favorite_change",
		Payload: ws.MustMarshalPayload(payload),
	}

	// Enviar solo al usuario específico
	s.hub.SendToUser(userID.String(), message)
	
	log.Printf("Notificación de favorito enviada: usuario=%s, producto=%s, acción=%s", 
		userID.String(), productID.String(), action)
}