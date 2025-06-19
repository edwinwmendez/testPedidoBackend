#!/bin/bash

# Colores para la salida
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # Sin color

echo -e "${YELLOW}=== Iniciando ExactoGas API en modo desarrollo ===${NC}"

# Verificar si Air está instalado
if ! command -v air &> /dev/null
then
    echo -e "${YELLOW}Air no está instalado. Instalando...${NC}"
    go install github.com/cosmtrek/air@latest
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}Error al instalar Air. Por favor, instálalo manualmente:${NC}"
        echo "go install github.com/cosmtrek/air@latest"
        exit 1
    fi
    
    echo -e "${GREEN}Air instalado correctamente.${NC}"
fi

# Crear configuración de Air si no existe
if [ ! -f ".air.toml" ]; then
    echo -e "${YELLOW}Generando configuración para Air...${NC}"
    cat > .air.toml << EOF
root = "."
tmp_dir = "tmp"
[build]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ."
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = true
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_error = true

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false
EOF
    echo -e "${GREEN}Configuración de Air generada correctamente.${NC}"
fi

# Establecer variables de entorno para desarrollo
export EXACTOGAS_ENV="development"

# Iniciar la aplicación con Air
echo -e "${GREEN}Iniciando servidor con recarga automática...${NC}"
air

exit 0 