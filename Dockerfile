# ===========================================
# Build Stage
# ===========================================
FROM golang:1.24-alpine AS builder

# Instalar dependencias de build
RUN apk add --no-cache git ca-certificates tzdata

# Configurar working directory
WORKDIR /app

# Copiar go.mod y go.sum primero (cache de dependencias)
COPY go.mod go.sum ./
RUN go mod download

# Copiar el resto del código
COPY . .

# Compilar el binario (optimizado para producción)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /mcp-server ./cmd/server

# ===========================================
# Runtime Stage
# ===========================================
FROM alpine:3.20

# Instalar certificados SSL y zona horaria
RUN apk --no-cache add ca-certificates tzdata

# Crear usuario no-root por seguridad
RUN adduser -D -g '' appuser

WORKDIR /app

# Copiar binario desde builder
COPY --from=builder /mcp-server /app/mcp-server

# Cambiar propietario
RUN chown -R appuser:appuser /app

# Cambiar a usuario no-root
USER appuser

# Exponer puerto (si usas SSE/HTTP en el futuro)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/mcp-server", "--health"]

# Comando por defecto (Stdio para MCP)
ENTRYPOINT ["/app/mcp-server"]
