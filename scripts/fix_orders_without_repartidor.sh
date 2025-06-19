#!/bin/bash

# Script para arreglar pedidos en IN_TRANSIT sin repartidor asignado
# Este script encuentra esos pedidos y los vuelve a estado CONFIRMED

echo "Buscando pedidos en IN_TRANSIT sin repartidor asignado..."

# Conectar a la base de datos y ejecutar la consulta de fix
# Asume que tienes las variables de entorno configuradas para la conexión a PostgreSQL

PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
UPDATE orders 
SET order_status = 'CONFIRMED', 
    confirmed_at = COALESCE(confirmed_at, NOW())
WHERE order_status = 'IN_TRANSIT' 
  AND assigned_repartidor_id IS NULL;

SELECT 'Pedidos actualizados: ' || ROW_COUNT() as result;

-- Mostrar los pedidos que fueron actualizados
SELECT order_id, client_id, order_status, assigned_repartidor_id 
FROM orders 
WHERE order_status = 'CONFIRMED' 
  AND assigned_repartidor_id IS NULL 
  AND updated_at >= NOW() - INTERVAL '1 minute';
"

echo "Script completado. Los pedidos en IN_TRANSIT sin repartidor han sido devueltos a estado CONFIRMED."
echo "Ahora necesitas asignar un repartidor antes de cambiarlos a IN_TRANSIT nuevamente."