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

# Exponer puerto
EXPOSE 8080

# Ejecutar la aplicación
CMD ["./main"] 