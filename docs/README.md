# Documentación de la API de ExactoGas

Este directorio contiene la documentación generada automáticamente para la API de ExactoGas utilizando Swagger.

## Acceso a la Documentación

La documentación de la API está disponible en la siguiente URL cuando el servidor está en ejecución:

```
http://localhost:8080/swagger/
```

## Características

- Documentación interactiva de todos los endpoints
- Posibilidad de probar los endpoints directamente desde la interfaz
- Autenticación mediante JWT
- Descripción detallada de los parámetros de entrada y respuestas

## Endpoints Documentados

La documentación incluye los siguientes grupos de endpoints:

- **Autenticación**: Registro, inicio de sesión y refresco de tokens
- **Productos**: Listado, creación, actualización y eliminación de productos
- **Pedidos**: Creación, listado y gestión de pedidos
- **Usuarios**: Gestión de perfiles de usuario

## Generación de la Documentación

La documentación se genera automáticamente a partir de las anotaciones en el código fuente utilizando Swaggo. Para regenerar la documentación después de hacer cambios en el código, ejecuta el siguiente comando:

```bash
swag init -g main.go
```

## Estructura de Archivos

- `docs.go`: Archivo principal de la documentación generada
- `swagger.json`: Especificación de la API en formato JSON
- `swagger.yaml`: Especificación de la API en formato YAML

## Personalización

Para personalizar la documentación, puedes modificar las anotaciones en los archivos de código fuente. Las anotaciones principales se encuentran en:

- `main.go`: Configuración general de la API
- Archivos de handlers: Documentación específica de cada endpoint

## Autenticación

Para probar endpoints que requieren autenticación:

1. Utiliza el endpoint `/auth/login` para obtener un token JWT
2. Haz clic en el botón "Authorize" en la parte superior de la página
3. Ingresa el token en el formato `Bearer {token}` 
4. Ahora puedes probar los endpoints protegidos

## Referencias

- [Swagger UI](https://swagger.io/tools/swagger-ui/)
- [Swaggo](https://github.com/swaggo/swag)
- [Fiber Swagger](https://github.com/swaggo/fiber-swagger) 