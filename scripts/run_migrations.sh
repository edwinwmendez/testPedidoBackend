#!/bin/bash

# Colores para la salida
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # Sin color

echo -e "${YELLOW}=== Ejecutando migraciones SQL para ExactoGas ===${NC}"

# Verificar que psql esté instalado
if ! command -v psql &> /dev/null
then
    echo -e "${RED}Error: psql no está instalado. Por favor instala PostgreSQL CLI.${NC}"
    exit 1
fi

# Configuración de la base de datos
# Por defecto usa valores locales, pero se pueden sobrescribir con variables de entorno
DB_HOST=${EXACTOGAS_DB_HOST:-localhost}
DB_PORT=${EXACTOGAS_DB_PORT:-5433}
DB_NAME=${EXACTOGAS_DB_NAME:-exactogas}
DB_USER=${EXACTOGAS_DB_USER:-postgres}
DB_PASSWORD=${EXACTOGAS_DB_PASSWORD:-postgres}

# Mostrar configuración
echo -e "${YELLOW}Usando la siguiente configuración:${NC}"
echo "Host: $DB_HOST"
echo "Puerto: $DB_PORT"
echo "Base de datos: $DB_NAME"
echo "Usuario: $DB_USER"
echo "Contraseña: ********"

# Crear la base de datos si no existe
echo -e "${YELLOW}Verificando si existe la base de datos...${NC}"
if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -lqt | cut -d \| -f 1 | grep -qw "$DB_NAME"; then
    echo -e "${GREEN}La base de datos '$DB_NAME' ya existe.${NC}"
else
    echo -e "${YELLOW}Creando base de datos '$DB_NAME'...${NC}"
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "CREATE DATABASE $DB_NAME;"
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Base de datos creada exitosamente.${NC}"
    else
        echo -e "${RED}Error al crear la base de datos.${NC}"
        exit 1
    fi
fi

# Directorio de migraciones
MIGRATIONS_DIR="database/migrations"

# Verificar que exista el directorio de migraciones
if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo -e "${RED}Error: Directorio de migraciones no encontrado: $MIGRATIONS_DIR${NC}"
    exit 1
fi

# Obtener lista de archivos SQL en el directorio
SQL_FILES=$(find "$MIGRATIONS_DIR" -name "*.sql" | sort)

if [ -z "$SQL_FILES" ]; then
    echo -e "${RED}Error: No se encontraron archivos SQL en $MIGRATIONS_DIR${NC}"
    exit 1
fi

# Ejecutar cada archivo SQL
for file in $SQL_FILES
do
    echo -e "${YELLOW}Ejecutando migración: $(basename $file)${NC}"
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$file"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Migración completada exitosamente.${NC}"
    else
        echo -e "${RED}Error al ejecutar la migración.${NC}"
        exit 1
    fi
done

echo -e "${GREEN}=== ¡Todas las migraciones fueron ejecutadas exitosamente! ===${NC}"
exit 0 