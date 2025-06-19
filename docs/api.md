# Documentación de la API de ExactoGas

Esta documentación describe los endpoints disponibles en la API RESTful de ExactoGas, sus parámetros, formatos de solicitud y respuesta, así como los códigos de estado HTTP que pueden devolver.

## Configuración de Red - Mobile App

### Conectividad Dinámica para Dispositivos Físicos

La aplicación móvil incluye un sistema de detección automática de IP para conectarse al backend desde dispositivos físicos en diferentes redes WiFi:

**NetworkConfig** (`frontend/appgas_mobile/lib/config/network_config.dart`):
- Detecta automáticamente la IP del servidor backend
- Maneja emuladores Android (usa `10.0.2.2`)
- Prueba múltiples IPs candidatas en orden de prioridad
- Incluye sistema de caché para mejorar performance
- Permite configuración manual para debug

**IPs de respaldo configuradas**:
1. `192.168.169.244` - IP actual detectada
2. `192.168.1.161` - IP anterior  
3. `192.168.0.1` - Router común
4. `10.0.0.1` - Otro router común

**Servicios actualizados**:
- `ApiService` - Usa detección automática para todas las peticiones HTTP
- `WebSocketService` - Conecta automáticamente al WebSocket con IP dinámica
- `ApiConfig` - Interfaz unificada para configuración de red

**Debug**:
- Pantalla de debug disponible en `/screens/debug/network_debug_screen.dart`
- Permite ver configuración actual y forzar IPs específicas
- Logs detallados en consola para troubleshooting

## URL Base

```
http://localhost:8080/api/v1
```

## Autenticación

La mayoría de los endpoints requieren autenticación mediante un token JWT. Este token debe ser incluido en el encabezado `Authorization` de la siguiente manera:

```
Authorization: Bearer <token>
```

El token se obtiene mediante el endpoint de inicio de sesión (`/auth/login`).

## Endpoints

### Salud del Sistema

#### `GET /health`

Verifica el estado del servidor.

**Respuesta exitosa (200 OK)**

```json
{
  "status": "ok",
  "message": "ExactoGas API está funcionando correctamente"
}
```

### Autenticación

#### `POST /auth/register`

Registra un nuevo usuario en el sistema.

**Cuerpo de la solicitud**

```json
{
  "full_name": "Nombre Completo",
  "email": "usuario@ejemplo.com",
  "phone_number": "999888777",
  "password": "contraseña123",
  "user_role": "CLIENT"  // CLIENT, REPARTIDOR, ADMIN
}
```

**Respuesta exitosa (201 Created)**

```json
{
  "message": "Usuario registrado exitosamente",
  "user_id": "uuid-del-usuario"
}
```

**Respuestas de error**

- `400 Bad Request`: Datos de entrada inválidos
- `409 Conflict`: El email o teléfono ya está registrado

#### `POST /auth/login`

Inicia sesión y obtiene un token JWT.

**Cuerpo de la solicitud**

```json
{
  "email": "usuario@ejemplo.com",
  "password": "contraseña123"
}
```

**Respuesta exitosa (200 OK)**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

**Respuestas de error**

- `400 Bad Request`: Datos de entrada inválidos
- `401 Unauthorized`: Email o contraseña incorrectos

#### `POST /auth/refresh`

Refresca un token JWT expirado.

**Cuerpo de la solicitud**

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Respuesta exitosa (200 OK)**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

**Respuestas de error**

- `400 Bad Request`: Datos de entrada inválidos
- `401 Unauthorized`: Token inválido o expirado

### Usuarios

#### `GET /users/me`

Obtiene la información del usuario autenticado.

**Requiere autenticación**: Sí

**Respuesta exitosa (200 OK)**

```json
{
  "user_id": "uuid-del-usuario",
  "email": "usuario@ejemplo.com",
  "full_name": "Nombre Completo",
  "phone_number": "999888777",
  "user_role": "CLIENT",
  "is_active": true,
  "created_at": "2025-06-12T17:24:33.726976-05:00",
  "updated_at": "2025-06-12T17:24:33.726976-05:00"
}
```

**Respuestas de error**

- `401 Unauthorized`: Token inválido o expirado

#### `PUT /users/me`

Actualiza la información del usuario autenticado.

**Requiere autenticación**: Sí

**Cuerpo de la solicitud**

```json
{
  "full_name": "Nuevo Nombre",
  "phone_number": "999888777",
  "password": "nueva_contraseña"  // Opcional
}
```

**Respuesta exitosa (200 OK)**

```json
{
  "user_id": "uuid-del-usuario",
  "email": "usuario@ejemplo.com",
  "full_name": "Nuevo Nombre",
  "phone_number": "999888777",
  "user_role": "CLIENT",
  "is_active": true,
  "created_at": "2025-06-12T17:24:33.726976-05:00",
  "updated_at": "2025-06-12T17:25:33.726976-05:00"
}
```

**Respuestas de error**

- `400 Bad Request`: Datos de entrada inválidos
- `401 Unauthorized`: Token inválido o expirado

### Productos

#### `GET /products`

Obtiene la lista de productos disponibles.

**Parámetros de consulta**

- `active` (opcional): Si es "true", solo devuelve productos activos

**Respuesta exitosa (200 OK)**

```json
[
  {
    "product_id": "uuid-del-producto",
    "name": "Balón de Gas 10kg",
    "description": "Balón de gas doméstico de 10kg",
    "price": 45,
    "stock_quantity": 100,
    "image_url": "",
    "is_active": true,
    "created_at": "2025-06-12T17:24:33.726976-05:00",
    "updated_at": "2025-06-12T17:24:33.726976-05:00"
  },
  // ... más productos
]
```

#### `GET /products/:id`

Obtiene los detalles de un producto específico.

**Parámetros de ruta**

- `id`: ID del producto

**Respuesta exitosa (200 OK)**

```json
{
  "product_id": "uuid-del-producto",
  "name": "Balón de Gas 10kg",
  "description": "Balón de gas doméstico de 10kg",
  "price": 45,
  "stock_quantity": 100,
  "image_url": "",
  "is_active": true,
  "created_at": "2025-06-12T17:24:33.726976-05:00",
  "updated_at": "2025-06-12T17:24:33.726976-05:00"
}
```

**Respuestas de error**

- `404 Not Found`: Producto no encontrado

#### `POST /products`

Crea un nuevo producto (solo administradores).

**Requiere autenticación**: Sí (ADMIN)

**Cuerpo de la solicitud**

```json
{
  "name": "Balón de Gas 5kg",
  "description": "Balón de gas doméstico de 5kg",
  "price": 30,
  "stock_quantity": 100,
  "image_url": "",
  "is_active": true
}
```

**Respuesta exitosa (201 Created)**

```json
{
  "product_id": "uuid-del-producto",
  "name": "Balón de Gas 5kg",
  "description": "Balón de gas doméstico de 5kg",
  "price": 30,
  "stock_quantity": 100,
  "image_url": "",
  "is_active": true,
  "created_at": "2025-06-12T17:24:33.726976-05:00",
  "updated_at": "2025-06-12T17:24:33.726976-05:00"
}
```

**Respuestas de error**

- `400 Bad Request`: Datos de entrada inválidos
- `401 Unauthorized`: Token inválido o expirado
- `403 Forbidden`: No tiene permisos de administrador

#### `PUT /products/:id`

Actualiza un producto existente (solo administradores).

**Requiere autenticación**: Sí (ADMIN)

**Parámetros de ruta**

- `id`: ID del producto

**Cuerpo de la solicitud**

```json
{
  "name": "Balón de Gas 5kg - Nuevo",
  "description": "Balón de gas doméstico de 5kg actualizado",
  "price": 35,
  "stock_quantity": 150,
  "image_url": "",
  "is_active": true
}
```

**Respuesta exitosa (200 OK)**

```json
{
  "product_id": "uuid-del-producto",
  "name": "Balón de Gas 5kg - Nuevo",
  "description": "Balón de gas doméstico de 5kg actualizado",
  "price": 35,
  "stock_quantity": 150,
  "image_url": "",
  "is_active": true,
  "created_at": "2025-06-12T17:24:33.726976-05:00",
  "updated_at": "2025-06-12T17:25:33.726976-05:00"
}
```

**Respuestas de error**

- `400 Bad Request`: Datos de entrada inválidos
- `401 Unauthorized`: Token inválido o expirado
- `403 Forbidden`: No tiene permisos de administrador
- `404 Not Found`: Producto no encontrado

#### `DELETE /products/:id`

Elimina un producto (solo administradores).

**Requiere autenticación**: Sí (ADMIN)

**Parámetros de ruta**

- `id`: ID del producto

**Respuesta exitosa (204 No Content)**

**Respuestas de error**

- `401 Unauthorized`: Token inválido o expirado
- `403 Forbidden`: No tiene permisos de administrador
- `404 Not Found`: Producto no encontrado

### Pedidos

#### `POST /orders`

Crea un nuevo pedido.

**Requiere autenticación**: Sí (CLIENT)

**Cuerpo de la solicitud**

```json
{
  "items": [
    {
      "product_id": "uuid-del-producto",
      "quantity": 2
    },
    {
      "product_id": "uuid-de-otro-producto",
      "quantity": 1
    }
  ],
  "latitude": -10.123456,
  "longitude": -75.123456,
  "delivery_address_text": "Calle Principal 123, Atalaya",
  "payment_note": "Pago con billete de 100 soles"
}
```

**Respuesta exitosa (201 Created)**

```json
{
  "order_id": "uuid-del-pedido",
  "client_id": "uuid-del-cliente",
  "total_amount": 125,
  "latitude": -10.123456,
  "longitude": -75.123456,
  "delivery_address_text": "Calle Principal 123, Atalaya",
  "payment_note": "Pago con billete de 100 soles",
  "order_status": "PENDING",
  "order_time": "2025-06-12T17:24:33.726976-05:00",
  "created_at": "2025-06-12T17:24:33.726976-05:00",
  "updated_at": "2025-06-12T17:24:33.726976-05:00",
  "items": [
    {
      "product_id": "uuid-del-producto",
      "name": "Balón de Gas 10kg",
      "quantity": 2,
      "unit_price": 45,
      "subtotal": 90
    },
    {
      "product_id": "uuid-de-otro-producto",
      "name": "Balón de Gas 30kg",
      "quantity": 1,
      "unit_price": 35,
      "subtotal": 35
    }
  ]
}
```

**Respuestas de error**

- `400 Bad Request`: Datos de entrada inválidos
- `401 Unauthorized`: Token inválido o expirado
- `404 Not Found`: Producto no encontrado

#### `GET /orders`

Obtiene la lista de pedidos según el rol del usuario.

**Requiere autenticación**: Sí (CLIENT, REPARTIDOR, ADMIN)

**Parámetros de consulta**

- `status` (opcional): Filtra por estado del pedido
- `assigned_repartidor_id` (opcional, solo ADMIN): Filtra por repartidor asignado
- `client_id` (opcional, solo ADMIN): Filtra por cliente

**Respuesta exitosa (200 OK)**

```json
[
  {
    "order_id": "uuid-del-pedido",
    "client_id": "uuid-del-cliente",
    "total_amount": 125,
    "delivery_address_text": "Calle Principal 123, Atalaya",
    "order_status": "PENDING",
    "order_time": "2025-06-12T17:24:33.726976-05:00",
    "confirmed_at": null,
    "estimated_arrival_time": null,
    "assigned_repartidor_id": null,
    "delivered_at": null,
    "cancelled_at": null,
    "client_name": "Nombre del Cliente",
    "repartidor_name": null
  },
  // ... más pedidos
]
```

**Respuestas de error**

- `401 Unauthorized`: Token inválido o expirado

#### `GET /orders/:id`

Obtiene los detalles de un pedido específico.

**Requiere autenticación**: Sí (CLIENT - solo sus pedidos, REPARTIDOR - solo sus pedidos asignados o pendientes, ADMIN - cualquier pedido)

**Parámetros de ruta**

- `id`: ID del pedido

**Respuesta exitosa (200 OK)**

```json
{
  "order_id": "uuid-del-pedido",
  "client_id": "uuid-del-cliente",
  "total_amount": 125,
  "latitude": -10.123456,
  "longitude": -75.123456,
  "delivery_address_text": "Calle Principal 123, Atalaya",
  "payment_note": "Pago con billete de 100 soles",
  "order_status": "PENDING",
  "order_time": "2025-06-12T17:24:33.726976-05:00",
  "confirmed_at": null,
  "estimated_arrival_time": null,
  "assigned_repartidor_id": null,
  "delivered_at": null,
  "cancelled_at": null,
  "created_at": "2025-06-12T17:24:33.726976-05:00",
  "updated_at": "2025-06-12T17:24:33.726976-05:00",
  "client": {
    "user_id": "uuid-del-cliente",
    "full_name": "Nombre del Cliente",
    "phone_number": "999888777"
  },
  "repartidor": null,
  "items": [
    {
      "order_item_id": "uuid-del-item",
      "product_id": "uuid-del-producto",
      "product_name": "Balón de Gas 10kg",
      "quantity": 2,
      "unit_price": 45,
      "subtotal": 90
    },
    {
      "order_item_id": "uuid-del-item-2",
      "product_id": "uuid-de-otro-producto",
      "product_name": "Balón de Gas 30kg",
      "quantity": 1,
      "unit_price": 35,
      "subtotal": 35
    }
  ]
}
```

**Respuestas de error**

- `401 Unauthorized`: Token inválido o expirado
- `403 Forbidden`: No tiene permisos para ver este pedido
- `404 Not Found`: Pedido no encontrado

#### `PUT /orders/:id/status`

Actualiza el estado de un pedido.

**Requiere autenticación**: Sí (CLIENT - solo puede cancelar sus pedidos pendientes, REPARTIDOR - puede confirmar, marcar en tránsito y entregado, ADMIN - puede cambiar a cualquier estado)

**Parámetros de ruta**

- `id`: ID del pedido

**Cuerpo de la solicitud**

```json
{
  "new_status": "CONFIRMED",
  "estimated_arrival_time": "2025-06-12T18:30:00-05:00"  // Opcional, solo si el repartidor/admin confirma
}
```

**Respuesta exitosa (200 OK)**

```json
{
  "order_id": "uuid-del-pedido",
  "client_id": "uuid-del-cliente",
  "total_amount": 125,
  "latitude": -10.123456,
  "longitude": -75.123456,
  "delivery_address_text": "Calle Principal 123, Atalaya",
  "payment_note": "Pago con billete de 100 soles",
  "order_status": "CONFIRMED",
  "order_time": "2025-06-12T17:24:33.726976-05:00",
  "confirmed_at": "2025-06-12T17:30:33.726976-05:00",
  "estimated_arrival_time": "2025-06-12T18:30:00-05:00",
  "assigned_repartidor_id": "uuid-del-repartidor",
  "delivered_at": null,
  "cancelled_at": null,
  "created_at": "2025-06-12T17:24:33.726976-05:00",
  "updated_at": "2025-06-12T17:30:33.726976-05:00"
}
```

**Respuestas de error**

- `400 Bad Request`: Estado inválido o transición no permitida
- `401 Unauthorized`: Token inválido o expirado
- `403 Forbidden`: No tiene permisos para cambiar el estado
- `404 Not Found`: Pedido no encontrado

#### `POST /orders/:id/assign`

Asigna un repartidor a un pedido.

**Requiere autenticación**: Sí (REPARTIDOR - se autoasigna, ADMIN - asigna a cualquier repartidor)

**Parámetros de ruta**

- `id`: ID del pedido

**Cuerpo de la solicitud**

```json
{
  "repartidor_id": "uuid-del-repartidor"  // Opcional para REPARTIDOR, requerido para ADMIN
}
```

**Respuesta exitosa (200 OK)**

```json
{
  "order_id": "uuid-del-pedido",
  "client_id": "uuid-del-cliente",
  "total_amount": 125,
  "latitude": -10.123456,
  "longitude": -75.123456,
  "delivery_address_text": "Calle Principal 123, Atalaya",
  "payment_note": "Pago con billete de 100 soles",
  "order_status": "CONFIRMED",
  "order_time": "2025-06-12T17:24:33.726976-05:00",
  "confirmed_at": "2025-06-12T17:30:33.726976-05:00",
  "estimated_arrival_time": null,
  "assigned_repartidor_id": "uuid-del-repartidor",
  "delivered_at": null,
  "cancelled_at": null,
  "created_at": "2025-06-12T17:24:33.726976-05:00",
  "updated_at": "2025-06-12T17:30:33.726976-05:00"
}
```

**Respuestas de error**

- `400 Bad Request`: Pedido ya asignado o en estado no asignable
- `401 Unauthorized`: Token inválido o expirado
- `403 Forbidden`: No tiene permisos para asignar
- `404 Not Found`: Pedido o repartidor no encontrado

## Códigos de Estado HTTP

- `200 OK`: La solicitud se ha completado correctamente
- `201 Created`: El recurso se ha creado correctamente
- `204 No Content`: La solicitud se ha completado correctamente pero no hay contenido para devolver
- `400 Bad Request`: La solicitud contiene datos inválidos o incompletos
- `401 Unauthorized`: No se ha proporcionado un token válido
- `403 Forbidden`: El token es válido pero no tiene permisos para acceder al recurso
- `404 Not Found`: El recurso solicitado no existe
- `409 Conflict`: La solicitud no se puede completar debido a un conflicto con el estado actual del recurso
- `500 Internal Server Error`: Error interno del servidor

## Cambios Recientes en la API

### Usuarios
- **Campo `is_active` agregado**: Todos los usuarios ahora incluyen un campo `is_active` (boolean) que indica si la cuenta está activa y puede acceder al sistema.
- **Control de acceso**: Los usuarios inactivos no pueden iniciar sesión ni realizar operaciones en el sistema.

### Productos
- **Campo `stock_quantity` agregado**: Todos los productos ahora incluyen un campo `stock_quantity` (integer) que representa la cantidad disponible en inventario.
- **Validación de stock**: El sistema valida que haya suficiente stock disponible antes de crear pedidos.
- **Gestión automática de inventario**: El stock se reduce automáticamente cuando los pedidos se marcan como entregados mediante triggers de base de datos.
- **Restricciones**: El campo `stock_quantity` debe ser un valor no negativo (>= 0).

### Estados de Pedidos
- **Estado `ASSIGNED` agregado**: Nuevo estado intermedio entre `CONFIRMED` y `IN_TRANSIT` que indica que un pedido tiene repartidor asignado pero aún no ha iniciado la entrega.
- **Campo `assigned_at` agregado**: Timestamp que registra cuándo se asignó un repartidor al pedido.
- **Flujo de estados actualizado**: PENDING → CONFIRMED → ASSIGNED → IN_TRANSIT → DELIVERED

### Control de Permisos
- **Permisos estrictos por rol**: Implementación de control de acceso granular donde cada rol tiene permisos específicos para cambios de estado.
- **Validación dual**: Los permisos se validan tanto en frontend como en backend para máxima seguridad.
- **Aislamiento de sesiones**: Las sesiones están aisladas por pestaña en navegadores web para prevenir interferencia entre usuarios. 