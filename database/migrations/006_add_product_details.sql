-- Migración 006: Agregar campos básicos de detalle para productos MVP
-- Campos para productos de primera necesidad (supermercado/tienda)

-- Enum para unidades de medida comunes
CREATE TYPE unit_of_measure AS ENUM (
    'kg',           -- kilogramos
    'g',            -- gramos
    'l',            -- litros
    'ml',           -- mililitros
    'unidad',       -- unidades individuales
    'paquete',      -- paquetes
    'caja',         -- cajas
    'bolsa',        -- bolsas
    'lata',         -- latas
    'botella',      -- botellas
    'sobre',        -- sobres
    'rollo',        -- rollos
    'docena',       -- docenas
    'par'           -- pares
);

-- Agregar nuevos campos a la tabla products
ALTER TABLE products ADD COLUMN description TEXT;
ALTER TABLE products ADD COLUMN brand VARCHAR(100);
ALTER TABLE products ADD COLUMN unit_of_measure unit_of_measure DEFAULT 'unidad';
ALTER TABLE products ADD COLUMN net_weight DECIMAL(10,3); -- peso neto en kg
ALTER TABLE products ADD COLUMN net_volume DECIMAL(10,3); -- volumen neto en litros
ALTER TABLE products ADD COLUMN package_size VARCHAR(50); -- tamaño del empaque (ej: "500g", "1L", "Pack x12")
ALTER TABLE products ADD COLUMN ingredients TEXT; -- ingredientes/componentes
ALTER TABLE products ADD COLUMN expiration_days INTEGER; -- días de vencimiento desde producción
ALTER TABLE products ADD COLUMN origin_country VARCHAR(100); -- país de origen
ALTER TABLE products ADD COLUMN barcode VARCHAR(50); -- código de barras
ALTER TABLE products ADD COLUMN nutritional_info JSONB; -- información nutricional
ALTER TABLE products ADD COLUMN storage_instructions TEXT; -- instrucciones de almacenamiento

-- Índices para mejorar búsquedas
CREATE INDEX idx_products_brand ON products(brand);
CREATE INDEX idx_products_unit_measure ON products(unit_of_measure);
CREATE INDEX idx_products_barcode ON products(barcode);

-- Comentarios para documentar campos
COMMENT ON COLUMN products.description IS 'Descripción detallada del producto';
COMMENT ON COLUMN products.brand IS 'Marca o fabricante del producto';
COMMENT ON COLUMN products.unit_of_measure IS 'Unidad de medida para la venta';
COMMENT ON COLUMN products.net_weight IS 'Peso neto en kilogramos';
COMMENT ON COLUMN products.net_volume IS 'Volumen neto en litros';
COMMENT ON COLUMN products.package_size IS 'Tamaño del empaque (ej: 500g, 1L, Pack x12)';
COMMENT ON COLUMN products.ingredients IS 'Lista de ingredientes o componentes';
COMMENT ON COLUMN products.expiration_days IS 'Días de vencimiento desde fecha de producción';
COMMENT ON COLUMN products.origin_country IS 'País de origen del producto';
COMMENT ON COLUMN products.barcode IS 'Código de barras del producto';
COMMENT ON COLUMN products.nutritional_info IS 'Información nutricional en formato JSON';
COMMENT ON COLUMN products.storage_instructions IS 'Instrucciones de almacenamiento y conservación';