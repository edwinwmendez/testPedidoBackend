#!/bin/bash
# Script de inicio para ExactoGas API en Render.com

set -e  # Salir si algún comando falla

echo "🚀 Iniciando ExactoGas API..."

# Mostrar información del entorno
echo "📋 Información del entorno:"
echo "   - PORT: ${PORT:-8080}"
echo "   - DATABASE_URL configurado: $([ -n "$DATABASE_URL" ] && echo "✅ Sí" || echo "❌ No")"
echo "   - JWT_SECRET configurado: $([ -n "$JWT_SECRET" ] && echo "✅ Sí" || echo "❌ No")"

# Verificar conectividad de base de datos antes de iniciar
echo "🔍 Verificando conectividad de base de datos..."
if [ -n "$DATABASE_URL" ]; then
    # Extraer host y puerto de DATABASE_URL para verificación básica
    DB_HOST=$(echo $DATABASE_URL | sed -n 's/.*@\([^:]*\).*/\1/p')
    if [ -n "$DB_HOST" ]; then
        echo "   - Host de BD detectado: $DB_HOST"
    fi
else
    echo "⚠️  DATABASE_URL no configurada - usando configuración por defecto"
fi

# Verificar que el binario existe
if [ ! -f "./app" ]; then
    echo "❌ Error: Binario './app' no encontrado"
    echo "   Asegúrate de que el build se ejecutó correctamente"
    exit 1
fi

# Hacer el binario ejecutable
chmod +x ./app

echo "✅ Verificaciones completadas"
echo "🎯 Iniciando servidor en puerto ${PORT:-8080}..."
echo "📚 Swagger estará disponible en: /swagger"
echo "🔧 Health check disponible en: /api/v1/health"

# Iniciar aplicación
# La aplicación manejará las migraciones automáticamente
exec ./app
