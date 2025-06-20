-- Migración para agregar tabla de categorías
-- Fecha: 2024-06-20
-- Descripción: Agregar sistema de categorías para la tienda en línea PedidoMendez

-- Tabla de categorías
CREATE TABLE categories (
    category_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    icon_name VARCHAR(50) NOT NULL,
    color_hex VARCHAR(7) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Agregar campo category_id a la tabla products
ALTER TABLE products ADD COLUMN category_id UUID;

-- Agregar campo image_url si no existe (por compatibilidad)
ALTER TABLE products ADD COLUMN IF NOT EXISTS image_url VARCHAR(255);

-- Agregar campo stock_quantity si no existe (por compatibilidad)
ALTER TABLE products ADD COLUMN IF NOT EXISTS stock_quantity INTEGER NOT NULL DEFAULT 100 CHECK (stock_quantity >= 0);

-- Crear foreign key para categorías
ALTER TABLE products ADD CONSTRAINT fk_category 
    FOREIGN KEY (category_id) REFERENCES categories(category_id) ON DELETE SET NULL;

-- Índices para optimizar consultas
CREATE INDEX idx_categories_name ON categories (name);
CREATE INDEX idx_categories_is_active ON categories (is_active);
CREATE INDEX idx_products_category_id ON products (category_id);

-- Trigger para updated_at en categories
CREATE TRIGGER trigger_set_updated_at_categories
BEFORE UPDATE ON categories
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_timestamp();

-- Insertar categorías por defecto para la tienda PedidoMendez
INSERT INTO categories (name, description, icon_name, color_hex, is_active) VALUES
('Alimentos', 'Productos alimentarios y comestibles', 'food', '#FF6B6B', TRUE),
('Bebidas', 'Refrescos, jugos y bebidas en general', 'drinks', '#4ECDC4', TRUE),
('Limpieza', 'Productos de limpieza y hogar', 'cleaning', '#45B7D1', TRUE),
('Electrónicos', 'Dispositivos y accesorios electrónicos', 'electronics', '#96CEB4', TRUE),
('Hogar', 'Artículos para el hogar y decoración', 'home', '#FFEAA7', TRUE),
('Belleza', 'Productos de cuidado personal y belleza', 'beauty', '#DDA0DD', TRUE),
('Salud', 'Medicamentos y productos para la salud', 'health', '#98D8C8', TRUE),
('Deportes', 'Artículos deportivos y fitness', 'sports', '#F7DC6F', TRUE);

-- Actualizar productos existentes para asignarles categorías por defecto
-- Asignar categoría "Hogar" a productos existentes como categoría temporal
UPDATE products SET category_id = (
    SELECT category_id FROM categories WHERE name = 'Hogar' LIMIT 1
) WHERE category_id IS NULL;

-- Actualizar la descripción de productos existentes (cambiar referencias a gas)
UPDATE products SET 
    name = REPLACE(name, 'Balón de Gas', 'Producto'),
    description = REPLACE(description, 'gas doméstico', 'uso doméstico'),
    description = REPLACE(description, 'gas industrial', 'uso comercial')
WHERE name LIKE '%Balón de Gas%';

-- Vista actualizada para productos con categorías
CREATE OR REPLACE VIEW view_active_products AS
SELECT
    p.product_id,
    p.name,
    p.description,
    p.price,
    p.image_url,
    p.stock_quantity,
    p.category_id,
    c.name AS category_name,
    c.icon_name AS category_icon,
    c.color_hex AS category_color
FROM products p
LEFT JOIN categories c ON p.category_id = c.category_id
WHERE p.is_active = TRUE;

-- Vista para categorías activas con conteo de productos
CREATE OR REPLACE VIEW view_categories_with_product_count AS
SELECT
    c.category_id,
    c.name,
    c.description,
    c.icon_name,
    c.color_hex,
    c.is_active,
    c.created_at,
    c.updated_at,
    COUNT(p.product_id) AS product_count
FROM categories c
LEFT JOIN products p ON c.category_id = p.category_id AND p.is_active = TRUE
WHERE c.is_active = TRUE
GROUP BY c.category_id, c.name, c.description, c.icon_name, c.color_hex, c.is_active, c.created_at, c.updated_at
ORDER BY c.name;