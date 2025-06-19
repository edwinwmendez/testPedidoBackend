package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Product representa un producto en el sistema (balón de gas)
type Product struct {
	ProductID     uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"product_id"`
	Name          string    `gorm:"type:varchar(255);not null;unique" json:"name"`
	Description   string    `gorm:"type:text" json:"description"`
	Price         float64   `gorm:"type:decimal(10,2);not null;check:price > 0" json:"price"`
	ImageURL      string    `gorm:"type:varchar(255)" json:"image_url"`
	StockQuantity int       `gorm:"type:integer;not null;default:100;check:stock_quantity >= 0" json:"stock_quantity"`
	IsActive      bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt     time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null;default:now()" json:"updated_at"`
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
