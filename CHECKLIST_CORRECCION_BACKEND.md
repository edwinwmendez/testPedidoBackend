# Checklist de Corrección y Mejora Detallado: Backend testPedidoBackend

**Fuente:** INFORME_AUDITORIA_BACKEND_COMPLETO.md (Claude)

Este checklist proporciona un plan de acción priorizado para abordar los hallazgos de la auditoría de código detallada.

## Fase 1: Correcciones Críticas de Estabilidad y Seguridad (Prioridad Máxima - Realizar en Semana 1)

- [ ] **1.1. Eliminar Riesgo de Pánico del Servidor por UUIDs Inválidos**
    - **Ubicación:** `api/v1/handlers/order_handler.go` (Función `CreateOrder`).
    - **Qué hacer:** Reemplazar todas las llamadas a `uuid.MustParse()` por `uuid.Parse()`. Capturar el error devuelto y responder con un `400 Bad Request` si el parseo falla.
    - **Por qué:** Evita que una solicitud malformada o un token con un `UserID` inválido causen un pánico que detenga por completo el servidor. Es la corrección más crítica para la estabilidad de la API.
    - **Resultado esperado:** El servidor es resiliente a UUIDs inválidos y responde con errores HTTP apropiados en lugar de fallar.

- [ ] **1.2. Solucionar Problema de Rendimiento N+1 en Carga de Pedidos**
    - **Ubicación:** `internal/repositories/order_repository.go` (Función `FindByID`).
    - **Qué hacer:** Modificar la consulta de GORM para usar `Preload("OrderItems.Product")`. Esto cargará el pedido, todos sus items y todos los productos asociados en 3 consultas eficientes en lugar de N+2.
    - **Por qué:** Mejora drásticamente el rendimiento de la carga de detalles de pedidos, especialmente para pedidos con muchos items. Es esencial para la escalabilidad.
    - **Resultado esperado:** Reducción significativa de la latencia en el endpoint `GET /orders/{id}`.

- [ ] **1.3. Eliminar Exposición de Tokens JWT en Logs**
    - **Ubicación:** `internal/ws/handler.go` (Línea ~16).
    - **Qué hacer:** Eliminar o comentar la línea `log.Printf("[WebSocket] Token recibido: %s", token)`.
    - **Por qué:** Los tokens de sesión son información sensible. Exponerlos en logs representa un riesgo de seguridad significativo si los logs son accedidos por personal no autorizado.
    - **Resultado esperado:** Los logs del servidor ya no contienen tokens de sesión, mejorando la seguridad.

- [ ] **1.4. Implementar Transacciones Atómicas en Creación de Pedidos**
    - **Ubicación:** `internal/services/order_service.go` (Función `CreateOrder`).
    - **Qué hacer:** Envolver la lógica de creación del pedido y sus items dentro de una transacción de base de datos (`db.Transaction`).
    - **Por qué:** Garantiza que la creación de un pedido sea una operación atómica. Si falla la creación de un `OrderItem`, toda la operación se revierte, evitando pedidos "huérfanos" o inconsistentes en la base de datos.
    - **Resultado esperado:** Mayor integridad y consistencia de los datos.

- [ ] **1.5. Corregir Condición de Carrera (Race Condition) en WebSocket Hub**
    - **Ubicación:** `internal/ws/hub.go` (Función `SendToRole`).
    - **Qué hacer:** Cambiar el `h.mu.RLock()` por un `h.mu.Lock()` al inicio de la función y `h.mu.RUnlock()` por `h.mu.Unlock()` al final. Se está modificando el mapa de clientes (`delete`) dentro de un bucle que solo tiene un bloqueo de lectura.
    - **Por qué:** Previene una condición de carrera que puede ocurrir cuando múltiples goroutines intentan leer y escribir en los mapas de clientes simultáneamente, lo que puede causar un pánico.
    - **Resultado esperado:** El WebSocket Hub es seguro para el uso concurrente y estable.

- [ ] **1.6. Restringir CORS en Producción**
    - **Ubicación:** `main.go`.
    - **Qué hacer:** Modificar la configuración de CORS para que en el entorno de producción (`RENDER=true`), solo se permitan los dominios específicos de la aplicación frontend.
    - **Por qué:** Previene que sitios web maliciosos realicen solicitudes a la API desde el navegador de un usuario.
    - **Resultado esperado:** Mayor seguridad al limitar las solicitudes de origen cruzado a fuentes confiables.

## Fase 2: Mejoras de Robustez y Lógica de Negocio (Prioridad Alta - Realizar en Semana 2-3)

- [ ] **2.1. Implementar Validación Estructurada con Tags**
    - **Ubicación:** Todos los archivos en `api/v1/handlers/`.
    - **Qué hacer:** Añadir la librería `go-playground/validator`. Reemplazar los bloques `if` de validación manual por etiquetas `validate` en los structs de solicitud y añadir un validador global.
    - **Por qué:** Centraliza y simplifica la lógica de validación, haciendo el código más limpio, declarativo y fácil de mantener.
    - **Resultado esperado:** Código de validación más robusto y menos propenso a errores.

- [ ] **2.2. Implementar Lógica de Devolución de Stock al Cancelar Pedido**
    - **Ubicación:** `internal/services/order_service.go` (Función `UpdateOrderStatus`).
    - **Qué hacer:** Añadir lógica para reponer el stock de los productos cuando un pedido es cancelado. La cantidad a reponer debe ser leída de los `OrderItems` del pedido.
    - **Por qué:** Evita discrepancias en el inventario, lo que es crucial para la lógica de negocio.
    - **Resultado esperado:** El stock se gestiona de forma precisa tanto en ventas como en cancelaciones.

- [ ] **2.3. Corregir Falla de Integridad de Datos en Carga de Pedidos**
    - **Ubicación:** `internal/repositories/order_repository.go` (Función `FindByID`).
    - **Qué hacer:** Este punto se soluciona al implementar la corrección 1.2 (usar `Preload`). Si no se usa `Preload`, se debe modificar el bucle para que devuelva un error si no se encuentra un producto, en lugar de asignar un struct vacío.
    - **Por qué:** Asegura que la API nunca devuelva datos inconsistentes o incompletos.
    - **Resultado esperado:** La API es más confiable y no oculta problemas de integridad de datos.

## Fase 3: Mantenibilidad y Calidad de Código (Prioridad Media)

- [ ] **3.1. Refactorizar Creación de Entidades con `IsActive=false`**
    - **Ubicación:** `internal/repositories/category_repository.go` y `product_repository.go`.
    - **Qué hacer:** Cambiar el tipo de los campos `IsActive` en los modelos `Category` y `Product` a `*bool` (puntero a booleano). Esto permite a GORM distinguir entre un valor `false` explícito y un campo no proporcionado. Eliminar el SQL crudo y usar `db.Create()` directamente.
    - **Por qué:** Elimina el "code smell" de usar SQL crudo como solución alternativa, haciendo el código más idiomático y fácil de entender.
    - **Resultado esperado:** Código más limpio y mantenible en la capa de repositorio.

- [ ] **3.2. Estandarizar Formato de Errores de la API**
    - **Ubicación:** `api/v1/handlers/`.
    - **Qué hacer:** Crear una función de ayuda o un middleware que formatee todas las respuestas de error de la API de manera consistente (ej. `{"error": {"code": "INVALID_INPUT", "message": "..."}}`).
    - **Por qué:** Proporciona una experiencia predecible para los desarrolladores del frontend.
    - **Resultado esperado:** API más fácil de consumir.
