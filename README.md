# API de ExactoGas

Backend para la aplicación de gestión de pedidos de gas a domicilio "ExactoGas".

## Tecnologías utilizadas

- **Go**: Lenguaje de programación
- **Fiber**: Framework web
- **GORM**: ORM para PostgreSQL
- **JWT**: Para autenticación
- **PostgreSQL**: Base de datos

## Requisitos previos

- Go 1.18 o superior
- PostgreSQL 13 o superior
- Air (opcional, para recarga automática en desarrollo)

## Configuración

1. Copia el archivo de ejemplo de configuración:

```bash
cp app.env.example app.env
```

2. Edita el archivo `app.env` con tus configuraciones.

## Ejecución de migraciones

Puedes ejecutar las migraciones de SQL manualmente con el script provisto:

```bash
./scripts/run_migrations.sh
```

O dejar que la aplicación las ejecute automáticamente al iniciar.

## Ejecución de la aplicación

### Modo desarrollo

Para ejecutar la aplicación en modo desarrollo con recarga automática:

```bash
./scripts/dev.sh
```

### Modo producción

Para compilar y ejecutar en modo producción:

```bash
go build -o exactogas-api
./exactogas-api
```

## Endpoints de la API

La API sigue las convenciones REST y devuelve respuestas en formato JSON.

### Autenticación

- `POST /api/v1/auth/register`: Registrar un nuevo usuario
- `POST /api/v1/auth/login`: Iniciar sesión y obtener tokens
- `POST /api/v1/auth/refresh`: Refrescar token de acceso

### Productos

- `GET /api/v1/products`: Obtener todos los productos
- `GET /api/v1/products/:id`: Obtener un producto por ID
- `POST /api/v1/products`: Crear un nuevo producto (Admin)
- `PUT /api/v1/products/:id`: Actualizar un producto (Admin)
- `DELETE /api/v1/products/:id`: Eliminar un producto (Admin)

### Pedidos

Próximamente...

## Autenticación y Autorización

La API utiliza tokens JWT para la autenticación. Los endpoints protegidos requieren un header de autorización con el formato:

```js
Authorization: Bearer <token>
```

Existen tres roles de usuario:

- `CLIENT`: Cliente normal que puede hacer pedidos
- `REPARTIDOR`: Repartidor que puede ver y gestionar pedidos asignados
- `ADMIN`: Administrador con acceso completo

## Ejemplos de uso

### Registro de usuario

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "usuario@ejemplo.com",
    "password": "contraseña123",
    "full_name": "Usuario Ejemplo",
    "phone_number": "987654321",
    "user_role": "CLIENT"
  }'
```

### Iniciar sesión

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "usuario@ejemplo.com",
    "password": "contraseña123"
  }'
```

### Obtener productos

```bash
curl -X GET http://localhost:8080/api/v1/products
```

## Desarrollo

La estructura del proyecto sigue una arquitectura monolítica modular:

- `api/`: Contiene los handlers y middlewares HTTP
- `config/`: Configuración de la aplicación
- `database/`: Conexión a la base de datos y migraciones
- `internal/`: Lógica de negocio y servicios
- `pkg/`: Código reutilizable y utilidades

Para añadir un nuevo módulo:

1. Crea los modelos en `internal/models/`
2. Implementa el servicio en `internal/[nombre_modulo]/`
3. Crea los handlers en `api/v1/handlers/`
4. Registra las rutas en `main.go`
