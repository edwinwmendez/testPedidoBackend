FROM golang:1.22-alpine AS builder

# Establecer directorio de trabajo
WORKDIR /app

# Instalar dependencias
RUN apk add --no-cache git

# Copiar archivos de dependencias
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar el resto del código fuente
COPY . .

# Compilar la aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Imagen final
FROM alpine:latest

# Instalar certificados SSL
RUN apk --no-cache add ca-certificates

# Establecer directorio de trabajo
WORKDIR /app

# Copiar el binario compilado
COPY --from=builder /app/main .

# Copiar archivos de configuración
COPY app.env.example app.env

# Copiar migraciones y scripts
COPY database/ database/
COPY scripts/ scripts/

# Instalar PostgreSQL client para migraciones
RUN apk add --no-cache postgresql-client

# Exponer puerto
EXPOSE 8080

# Crear script de inicio que ejecute migraciones y luego la app
RUN echo '#!/bin/sh' > /app/start.sh && \
    echo 'echo "Ejecutando migraciones..."' >> /app/start.sh && \
    echo 'if [ -d "database/migrations" ]; then' >> /app/start.sh && \
    echo '  for file in database/migrations/*.sql; do' >> /app/start.sh && \
    echo '    echo "Ejecutando migración: $file"' >> /app/start.sh && \
    echo '    psql $DATABASE_URL -f "$file" 2>/dev/null || echo "Migración ya aplicada: $file"' >> /app/start.sh && \
    echo '  done' >> /app/start.sh && \
    echo 'fi' >> /app/start.sh && \
    echo 'echo "Iniciando aplicación..."' >> /app/start.sh && \
    echo 'exec ./main' >> /app/start.sh && \
    chmod +x /app/start.sh

# Ejecutar script de inicio
CMD ["./start.sh"] 