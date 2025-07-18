-- =====================================================
-- Migración 009: Sistema de Ofertas Individuales
-- 
-- Descripción: Permite crear ofertas simples para productos individuales
-- Tipos soportados: percentage, fixed_amount, fixed_price
-- Regla: Solo una oferta activa por producto
-- =====================================================

-- Crear ENUM para tipos de descuento
CREATE TYPE offer_discount_type AS ENUM (
    'percentage',    -- Descuento porcentual (ej: 20%)
    'fixed_amount',  -- Monto fijo de descuento (ej: $5.00)
    'fixed_price'    -- Precio fijo final (ej: $15.99)
);

-- Tabla principal de ofertas por producto
CREATE TABLE product_offers (
    offer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    
    -- Configuración del descuento
    discount_type offer_discount_type NOT NULL,
    discount_value DECIMAL(10,2) NOT NULL,
    
    -- Vigencia de la oferta
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    
    -- Control administrativo
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(user_id),
    
    -- Auditoría
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Validaciones a nivel de base de datos
    CONSTRAINT valid_dates CHECK (end_date > start_date),
    CONSTRAINT valid_discount_value CHECK (discount_value > 0),
    CONSTRAINT valid_percentage CHECK (
        discount_type != 'percentage' OR 
        (discount_value > 0 AND discount_value <= 100)
    )
);

-- Índices para optimizar consultas
CREATE INDEX idx_product_offers_product_id ON product_offers(product_id);
CREATE INDEX idx_product_offers_active ON product_offers(is_active);
CREATE INDEX idx_product_offers_dates ON product_offers(start_date, end_date);
CREATE INDEX idx_product_offers_active_dates ON product_offers(is_active, start_date, end_date) 
WHERE is_active = true;

-- CONSTRAINT ÚNICO: Solo una oferta activa por producto
-- Esto previene conflictos y garantiza consistencia
CREATE UNIQUE INDEX idx_one_active_offer_per_product 
ON product_offers(product_id) 
WHERE is_active = true;

-- Función para actualizar updated_at automáticamente
CREATE OR REPLACE FUNCTION update_product_offers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger para actualizar updated_at en cada UPDATE
CREATE TRIGGER trigger_update_product_offers_updated_at
    BEFORE UPDATE ON product_offers
    FOR EACH ROW
    EXECUTE FUNCTION update_product_offers_updated_at();

-- Comentarios para documentación
COMMENT ON TABLE product_offers IS 'Ofertas individuales aplicadas a productos específicos';
COMMENT ON COLUMN product_offers.discount_type IS 'Tipo de descuento: percentage, fixed_amount, o fixed_price';
COMMENT ON COLUMN product_offers.discount_value IS 'Valor del descuento según el tipo especificado';
COMMENT ON COLUMN product_offers.start_date IS 'Fecha y hora de inicio de la oferta';
COMMENT ON COLUMN product_offers.end_date IS 'Fecha y hora de finalización de la oferta';
COMMENT ON INDEX idx_one_active_offer_per_product IS 'Garantiza solo una oferta activa por producto';