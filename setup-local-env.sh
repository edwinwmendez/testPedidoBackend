#!/bin/bash
# Script para configurar entorno local de forma segura

echo "🔧 Configurador de Entorno Local Seguro - ExactoGas API"
echo "====================================================="

# Colores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Verificar si ya existe app.env
if [ -f "app.env" ]; then
    echo -e "${YELLOW}⚠️  Ya existe app.env${NC}"
    read -p "¿Quieres sobrescribirlo? (y/N): " overwrite
    if [[ ! $overwrite =~ ^[Yy]$ ]]; then
        echo "Operación cancelada"
        exit 0
    fi
fi

echo ""
echo "📝 Configurando variables para desarrollo local..."

# Crear app.env seguro para desarrollo local
cat > app.env << 'EOF'
# Configuración LOCAL para desarrollo
# ⚠️ ESTE ARCHIVO ESTÁ EN .gitignore - NO SE SUBE A GITHUB

# Servidor
SERVER_HOST=localhost
SERVER_PORT=8080
SERVER_SHUTDOWN_TIMEOUT=5s
SERVER_READ_TIMEOUT=5s
SERVER_WRITE_TIMEOUT=10s
SERVER_IDLE_TIMEOUT=120s

# Base de datos LOCAL (PostgreSQL local o Docker)
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=exactogas
DB_SSLMODE=disable

# JWT (para desarrollo - cambiar en producción)
JWT_SECRET=dev-secret-key-change-in-production-256-bits-long
JWT_ACCESS_TOKEN_EXP=15m
JWT_REFRESH_TOKEN_EXP=7d

# Firebase (opcional para desarrollo)
FIREBASE_PROJECT_ID=exactogas-app-dev
FIREBASE_CREDENTIALS_FILE=config/firebase-credentials.json

# Configuración del negocio
BUSINESS_HOURS_START=6
BUSINESS_HOURS_END=20
TIMEZONE=America/Lima

# Para usar tu base de datos de Render en desarrollo (opcional)
# Descomenta la siguiente línea y comenta las variables DB_* de arriba
# DATABASE_URL=postgresql://user:password@host:port/database_name
EOF

echo -e "${GREEN}✅ Archivo app.env creado para desarrollo local${NC}"

# Verificar .gitignore
if ! grep -q "app.env" .gitignore 2>/dev/null; then
    echo "app.env" >> .gitignore
    echo -e "${GREEN}✅ app.env agregado a .gitignore${NC}"
fi

if ! grep -q "^\.env$" .gitignore 2>/dev/null; then
    echo ".env" >> .gitignore
    echo -e "${GREEN}✅ .env agregado a .gitignore${NC}"
fi

echo ""
echo "🎯 Configuración completada:"
echo "   📄 app.env - Variables para desarrollo local"
echo "   🚫 .gitignore - Protege archivos sensibles"
echo ""
echo "💡 Próximos pasos:"
echo "   1. Ajusta las variables en app.env según tu setup local"
echo "   2. Si quieres usar tu BD de Render en desarrollo, descomenta DATABASE_URL"
echo "   3. Ejecuta: go run main.go"
echo ""
echo -e "${YELLOW}⚠️  RECUERDA: Las credenciales de producción van SOLO en el Dashboard de Render${NC}"

# Verificar conexión de BD local (opcional)
read -p "¿Quieres probar la conexión a BD local? (y/N): " test_db
if [[ $test_db =~ ^[Yy]$ ]]; then
    echo "🔍 Probando conexión a PostgreSQL local..."
    if command -v psql &> /dev/null; then
        if psql -h localhost -p 5433 -U postgres -d postgres -c "SELECT version();" 2>/dev/null; then
            echo -e "${GREEN}✅ Conexión a PostgreSQL local exitosa${NC}"
        else
            echo -e "${RED}❌ No se pudo conectar a PostgreSQL local${NC}"
            echo "   Asegúrate de que PostgreSQL esté corriendo en puerto 5433"
            echo "   O ajusta las variables DB_* en app.env"
        fi
    else
        echo -e "${YELLOW}⚠️  psql no encontrado - no se puede probar conexión${NC}"
    fi
fi

echo ""
echo -e "${GREEN}🎉 ¡Entorno local configurado de forma segura!${NC}"
