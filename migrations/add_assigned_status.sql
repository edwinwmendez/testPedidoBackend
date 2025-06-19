-- Migración para agregar el estado ASSIGNED y campo assigned_at
-- Fecha: 2025-01-15
-- Descripción: Añade el nuevo estado ASSIGNED al flujo de órdenes y el campo assigned_at

-- 1. Agregar el campo assigned_at a la tabla orders
ALTER TABLE orders ADD COLUMN assigned_at TIMESTAMP NULL;

-- 2. Actualizar la restricción CHECK para incluir el nuevo estado ASSIGNED
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_order_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_order_status_check 
    CHECK (order_status IN ('PENDING', 'PENDING_OUT_OF_HOURS', 'CONFIRMED', 'ASSIGNED', 'IN_TRANSIT', 'DELIVERED', 'CANCELLED'));

-- 3. Crear índice para el nuevo campo assigned_at (para consultas de rendimiento)
CREATE INDEX IF NOT EXISTS idx_orders_assigned_at ON orders(assigned_at);

-- 4. Actualizar pedidos existentes que estén en estado CONFIRMED con repartidor asignado a ASSIGNED
-- Esto es opcional y solo si quieres migrar datos existentes
UPDATE orders 
SET order_status = 'ASSIGNED', 
    assigned_at = COALESCE(confirmed_at, updated_at)
WHERE order_status = 'CONFIRMED' 
  AND assigned_repartidor_id IS NOT NULL;

-- 5. Comentarios para documentación
COMMENT ON COLUMN orders.assigned_at IS 'Timestamp cuando se asignó el repartidor al pedido';
COMMENT ON CONSTRAINT orders_order_status_check ON orders IS 'Estados válidos: PENDING, PENDING_OUT_OF_HOURS, CONFIRMED, ASSIGNED, IN_TRANSIT, DELIVERED, CANCELLED';

-- Verificar la migración
SELECT 
    order_id,
    order_status,
    assigned_repartidor_id,
    confirmed_at,
    assigned_at,
    created_at
FROM orders 
WHERE order_status IN ('CONFIRMED', 'ASSIGNED')
ORDER BY created_at DESC
LIMIT 10;