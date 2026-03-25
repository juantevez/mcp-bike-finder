# MCP Bike Finder

Servidor MCP (Model Context Protocol) en Go para detectar bicicletas robadas. Registra una bicicleta con sus datos y modificaciones, busca automáticamente en marketplaces y genera alertas cuando encuentra posibles coincidencias.

> **Estado:** Proof of Concept

---

## Como funciona

```
Bicicleta registrada en DB (marca, modelo, modificaciones, imagen S3)
    → Scheduler cada N horas
        → Scraper en MercadoLibre / OLX
            → Scoring de similitud (marca + modelo + color + talle + componentes)
                → Si score >= 60 → Alerta nueva (con deduplicación por URL)
```

El usuario interactúa con el sistema a través de herramientas MCP desde un cliente compatible (Claude Desktop, VSCode, etc.).

---

## Arquitectura

Arquitectura hexagonal (Puertos y Adaptadores):

```
┌──────────────────────────────────────────┐
│  Cliente MCP (Claude Desktop, VSCode...) │
└──────────────────────────────────────────┘
                     │ JSON-RPC / stdio
                     ▼
┌──────────────────────────────────────────┐
│  Adaptador de entrada: internal/mcp      │
│  - server.go  (registro de tools)        │
│  - handlers.go (lógica de cada tool)     │
└──────────────────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────┐
│  Servicios de aplicación: internal/service│
│  - BicicletaService  (CRUD + validación) │
│  - ExtractorService  (DB → BicicletaInfo)│
│  - BusquedaService   (scraping)          │
│  - AlertaService     (scoring + alertas) │
│  - SchedulerService  (ejecución periódica│
└──────────────────────────────────────────┘
                     │
          ┌──────────┴──────────┐
          ▼                     ▼
┌─────────────────┐   ┌─────────────────────┐
│  Dominio        │   │  Infraestructura     │
│  internal/domain│   │  internal/infra...   │
│  - entities.go  │   │  - PostgreSQL repos  │
│  - repositories │   │  - AWS S3 client     │
│    (ports)      │   │  - Scraper (goquery) │
│  - ports.go     │   │  - Mocks en memoria  │
└─────────────────┘   └─────────────────────┘
```

### Capas

| Capa | Paquete | Responsabilidad |
|------|---------|-----------------|
| Dominio | `internal/domain` | Entidades, interfaces (ports), reglas de negocio |
| Aplicación | `internal/service` | Orquestación de casos de uso |
| Adaptadores de entrada | `internal/mcp` | Traducción MCP ↔ servicios |
| Adaptadores de salida | `internal/infrastructure` | DB, S3, scraper |

---

## Herramientas MCP

| Tool | Descripción | Inputs principales |
|------|-------------|-------------------|
| `guardar_bicicleta` | Registra una bicicleta con sus datos y modificaciones | marca, modelo, color, talle, componentes, imagen_s3_url |
| `analizar_imagen_bici` | Carga los datos de una bici registrada por su imagen S3 | imagen_s3_url |
| `buscar_bicis_similares` | Búsqueda manual en marketplaces | imagen_s3_url, presupuesto |
| `buscar_y_alertar` | Busca y genera alertas para una bici registrada | bicicleta_id |
| `listar_alertas` | Lista alertas de un usuario o bicicleta | usuario_id / bicicleta_id, status |
| `actualizar_estado_alerta` | Marca una alerta como revisada, confirmada o descartada | id, status |
| `obtener_historial_busquedas` | Historial de búsquedas de un usuario | usuario_id |

### Estados de una alerta

```
NUEVA → REVISADA → CONFIRMADA
                 → DESCARTADA
```

### Scoring de similitud

| Criterio | Puntos |
|----------|--------|
| Marca en título del listado | 25 |
| Modelo en título del listado | 35 |
| Color en título del listado | 20 |
| Talle en título del listado | 10 |
| Componente modificado en título | 10 |
| **Umbral para generar alerta** | **≥ 60** |

---

## Inicio rápido

### Prerrequisitos

- Go 1.21+
- Docker y Docker Compose
- PostgreSQL (incluido en docker-compose)
- Credenciales AWS con acceso a S3 (OCR/Vision no requeridos)

### 1. Configurar variables de entorno

```bash
cp .env.example .env
```

Variables relevantes:

```env
# Base de datos
DB_HOST=localhost
DB_PORT=5432
DB_NAME=bike_finder
DB_USER=postgres
DB_PASSWORD=tu_password

# AWS S3 (para imágenes de bicis)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=tu_key
AWS_SECRET_ACCESS_KEY=tu_secret
S3_BUCKET_NAME=tu_bucket

# Scheduler
SCHEDULER_INTERVALO_HORAS=6   # cada cuántas horas corre la búsqueda automática
SCHEDULER_LIMITE_BICIS=100    # máximo de bicis por ronda

# Scraper
SCRAPER_USER_AGENT=BikeFinderBot/1.0
```

### 2. Levantar con Docker

```bash
docker-compose up --build
```

### 3. Ejecutar localmente

```bash
go mod download
go run ./cmd/server
```

### 4. Configurar en Claude Desktop

```json
{
  "mcpServers": {
    "bike-finder": {
      "command": "go",
      "args": ["run", "./cmd/server"],
      "cwd": "/ruta/al/proyecto"
    }
  }
}
```

---

## Estructura del proyecto

```
mcp-bike-finder/
├── cmd/server/
│   └── main.go                   # Entry point, wiring de dependencias
├── internal/
│   ├── config/
│   │   └── config.go             # Configuración por env vars
│   ├── domain/
│   │   ├── entities.go           # Bicicleta, Alerta, BicicletaInfo, etc.
│   │   ├── repositories.go       # Interfaces de repositorios (ports)
│   │   └── ports.go              # Interfaces de servicios externos (ImageStorage)
│   ├── service/
│   │   ├── bicicleta.go          # CRUD y validación de bicicletas
│   │   ├── extractor.go          # Carga bici desde DB → BicicletaInfo
│   │   ├── busqueda.go           # Scraping + scoring de resultados
│   │   ├── alerta.go             # Evaluación de coincidencias + alertas
│   │   ├── scheduler.go          # Búsquedas periódicas automáticas
│   │   └── parser.go             # (interno) parsing de texto
│   ├── mcp/
│   │   ├── server.go             # Registro de tools, resources y prompts
│   │   └── handlers.go           # Implementación de cada tool MCP
│   └── infrastructure/
│       ├── database/
│       │   ├── postgres.go
│       │   ├── bicycle_repository.go
│       │   ├── reference_repository.go
│       │   ├── busqueda_repository_mock.go
│       │   └── alerta_repository_mock.go
│       ├── s3/
│       │   └── client.go         # Descarga imágenes desde S3
│       ├── vision/               # Stub para futura integración de vision
│       │   └── client.go
│       └── scraper/
│           └── client.go         # Scraping MercadoLibre y OLX con goquery
├── migrations/
│   └── 001_initial.sql
├── docker-compose.yml
└── Dockerfile
```

---

## Estado de implementación

| Componente | Estado |
|------------|--------|
| MCP server (stdio) | Funcionando |
| PostgreSQL repositories | Implementados |
| Scraper MercadoLibre / OLX | Implementado |
| Scheduler automático | Implementado |
| Alertas con deduplicación | Implementado |
| AWS S3 (descarga de imágenes) | Mock — SDK comentado, listo para descomentar |
| AWS Rekognition (vision) | Mock — pendiente definir estrategia |
| Alertas en PostgreSQL | Mock en memoria — requiere migración |
| Historial de búsquedas en DB | Mock en memoria |

---

## Licencia

MIT
