package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category representa una categoría de productos en el sistema
type Category struct {
	CategoryID  uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"category_id"`
	Name        string    `gorm:"type:varchar(100);not null;unique" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	IconName    string    `gorm:"type:varchar(50);not null" json:"icon_name"`
	ColorHex    string    `gorm:"type:varchar(7);not null" json:"color_hex"`
	IsActive    bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relación con productos
	Products []Product `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

// CategoryWithProductCount representa una categoría con el conteo de productos
type CategoryWithProductCount struct {
	CategoryID   uuid.UUID `json:"category_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	IconName     string    `json:"icon_name"`
	ColorHex     string    `json:"color_hex"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ProductCount int64     `json:"product_count"`
}

// BeforeCreate se ejecuta antes de crear una nueva categoría
func (c *Category) BeforeCreate(tx *gorm.DB) (err error) {
	// Si no se proporciona un ID, generamos uno
	if c.CategoryID == uuid.Nil {
		c.CategoryID = uuid.New()
	}
	return nil
}

// TableName especifica el nombre de la tabla para Category
func (Category) TableName() string {
	return "categories"
}

// CreateCategoryRequest representa la solicitud para crear una categoría
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	IconName    string `json:"icon_name" binding:"required,min=1,max=50"`
	ColorHex    string `json:"color_hex" binding:"required,len=7"`
	IsActive    *bool  `json:"is_active"`
}

// UpdateCategoryRequest representa la solicitud para actualizar una categoría
type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100"`
	Description string `json:"description" binding:"omitempty,max=500"`
	IconName    string `json:"icon_name" binding:"omitempty,min=1,max=50"`
	ColorHex    string `json:"color_hex" binding:"omitempty,len=7"`
	IsActive    *bool  `json:"is_active"`
}

// CategoryResponse representa la respuesta de una categoría
type CategoryResponse struct {
	CategoryID   uuid.UUID `json:"category_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	IconName     string    `json:"icon_name"`
	ColorHex     string    `json:"color_hex"`
	IsActive     bool      `json:"is_active"`
	ProductCount int64     `json:"product_count,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToResponse convierte una Category a CategoryResponse
func (c *Category) ToResponse() CategoryResponse {
	return CategoryResponse{
		CategoryID:  c.CategoryID,
		Name:        c.Name,
		Description: c.Description,
		IconName:    c.IconName,
		ColorHex:    c.ColorHex,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// ToResponseWithCount convierte CategoryWithProductCount a CategoryResponse
func (c *CategoryWithProductCount) ToResponse() CategoryResponse {
	return CategoryResponse{
		CategoryID:   c.CategoryID,
		Name:         c.Name,
		Description:  c.Description,
		IconName:     c.IconName,
		ColorHex:     c.ColorHex,
		IsActive:     c.IsActive,
		ProductCount: c.ProductCount,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}
