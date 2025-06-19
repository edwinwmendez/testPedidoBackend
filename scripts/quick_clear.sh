#!/bin/bash

# Script rápido para limpiar órdenes desde la línea de comandos
# Uso: ./quick_clear.sh

echo "🗑️  Limpiando pedidos de la base de datos..."

# Asegúrate de que estas variables coincidan con tu configuración
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-exactogas_db}"
DB_USER="${DB_USER:-exactogas_user}"

echo "📡 Conectando a la base de datos..."
echo "   Host: $DB_HOST:$DB_PORT"
echo "   Database: $DB_NAME"
echo "   User: $DB_USER"

# Ejecutar los comandos SQL
psql -h "$DB_HOST" -p "$DB_PORT" -d "$DB_NAME" -U "$DB_USER" << EOF
-- Mostrar conteo actual
SELECT 'ANTES - Orders:' as info, COUNT(*) as count FROM orders
UNION ALL
SELECT 'ANTES - Order Items:' as info, COUNT(*) as count FROM order_items;

-- Eliminar todos los order_items primero
DELETE FROM order_items;

-- Eliminar todos los orders
DELETE FROM orders;

-- Mostrar conteo final
SELECT 'DESPUÉS - Orders:' as info, COUNT(*) as count FROM orders
UNION ALL
SELECT 'DESPUÉS - Order Items:' as info, COUNT(*) as count FROM order_items;

EOF

echo "✅ Limpieza completada."