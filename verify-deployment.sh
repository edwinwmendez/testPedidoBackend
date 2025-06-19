#!/bin/bash
# Script de verificación pre-deployment para ExactoGas API

echo "🔍 Verificando preparación para deployment en Render.com..."
echo "=================================================="

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Contador de verificaciones
CHECKS=0
PASSED=0

# Función para check
check() {
    CHECKS=$((CHECKS + 1))
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ $1${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ $1${NC}"
    fi
}

# Función para verificar archivo
check_file() {
    if [ -f "$1" ]; then
        echo -e "${GREEN}✅ Archivo encontrado: $1${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ Archivo faltante: $1${NC}"
    fi
    CHECKS=$((CHECKS + 1))
}

# Función para verificar comando
check_command() {
    if command -v "$1" &> /dev/null; then
        echo -e "${GREEN}✅ Comando disponible: $1${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${YELLOW}⚠️  Comando no encontrado: $1${NC}"
    fi
    CHECKS=$((CHECKS + 1))
}

echo "📋 Verificando archivos necesarios..."
check_file "go.mod"
check_file "go.sum"
check_file "main.go"
check_file "Dockerfile.render"
check_file "render.yaml"
check_file "build.sh"
check_file "start.sh"
check_file ".gitignore"
check_file "README_DEPLOYMENT.md"

echo ""
echo "🛠️  Verificando herramientas..."
check_command "go"
check_command "git"

echo ""
echo "🔧 Verificando permisos de scripts..."
if [ -x "build.sh" ]; then
    echo -e "${GREEN}✅ build.sh es ejecutable${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ build.sh no es ejecutable${NC}"
    echo "   Ejecuta: chmod +x build.sh"
fi
CHECKS=$((CHECKS + 1))

if [ -x "start.sh" ]; then
    echo -e "${GREEN}✅ start.sh es ejecutable${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ start.sh no es ejecutable${NC}"
    echo "   Ejecuta: chmod +x start.sh"
fi
CHECKS=$((CHECKS + 1))

echo ""
echo "📦 Verificando dependencias de Go..."
go mod verify 2>/dev/null
check "Dependencias de Go verificadas"

echo ""
echo "🏗️  Probando build local..."
go build -o test-app . 2>/dev/null
if [ -f "test-app" ]; then
    echo -e "${GREEN}✅ Build local exitoso${NC}"
    PASSED=$((PASSED + 1))
    rm -f test-app
else
    echo -e "${RED}❌ Build local falló${NC}"
fi
CHECKS=$((CHECKS + 1))

echo ""
echo "🔒 Verificando seguridad..."

# Verificar que no hay credenciales en archivos que van a Git
echo "Buscando credenciales expuestas..."
# Buscar credenciales reales (no ejemplos/plantillas)
if grep -r "postgresql://.*@dpg-" . --exclude-dir=.git --exclude="*.md" 2>/dev/null | grep -v "user:password" | grep -v "example"; then
    echo -e "${RED}❌ PELIGRO: Credenciales reales encontradas en archivos de código${NC}"
    echo "   Revisa y remueve antes de hacer commit"
    CHECKS=$((CHECKS + 1))
else
    echo -e "${GREEN}✅ No se encontraron credenciales reales expuestas${NC}"
    PASSED=$((PASSED + 1))
    CHECKS=$((CHECKS + 1))
fi

# Verificar .gitignore incluye archivos sensibles
if grep -q "app\.env" .gitignore 2>/dev/null && grep -q "\.env" .gitignore 2>/dev/null; then
    echo -e "${GREEN}✅ .gitignore configurado correctamente${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ .gitignore no protege archivos sensibles${NC}"
    echo "   Agrega: app.env y .env al .gitignore"
fi
CHECKS=$((CHECKS + 1))

echo ""
echo "🌐 Verificando configuración de Render..."

# Verificar que DATABASE_URL esté configurada en app.env.production
if grep -q "DATABASE_URL=" app.env.production 2>/dev/null; then
    echo -e "${GREEN}✅ DATABASE_URL configurada en app.env.production${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}⚠️  DATABASE_URL no encontrada en app.env.production${NC}"
fi
CHECKS=$((CHECKS + 1))

# Verificar git
echo ""
echo "📝 Verificando Git..."
if [ -d ".git" ]; then
    echo -e "${GREEN}✅ Repositorio Git inicializado${NC}"
    PASSED=$((PASSED + 1))
    
    # Verificar si hay cambios sin commit
    if git diff-index --quiet HEAD -- 2>/dev/null; then
        echo -e "${GREEN}✅ No hay cambios sin commit${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${YELLOW}⚠️  Hay cambios sin commit${NC}"
        echo "   Ejecuta: git add . && git commit -m 'Update for deployment'"
    fi
    CHECKS=$((CHECKS + 1))
else
    echo -e "${RED}❌ Repositorio Git no inicializado${NC}"
    echo "   Ejecuta: git init"
fi
CHECKS=$((CHECKS + 1))

echo ""
echo "=================================================="
echo "📊 RESUMEN DE VERIFICACIÓN"
echo "=================================================="
echo "Total de verificaciones: $CHECKS"
echo "Verificaciones pasadas: $PASSED"
echo "Porcentaje de éxito: $(( PASSED * 100 / CHECKS ))%"

if [ $PASSED -eq $CHECKS ]; then
    echo -e "${GREEN}🎉 ¡Todo listo para deployment!${NC}"
    echo ""
    echo "📝 Próximos pasos:"
    echo "1. Subir código a GitHub"
    echo "2. Crear servicio en Render.com"
    echo "3. Configurar variables de entorno"
    echo "4. ¡Deployar!"
    exit 0
elif [ $(( PASSED * 100 / CHECKS )) -ge 80 ]; then
    echo -e "${YELLOW}⚠️  Casi listo - revisa las advertencias arriba${NC}"
    exit 0
else
    echo -e "${RED}❌ Hay problemas que necesitan atención${NC}"
    echo "   Revisa los errores marcados arriba"
    exit 1
fi
