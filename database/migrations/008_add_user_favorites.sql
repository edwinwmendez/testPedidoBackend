-- Migration: 008_add_user_favorites.sql
-- Description: Crear tabla para gestionar productos favoritos de usuarios
-- Author: Sistema de Favoritos
-- Date: 2025-01-16

-- Crear tabla de favoritos de usuarios
CREATE TABLE IF NOT EXISTS user_favorites (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Clave primaria compuesta para evitar duplicados
    PRIMARY KEY (user_id, product_id)
);

-- Crear índices para optimizar consultas
CREATE INDEX IF NOT EXISTS idx_user_favorites_user_id ON user_favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_user_favorites_product_id ON user_favorites(product_id);
CREATE INDEX IF NOT EXISTS idx_user_favorites_created_at ON user_favorites(created_at);

-- Comentarios para documentación
COMMENT ON TABLE user_favorites IS 'Tabla para gestionar productos favoritos de usuarios';
COMMENT ON COLUMN user_favorites.user_id IS 'ID del usuario que marcó el producto como favorito';
COMMENT ON COLUMN user_favorites.product_id IS 'ID del producto marcado como favorito';
COMMENT ON COLUMN user_favorites.created_at IS 'Fecha y hora cuando se marcó como favorito';