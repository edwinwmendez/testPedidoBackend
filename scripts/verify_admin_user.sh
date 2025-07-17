#!/bin/bash

# Colores para la salida
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # Sin color

echo -e "${YELLOW}=== Verificando Usuario Administrador ===${NC}"

# Configuración de la base de datos
DB_HOST=${EXACTOGAS_DB_HOST:-localhost}
DB_PORT=${EXACTOGAS_DB_PORT:-5432}
DB_NAME=${EXACTOGAS_DB_NAME:-pedidos_dev}
DB_USER=${EXACTOGAS_DB_USER:-postgres}
DB_PASSWORD=${EXACTOGAS_DB_PASSWORD:-postgres}

echo -e "${YELLOW}Usando la siguiente configuración:${NC}"
echo "Host: $DB_HOST"
echo "Puerto: $DB_PORT"
echo "Base de datos: $DB_NAME"
echo "Usuario: $DB_USER"
echo ""

# Verificar si el usuario admin existe
echo -e "${YELLOW}1. Verificando si el usuario admin existe...${NC}"
ADMIN_EXISTS=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM users WHERE email = 'admin@exactogas.com';")

if [ "$ADMIN_EXISTS" -eq 1 ]; then
    echo -e "${GREEN}✅ Usuario administrador encontrado${NC}"
    
    # Mostrar información del usuario
    echo -e "${YELLOW}2. Información del usuario administrador:${NC}"
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT user_id, email, full_name, phone_number, user_role, created_at FROM users WHERE email = 'admin@exactogas.com';"
    
    # Verificar el hash de la contraseña
    echo -e "${YELLOW}3. Hash de contraseña:${NC}"
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT email, password_hash FROM users WHERE email = 'admin@exactogas.com';"
    
    echo -e "${YELLOW}4. Probando autenticación via API...${NC}"
    
    # Verificar si el servidor está corriendo
    if curl -s http://localhost:8080/api/v1/auth/login > /dev/null 2>&1; then
        # Probar el login
        RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
            -H "Content-Type: application/json" \
            -d '{"email":"admin@exactogas.com","password":"admin123"}')
        
        if echo "$RESPONSE" | grep -q "access_token"; then
            echo -e "${GREEN}✅ Autenticación exitosa${NC}"
            echo -e "${GREEN}Credenciales correctas:${NC}"
            echo "  Email: admin@exactogas.com"
            echo "  Password: admin123"
        else
            echo -e "${RED}❌ Falló la autenticación${NC}"
            echo "Respuesta del servidor: $RESPONSE"
            echo -e "${YELLOW}Puede que necesites actualizar el hash de la contraseña${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  Servidor backend no está ejecutándose en localhost:8080${NC}"
        echo "Para probar la autenticación, inicia el servidor backend primero."
    fi
    
else
    echo -e "${RED}❌ Usuario administrador no encontrado${NC}"
    echo -e "${YELLOW}Ejecuta las migraciones de la base de datos:${NC}"
    echo "./scripts/run_migrations.sh"
fi

echo ""
echo -e "${GREEN}=== Verificación completada ===${NC}"