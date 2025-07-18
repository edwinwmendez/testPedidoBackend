package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Product representa un producto en el sistema de tienda PedidoMendez
type Product struct {
	ProductID     uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"product_id"`
	Name          string     `gorm:"type:varchar(255);not null;unique" json:"name"`
	Description   string     `gorm:"type:text" json:"description"`
	Price         float64    `gorm:"type:decimal(10,2);not null;check:price > 0" json:"price"`
	ImageURL            string     `gorm:"type:varchar(255)" json:"image_url"`
	UnitOfMeasure       string     `gorm:"type:varchar(50);not null;default:'unidad'" json:"unit_of_measure"`
	PackageSize         string     `gorm:"type:varchar(50)" json:"package_size"`
	StockQuantity       int        `gorm:"type:integer;not null;default:100;check:stock_quantity >= 0" json:"stock_quantity"`
	CategoryID          *uuid.UUID `gorm:"type:uuid" json:"category_id"`
	IsActive            bool       `gorm:"not null;default:true" json:"is_active"`

	// Analytics fields
	ViewCount       int     `gorm:"type:integer;not null;default:0" json:"view_count"`
	PurchaseCount   int     `gorm:"type:integer;not null;default:0" json:"purchase_count"`
	RatingAverage   float64 `gorm:"type:decimal(3,2);not null;default:0.00" json:"rating_average"`
	RatingCount     int     `gorm:"type:integer;not null;default:0" json:"rating_count"`
	PopularityScore float64 `gorm:"type:decimal(10,2);not null;default:0.00" json:"popularity_score"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relaciones
	Category    *Category        `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Ratings     []ProductRating  `gorm:"foreignKey:ProductID" json:"ratings,omitempty"`
	CurrentOffer *ProductOffer   `gorm:"foreignKey:ProductID" json:"current_offer,omitempty"`
}

// BeforeCreate se ejecuta antes de crear un nuevo producto
func (p *Product) BeforeCreate(tx *gorm.DB) (err error) {
	// Si no se proporciona un ID, generamos uno
	if p.ProductID == uuid.Nil {
		p.ProductID = uuid.New()
	}
	return nil
}

// TableName especifica el nombre de la tabla para Product
func (Product) TableName() string {
	return "products"
}

// GetFinalPrice obtiene el precio final considerando ofertas activas
func (p *Product) GetFinalPrice() float64 {
	if p.CurrentOffer != nil && p.CurrentOffer.IsCurrentlyActive() {
		return p.CurrentOffer.CalculateFinalPrice(p.Price)
	}
	return p.Price
}

// GetSavings obtiene el ahorro si hay una oferta activa
func (p *Product) GetSavings() float64 {
	if p.CurrentOffer != nil && p.CurrentOffer.IsCurrentlyActive() {
		return p.CurrentOffer.CalculateSavings(p.Price)
	}
	return 0
}

// GetDiscountPercentage obtiene el porcentaje de descuento para mostrar en UI
func (p *Product) GetDiscountPercentage() float64 {
	if p.CurrentOffer != nil && p.CurrentOffer.IsCurrentlyActive() {
		return p.CurrentOffer.GetDiscountPercentageDisplay(p.Price)
	}
	return 0
}

// IsOnOffer verifica si el producto tiene una oferta activa
func (p *Product) IsOnOffer() bool {
	return p.CurrentOffer != nil && p.CurrentOffer.IsCurrentlyActive()
}
