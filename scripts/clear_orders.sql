-- Script para limpiar todas las órdenes y sus items de la base de datos
-- ⚠️  CUIDADO: Esto eliminará TODOS los pedidos y sus items

BEGIN;

-- Eliminar todos los order_items primero (por la foreign key constraint)
DELETE FROM order_items;

-- Luego eliminar todos los orders
DELETE FROM orders;

-- Resetear las secuencias si las hay (opcional)
-- ALTER SEQUENCE orders_id_seq RESTART WITH 1;
-- ALTER SEQUENCE order_items_id_seq RESTART WITH 1;

-- Verificar que las tablas estén vacías
SELECT 'Orders count:' as table_info, COUNT(*) as count FROM orders
UNION ALL
SELECT 'Order items count:' as table_info, COUNT(*) as count FROM order_items;

COMMIT;

-- Si algo sale mal, puedes usar ROLLBACK; en lugar de COMMIT;