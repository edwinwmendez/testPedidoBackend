package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductRating representa una calificación de producto por parte de un usuario
type ProductRating struct {
	RatingID   uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"rating_id"`
	ProductID  uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Rating     int       `gorm:"type:integer;not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	ReviewText string    `gorm:"type:text" json:"review_text"`
	CreatedAt  time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relaciones
	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// BeforeCreate se ejecuta antes de crear un nuevo rating
func (pr *ProductRating) BeforeCreate(tx *gorm.DB) (err error) {
	if pr.RatingID == uuid.Nil {
		pr.RatingID = uuid.New()
	}
	return nil
}

// TableName especifica el nombre de la tabla para ProductRating
func (ProductRating) TableName() string {
	return "product_ratings"
}

// CreateRatingRequest representa la solicitud para crear un rating
type CreateRatingRequest struct {
	ProductID  uuid.UUID `json:"product_id" validate:"required"`
	Rating     int       `json:"rating" validate:"required,min=1,max=5"`
	ReviewText string    `json:"review_text" validate:"omitempty,max=1000"`
}

// UpdateRatingRequest representa la solicitud para actualizar un rating
type UpdateRatingRequest struct {
	Rating     int    `json:"rating" validate:"required,min=1,max=5"`
	ReviewText string `json:"review_text" validate:"omitempty,max=1000"`
}

// RatingResponse representa la respuesta con información del rating
type RatingResponse struct {
	RatingID   uuid.UUID `json:"rating_id"`
	ProductID  uuid.UUID `json:"product_id"`
	UserID     uuid.UUID `json:"user_id"`
	Rating     int       `json:"rating"`
	ReviewText string    `json:"review_text"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	UserName   string    `json:"user_name,omitempty"`
}