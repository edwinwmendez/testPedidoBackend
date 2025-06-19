-- Migración 003: Agregar campo stock_quantity a la tabla products
-- Agregado el campo para el manejo de inventario

-- Agregar columna stock_quantity a la tabla products
ALTER TABLE products ADD COLUMN stock_quantity INTEGER NOT NULL DEFAULT 100 CHECK (stock_quantity >= 0);

-- Crear índice para búsquedas de stock
CREATE INDEX idx_products_stock_quantity ON products (stock_quantity);

-- Actualizar productos existentes con stock inicial
UPDATE products SET stock_quantity = 100 WHERE stock_quantity IS NULL;

-- Crear vista actualizada de productos activos incluyendo stock
CREATE OR REPLACE VIEW view_active_products AS
SELECT
    product_id,
    name,
    description,
    price,
    stock_quantity,
    image_url
FROM products
WHERE is_active = TRUE;

-- Función para validar stock disponible antes de crear orden
CREATE OR REPLACE FUNCTION check_product_stock_availability(p_product_id UUID, p_requested_quantity INTEGER)
RETURNS BOOLEAN AS $$
DECLARE
    current_stock INTEGER;
BEGIN
    SELECT stock_quantity INTO current_stock
    FROM products
    WHERE product_id = p_product_id AND is_active = TRUE;
    
    IF current_stock IS NULL THEN
        RAISE EXCEPTION 'Product not found or inactive';
    END IF;
    
    RETURN current_stock >= p_requested_quantity;
END;
$$ LANGUAGE plpgsql;

-- Función para actualizar stock después de una venta
CREATE OR REPLACE FUNCTION update_product_stock_on_sale()
RETURNS TRIGGER AS $$
BEGIN
    -- Solo actualizar stock cuando el pedido se marca como DELIVERED
    IF NEW.order_status = 'DELIVERED' AND OLD.order_status != 'DELIVERED' THEN
        -- Reducir stock de todos los productos en el pedido
        UPDATE products 
        SET stock_quantity = stock_quantity - oi.quantity
        FROM order_items oi
        WHERE products.product_id = oi.product_id 
        AND oi.order_id = NEW.order_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger para actualizar stock cuando un pedido se entrega
CREATE TRIGGER trigger_update_product_stock_on_delivery
AFTER UPDATE OF order_status ON orders
FOR EACH ROW
EXECUTE FUNCTION update_product_stock_on_sale();