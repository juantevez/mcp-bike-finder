# 🚴 MCP Bike Finder

Servidor MCP (Model Context Protocol) en Go para buscar bicicletas similares en marketplaces basándose en imágenes almacenadas en AWS S3.

## 📋 Descripción

Este servidor expone herramientas MCP que permiten a un LLM:
- Analizar imágenes de bicicletas almacenadas en S3
- Extraer información (marca, modelo, color, talle, componentes)
- Buscar bicicletas similares en múltiples marketplaces
- Almacenar y recuperar datos de PostgreSQL

## 🏗️ Arquitectura

Arquitectura Hexagonal (Puertos y Adaptadores):
```
┌─────────────────────────────────────────────────────────┐
│ Host MCP (LLM)                                          │
│ (Claude Desktop, VSCode, etc.)                          │
└─────────────────────────────────────────────────────────┘
                          │
                          │ MCP Protocol (JSON-RPC)
                          ▼
┌─────────────────────────────────────────────────────────┐
│ Adaptador MCP (internal/mcp)                            │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ Servicios de Aplicación (internal/service)              │
│ - ExtractorService (OCR + Vision)                       │
│ - BusquedaService (Scraping)                            │
│ - BicicletaService (Gestión de dominio)                 │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ Infraestructura (internal/infrastructure)               │
│ - PostgreSQL Repository                                 │
│ - AWS S3 Client                                         │
│ - AWS Textract (OCR)                                    │
│ - Marketplace Scraper                                   │
└─────────────────────────────────────────────────────────┘
```


## 🚀 Inicio Rápido

### Prerrequisitos

- Go 1.24+
- Docker y Docker Compose
- Cuenta de AWS con permisos S3 y Textract
- PostgreSQL (incluido en docker-compose)

### 1. Clonar Repositorio

```bash
git clone https://github.com/juantevez/mcp-bike-finder.git
cd mcp-bike-finder

2. Configurar Variables de Entorno

cp .env.example .env
# Editar .env con tus credenciales

3. Ejecutar con Docker

# Construir y levantar servicios
docker-compose up --build

# Solo para desarrollo (incluye MCP Inspector)
docker-compose --profile dev up --build

4. Probar con MCP Inspector

Abrir navegador en http://localhost:5173
Configurar conexión:
Transport: stdio
Command: docker exec -i mcp-bike-finder /app/mcp-server
Conectar y probar herramientas

5. Ejecutar Localmente (Sin Docker)

# Instalar dependencias
go mod download

# Ejecutar servidor
go run ./cmd/server

🛠️ Herramientas MCP Expuestas




| Herramienta | Descripción | Input |
|----------|-------------|--------- |
| analizar_imagen_bici | Extrae información de una imagen en S3imagen_s3_url|
| buscar_bicis_similares | Busca bicis similares en marketplaces | imagen_s3_url, presupuesto|
| guardar_bicicleta| Guarda | datos de bicicleta en PostgreSQL|
| bicicleta_json | obtener_historial |  Obtiene historial de búsquedas |


📦 Recursos MCP Expuestos

Resource
URI
Descripción
Configuración
config://app/settings
Settings de la aplicación
Estadísticas
stats://searches/daily
Estadísticas diarias de búsquedas

🧪 Tests

# Tests unitarios
go test ./internal/... -v

# Tests de integración
go test ./tests/integration/... -v

# Coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

📝 Migraciones de Base de Datos
Las migraciones se encuentran en migrations/ y se ejecutan automáticamente al iniciar el contenedor de PostgreSQL.

-- migrations/001_initial.sql
CREATE TABLE IF NOT EXISTS bicicletas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    marca VARCHAR(100) NOT NULL,
    modelo VARCHAR(100) NOT NULL,
    anio INTEGER,
    color VARCHAR(50),
    talle VARCHAR(20),
    imagen_s3_url TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);


🔐 Seguridad
Las credenciales AWS se manejan vía variables de entorno
El contenedor corre como usuario no-root
Conexiones a DB con SSL (configurable)
Rate limiting en scraping para evitar bloqueos
📄 Licencia
MIT License
🤝 Contribuir
Fork el repositorio
Crear branch de feature (git checkout -b feature/AmazingFeature)
Commit cambios (git commit -m 'Add AmazingFeature')
Push al branch (git push origin feature/AmazingFeature)
Abrir Pull Request

