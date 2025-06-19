-- Script SQL para corregir pedidos en IN_TRANSIT sin repartidor asignado
-- Ejecutar este script directamente en la base de datos

-- Mostrar pedidos problemáticos antes de la corrección
SELECT 
    order_id, 
    client_id, 
    order_status, 
    assigned_repartidor_id,
    created_at
FROM orders 
WHERE order_status = 'IN_TRANSIT' 
  AND assigned_repartidor_id IS NULL;

-- Actualizar pedidos problemáticos de IN_TRANSIT a CONFIRMED
UPDATE orders 
SET 
    order_status = 'CONFIRMED',
    confirmed_at = COALESCE(confirmed_at, NOW()),
    updated_at = NOW()
WHERE order_status = 'IN_TRANSIT' 
  AND assigned_repartidor_id IS NULL;

-- Mostrar resultado de la actualización
SELECT 
    'Pedidos corregidos: ' || ROW_COUNT() as result;

-- Verificar que la corrección se aplicó correctamente
SELECT 
    order_id, 
    client_id, 
    order_status, 
    assigned_repartidor_id,
    confirmed_at,
    updated_at
FROM orders 
WHERE order_status = 'CONFIRMED' 
  AND assigned_repartidor_id IS NULL 
  AND updated_at >= NOW() - INTERVAL '1 minute';