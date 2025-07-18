package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OfferDiscountType representa los tipos de descuento disponibles
type OfferDiscountType string

const (
	DiscountTypePercentage  OfferDiscountType = "percentage"   // Descuento porcentual (ej: 20%)
	DiscountTypeFixedAmount OfferDiscountType = "fixed_amount" // Monto fijo (ej: $5.00)
	DiscountTypeFixedPrice  OfferDiscountType = "fixed_price"  // Precio fijo (ej: $15.99)
)

// ProductOffer representa una oferta aplicada a un producto específico
type ProductOffer struct {
	OfferID       uuid.UUID         `json:"offer_id" gorm:"primaryKey;column:offer_id;type:uuid;default:gen_random_uuid()"`
	ProductID     uuid.UUID         `json:"product_id" gorm:"column:product_id;type:uuid;not null"`
	DiscountType  OfferDiscountType `json:"discount_type" gorm:"column:discount_type;type:offer_discount_type;not null"`
	DiscountValue float64           `json:"discount_value" gorm:"column:discount_value;type:decimal(10,2);not null"`
	StartDate     time.Time         `json:"start_date" gorm:"column:start_date;type:timestamptz;not null"`
	EndDate       time.Time         `json:"end_date" gorm:"column:end_date;type:timestamptz;not null"`
	IsActive      bool              `json:"is_active" gorm:"column:is_active;default:true"`
	CreatedBy     uuid.UUID         `json:"created_by" gorm:"column:created_by;type:uuid;not null"`
	CreatedAt     time.Time         `json:"created_at" gorm:"column:created_at;type:timestamptz;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time         `json:"updated_at" gorm:"column:updated_at;type:timestamptz;default:CURRENT_TIMESTAMP"`

	// Relaciones
	Product *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ProductID"`
	Creator *User    `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;references:UserID"`
}

// TableName especifica el nombre de la tabla
func (ProductOffer) TableName() string {
	return "product_offers"
}

// IsCurrentlyActive verifica si la oferta está activa en este momento
func (po *ProductOffer) IsCurrentlyActive() bool {
	now := time.Now()
	return po.IsActive &&
		now.After(po.StartDate) &&
		now.Before(po.EndDate)
}

// CalculateFinalPrice calcula el precio final aplicando el descuento
func (po *ProductOffer) CalculateFinalPrice(originalPrice float64) float64 {
	if !po.IsCurrentlyActive() {
		return originalPrice
	}

	switch po.DiscountType {
	case DiscountTypePercentage:
		// Descuento porcentual: precio * (1 - porcentaje/100)
		return originalPrice * (1 - po.DiscountValue/100)
	case DiscountTypeFixedAmount:
		// Descuento fijo: precio - monto
		finalPrice := originalPrice - po.DiscountValue
		if finalPrice < 0 {
			return 0 // No puede ser precio negativo
		}
		return finalPrice
	case DiscountTypeFixedPrice:
		// Precio fijo: valor específico
		return po.DiscountValue
	default:
		return originalPrice
	}
}

// CalculateSavings calcula cuánto ahorra el cliente
func (po *ProductOffer) CalculateSavings(originalPrice float64) float64 {
	finalPrice := po.CalculateFinalPrice(originalPrice)
	return originalPrice - finalPrice
}

// GetDiscountPercentageDisplay obtiene el porcentaje a mostrar en UI
func (po *ProductOffer) GetDiscountPercentageDisplay(originalPrice float64) float64 {
	if originalPrice <= 0 {
		return 0
	}

	switch po.DiscountType {
	case DiscountTypePercentage:
		return po.DiscountValue
	case DiscountTypeFixedAmount:
		return (po.DiscountValue / originalPrice) * 100
	case DiscountTypeFixedPrice:
		savings := originalPrice - po.DiscountValue
		return (savings / originalPrice) * 100
	default:
		return 0
	}
}

// Validate valida los datos de la oferta
func (po *ProductOffer) Validate() error {
	// Debug: imprimir las fechas que se están comparando
	fmt.Printf("🔍 DEBUG Validate: StartDate=%v, EndDate=%v\n", po.StartDate, po.EndDate)
	fmt.Printf("🔍 DEBUG Validate: EndDate.Before(StartDate)=%v, EndDate.Equal(StartDate)=%v\n", po.EndDate.Before(po.StartDate), po.EndDate.Equal(po.StartDate))

	// Validar fechas
	if po.EndDate.Before(po.StartDate) || po.EndDate.Equal(po.StartDate) {
		return errors.New("la fecha de fin debe ser posterior a la fecha de inicio")
	}

	// No permitir ofertas en el pasado (con margen de 1 minuto)
	if po.StartDate.Before(time.Now().Add(-1 * time.Minute)) {
		return errors.New("no se pueden crear ofertas en el pasado")
	}

	// Validar valor del descuento según tipo
	if err := po.validateDiscountValue(); err != nil {
		return err
	}

	// Validar IDs
	if po.ProductID == uuid.Nil {
		return errors.New("product_id es requerido")
	}
	if po.CreatedBy == uuid.Nil {
		return errors.New("created_by es requerido")
	}

	return nil
}

// validateDiscountValue valida el valor del descuento según su tipo
func (po *ProductOffer) validateDiscountValue() error {
	switch po.DiscountType {
	case DiscountTypePercentage:
		if po.DiscountValue <= 0 || po.DiscountValue > 100 {
			return errors.New("el porcentaje de descuento debe estar entre 1 y 100")
		}
	case DiscountTypeFixedAmount:
		if po.DiscountValue <= 0 {
			return errors.New("el monto de descuento debe ser mayor a 0")
		}
	case DiscountTypeFixedPrice:
		if po.DiscountValue <= 0 {
			return errors.New("el precio fijo debe ser mayor a 0")
		}
	default:
		return errors.New("tipo de descuento inválido")
	}
	return nil
}

// BeforeCreate hook de GORM ejecutado antes de crear
func (po *ProductOffer) BeforeCreate(tx *gorm.DB) error {
	fmt.Printf("🔍 DEBUG BeforeCreate: StartDate=%v, EndDate=%v\n", po.StartDate, po.EndDate)

	// Si no se proporciona un ID, generamos uno
	if po.OfferID == uuid.Nil {
		po.OfferID = uuid.New()
	}
	// No validamos aquí porque las fechas pueden estar en valor cero
	// La validación se hace en el servicio antes de llamar a Create
	return nil
}

// BeforeUpdate hook de GORM ejecutado antes de actualizar
func (po *ProductOffer) BeforeUpdate(tx *gorm.DB) error {
	fmt.Printf("🔍 DEBUG BeforeUpdate: StartDate=%v, EndDate=%v\n", po.StartDate, po.EndDate)

	// Si las fechas están en valor cero, probablemente es una creación, no una actualización
	// No validamos en este caso para evitar errores
	if po.StartDate.IsZero() && po.EndDate.IsZero() {
		fmt.Printf("🔍 DEBUG BeforeUpdate: Skipping validation for zero dates (likely creation)\n")
		return nil
	}

	return po.Validate()
}
