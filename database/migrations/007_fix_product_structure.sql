-- Migración 007: Arreglar estructura de productos
-- Remover detailed_description si existe y asegurar que package_size esté presente

-- Eliminar detailed_description si existe
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'products' 
        AND column_name = 'detailed_description'
    ) THEN
        ALTER TABLE products DROP COLUMN detailed_description;
    END IF;
END $$;

-- Agregar package_size si no existe
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'products' 
        AND column_name = 'package_size'
    ) THEN
        ALTER TABLE products ADD COLUMN package_size VARCHAR(50);
    END IF;
END $$;

-- Agregar comentarios para documentar los campos
COMMENT ON COLUMN products.package_size IS 'Tamaño del empaque (ej: 500g, 1L, Pack x12)';