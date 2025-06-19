#!/bin/bash
# Script de build optimizado para Render.com

set -e  # Salir si algún comando falla

echo "🔧 Iniciando build para ExactoGas API..."

# Verificar versión de Go
echo "📋 Verificando entorno..."
go version

# Limpiar cache si es necesario
echo "🧹 Limpiando cache de Go..."
go clean -cache

# Descargar dependencias
echo "📦 Descargando dependencias..."
go mod download
go mod verify

# Generar documentación Swagger si swag está disponible
if command -v swag &> /dev/null; then
    echo "📚 Generando documentación Swagger..."
    swag init -g main.go --output ./docs
else
    echo "⚠️  swag no encontrado, omitiendo generación de docs..."
fi

# Compilar aplicación con optimizaciones para Render
echo "🏗️  Compilando aplicación..."
CGO_ENABLED=0 GOOS=linux go build \
    -tags netgo \
    -ldflags '-s -w -X main.version=$(date +%Y%m%d-%H%M%S)' \
    -o app \
    .

# Verificar que el binario se creó correctamente
if [ -f "./app" ]; then
    echo "✅ Build completado exitosamente!"
    echo "📊 Tamaño del binario: $(ls -lh app | awk '{print $5}')"
else
    echo "❌ Error: No se pudo crear el binario"
    exit 1
fi

echo "🚀 Listo para deployment!"
