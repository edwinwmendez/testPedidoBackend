version: '3.8'

services:
  backend:
    image: exactogas-backend:dev
    container_name: exactogas_backend
    environment:
      DB_HOST: host.docker.internal
      DB_PORT: 5432
      DB_USER: exactogas_user
      DB_PASSWORD: exactogas_pass
      DB_NAME: exactogas
      DB_SSLMODE: disable
      SERVER_PORT: 8080
    ports:
      - "8080:8080"
    networks:
      - exactogas_network
    volumes:
      - ./testPedidoBackend:/app
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

networks:
  exactogas_network:
    driver: bridge
