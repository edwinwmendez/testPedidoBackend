package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserFavorite representa la relación entre usuario y producto favorito
type UserFavorite struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	ProductID uuid.UUID `gorm:"type:uuid;primaryKey" json:"product_id"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`

	// Relaciones
	User    *User    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Product *Product `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"product,omitempty"`
}

// TableName especifica el nombre de la tabla para UserFavorite
func (UserFavorite) TableName() string {
	return "user_favorites"
}

// FavoriteResponse representa la respuesta de favoritos con información del producto
type FavoriteResponse struct {
	ProductID     uuid.UUID `json:"product_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	ImageURL      string    `json:"image_url"`
	UnitOfMeasure string    `json:"unit_of_measure"`
	PackageSize   string    `json:"package_size"`
	StockQuantity int       `json:"stock_quantity"`
	CategoryID    uuid.UUID `json:"category_id"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	AddedAt       time.Time `json:"added_at"` // Fecha cuando se agregó a favoritos
}

// FavoritesListResponse representa la respuesta paginada de favoritos
type FavoritesListResponse struct {
	Favorites   []FavoriteResponse `json:"favorites"`
	TotalCount  int                `json:"total_count"`
	CurrentPage int                `json:"current_page"`
	TotalPages  int                `json:"total_pages"`
	PageSize    int                `json:"page_size"`
	HasNext     bool               `json:"has_next"`
	HasPrevious bool               `json:"has_previous"`
}

// FavoriteActionRequest representa la petición para agregar/quitar favorito
type FavoriteActionRequest struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
}

// FavoriteActionResponse representa la respuesta de acción sobre favorito
type FavoriteActionResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	IsFavorite bool  `json:"is_favorite"`
	ProductID uuid.UUID `json:"product_id"`
}

// FavoriteStatusRequest representa la petición para verificar estado de favorito
type FavoriteStatusRequest struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
}

// FavoriteStatusResponse representa la respuesta del estado de favorito
type FavoriteStatusResponse struct {
	IsFavorite bool      `json:"is_favorite"`
	ProductID  uuid.UUID `json:"product_id"`
	AddedAt    *time.Time `json:"added_at,omitempty"`
}

// BeforeCreate se ejecuta antes de crear un nuevo favorito
func (f *UserFavorite) BeforeCreate(tx *gorm.DB) error {
	// Validar que el producto esté activo
	var product Product
	if err := tx.Where("product_id = ? AND is_active = true", f.ProductID).First(&product).Error; err != nil {
		return err
	}
	
	// Validar que el usuario esté activo
	var user User
	if err := tx.Where("user_id = ? AND is_active = true", f.UserID).First(&user).Error; err != nil {
		return err
	}
	
	return nil
}