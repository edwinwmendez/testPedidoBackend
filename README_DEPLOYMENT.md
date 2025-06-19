# 🚀 Guía de Deployment de ExactoGas API en Render.com

Esta guía te ayudará a deployar tu backend Go en Render.com para que puedas desarrollar y hacer pruebas con amigos desde la web.

## 📋 Preparación Previa

### ✅ Lo que ya tienes listo:
- ✅ Base de datos PostgreSQL en Render.com
- ✅ DATABASE_URL externa configurada
- ✅ Código backend optimizado para deployment
- ✅ Archivos de configuración para Render

## 🛠️ Pasos para el Deployment

### 1. **Crear Repositorio en GitHub**

```bash
# Desde tu directorio backend
cd /Users/edwinwm/Desktop/appGas/backend

# Inicializar git (si no está inicializado)
git init

# Agregar todos los archivos
git add .

# Commit inicial
git commit -m "feat: Preparar backend para deployment en Render.com"

# Crear repositorio en GitHub y conectar
git remote add origin https://github.com/TU_USUARIO/exactogas-backend.git
git branch -M main
git push -u origin main
```

### 2. **Deployment en Render.com**

#### Opción A: Usando Infrastructure as Code (Recomendado)
1. Ve a [render.com](https://render.com) y haz login
2. Haz clic en "New" → "Blueprint"
3. Conecta tu repositorio GitHub
4. Render detectará automáticamente el archivo `render.yaml`
5. Revisa la configuración y haz clic en "Apply"

#### Opción B: Deployment Manual
1. Ve a [render.com](https://render.com) y haz login
2. Haz clic en "New" → "Web Service"
3. Conecta tu repositorio GitHub
4. Configura:
   - **Name**: `exactogas-api`
   - **Runtime**: `Docker`
   - **Dockerfile Path**: `./Dockerfile.render`
   - **Branch**: `main`

### 3. **Variables de Entorno en Render (CONFIGURACIÓN SEGURA)**

⚠️ **IMPORTANTE DE SEGURIDAD**: NUNCA subas credenciales a GitHub. Configura todas las variables sensibles **SOLO** en el Dashboard de Render.

En el dashboard de tu servicio, ve a la pestaña "Environment" y agrega estas variables una por una:

#### **Variables OBLIGATORIAS (configurar manualmente):**
```env
# Base de datos (CRÍTICO - NUNCA en GitHub)
DATABASE_URL
postgresql://edwin:6nXW48kpjNnnAVBXzf7m5DkQIkZtdCxe@dpg-d1a4gvaeli5vc71apeh0g-a.oregon-postgres.render.com/tienda_mendez_db

# JWT Secret (CRÍTICO - usar "Generate Value")
JWT_SECRET
[Hacer clic en "Generate Value" para crear automáticamente]
```

#### **Variables de configuración (seguras para el blueprint):**

#### **Variables de configuración (seguras para el blueprint):**
```env
# Configuración JWT (seguras - ya en render.yaml)
JWT_ACCESS_TOKEN_EXP=15m
JWT_REFRESH_TOKEN_EXP=7d

# Configuración del servidor (seguras - ya en render.yaml)
SERVER_HOST=0.0.0.0
SERVER_READ_TIMEOUT=5s
SERVER_WRITE_TIMEOUT=10s
SERVER_IDLE_TIMEOUT=120s

# Configuración del negocio (seguras - ya en render.yaml)
BUSINESS_HOURS_START=6
BUSINESS_HOURS_END=20
TIMEZONE=America/Lima

# Identificador de entorno (seguro - ya en render.yaml)
RENDER=true

# Firebase opcional (no sensible - ya en render.yaml)
FIREBASE_PROJECT_ID=exactogas-app
```

### 🔒 **PASOS DE CONFIGURACIÓN SEGURA:**

1. **Deployment inicial**:
   - Render creará el servicio con las variables no-sensibles del `render.yaml`
   - Las variables sensibles (DATABASE_URL, JWT_SECRET) estarán faltando

2. **Configurar variables sensibles**:
   - Ve al Dashboard → Tu servicio → "Environment"
   - Agrega manualmente `DATABASE_URL` con tu credencial completa
   - Para `JWT_SECRET`: haz clic en "Generate Value" 
   - Guarda y redeploya automáticamente

## 🔗 URLs de tu API

Una vez deployado, tu API estará disponible en:
- **API Base**: `https://exactogas-api.onrender.com`
- **Health Check**: `https://exactogas-api.onrender.com/api/v1/health`
- **Swagger Docs**: `https://exactogas-api.onrender.com/swagger`

## 🧪 Testing con Amigos

### Endpoints Principales:
```
POST /api/v1/auth/register    # Registro de usuarios
POST /api/v1/auth/login       # Login
GET  /api/v1/products         # Lista productos
POST /api/v1/orders           # Crear pedido
GET  /api/v1/orders           # Ver pedidos
```

### Ejemplo de Prueba:
```bash
# Health check
curl https://exactogas-api.onrender.com/api/v1/health

# Ver productos disponibles
curl https://exactogas-api.onrender.com/api/v1/products
```

## 🔄 Auto-Deployment

Con la configuración actual:
- ✅ Cada push a `main` dispara deployment automático
- ✅ Migraciones se ejecutan automáticamente
- ✅ Health checks configurados
- ✅ SSL/HTTPS habilitado por defecto

## 🐛 Troubleshooting

### Si el deployment falla:
1. **Revisa los logs** en el dashboard de Render
2. **Verifica variables de entorno** están configuradas
3. **Revisa conectividad de BD** con DATABASE_URL

### Comandos útiles para debugging:
```bash
# Ver logs del build
render logs --service=exactogas-api --type=build

# Ver logs runtime
render logs --service=exactogas-api --type=deploy
```

## 📱 Desarrollo Local

Para desarrollo local, sigue usando:
```bash
# Instalar dependencias
go mod download

# Generar docs Swagger
swag init

# Ejecutar en modo desarrollo
go run main.go
```

## 🔒 Seguridad

- ✅ Variables sensibles en variables de entorno
- ✅ SSL/TLS habilitado automáticamente
- ✅ Usuario no-root en contenedor
- ✅ Health checks configurados

## 💡 Próximos Pasos

1. **Configurar dominio custom** (opcional)
2. **Monitoreo y alertas**
3. **Backup automático de BD**
4. **CI/CD avanzado**

---

**¡Tu API ya está lista para testing en producción! 🎉**

Para soporte: [Documentación de Render](https://render.com/docs)
