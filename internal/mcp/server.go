package mcp

import (
	"context"
	"log"

	"github.com/juantevez/mcp-bike-finder/internal/config"
	"github.com/juantevez/mcp-bike-finder/internal/domain"
	"github.com/juantevez/mcp-bike-finder/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ===========================================
// Server MCP
// ===========================================

// internal/mcp/server.go

type Server struct {
	mcpServer    *mcp.Server
	cfg          config.MCPConfig
	bicicletaSvc *service.BicicletaService
	extractorSvc *service.ExtractorService
	busquedaSvc  *service.BusquedaService
	alertaSvc    *service.AlertaService
	refRepo      domain.ReferenceRepository
}

func NewServer(
	cfg config.MCPConfig,
	bicicletaSvc *service.BicicletaService,
	extractorSvc *service.ExtractorService,
	busquedaSvc *service.BusquedaService,
	alertaSvc *service.AlertaService,
	refRepo domain.ReferenceRepository,
) *Server {
	return &Server{
		cfg:          cfg,
		bicicletaSvc: bicicletaSvc,
		extractorSvc: extractorSvc,
		busquedaSvc:  busquedaSvc,
		alertaSvc:    alertaSvc,
		refRepo:      refRepo,
	}
}

// Run inicia el servidor MCP
func (s *Server) Run(ctx context.Context) error {
	// 1. Crear el servidor MCP
	s.mcpServer = mcp.NewServer(&mcp.Implementation{
		Name:    s.cfg.Name,
		Version: s.cfg.Version,
	}, nil)

	// 2. Registrar todas las herramientas (Tools)
	s.registrarTools()

	// 3. Registrar todos los recursos (Resources)
	s.registrarResources()

	// 4. Registrar prompts (opcional)
	s.registrarPrompts()

	// 5. Iniciar el transporte (Stdio por defecto)
	log.Println("📡 Esperando conexiones MCP...")
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

// registrarTools registra todas las herramientas disponibles
func (s *Server) registrarTools() {
	log.Println("🔧 Registrando herramientas MCP...")

	// Tool 1: Analizar imagen de bicicleta
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "analizar_imagen_bici",
		Description: "Analiza una imagen de bicicleta almacenada en S3 y extrae información (marca, modelo, color, talle, componentes)",
	}, s.handleAnalizarImagen)

	// Tool 2: Buscar bicicletas similares
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "buscar_bicis_similares",
		Description: "Busca bicicletas similares en marketplaces basándose en una imagen de referencia y presupuesto",
	}, s.handleBuscarSimilares)

	// Tool 3: Guardar bicicleta en base de datos
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "guardar_bicicleta",
		Description: "Guarda los datos de una bicicleta en la base de datos PostgreSQL",
	}, s.handleGuardarBicicleta)

	// Tool 4: Obtener historial de búsquedas
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "obtener_historial_busquedas",
		Description: "Obtiene el historial de búsquedas de un usuario",
	}, s.handleObtenerHistorial)

	// Tool 5: Buscar y generar alertas de bicicleta robada
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "buscar_y_alertar",
		Description: "Busca en marketplaces y genera alertas cuando encuentra posibles coincidencias con una bicicleta registrada",
	}, s.handleBuscarYAlertar)

	// Tool 6: Listar alertas de un usuario
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listar_alertas",
		Description: "Lista las alertas de posibles coincidencias para un usuario, filtrable por estado",
	}, s.handleListarAlertas)

	// Tool 7: Actualizar estado de una alerta
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "actualizar_estado_alerta",
		Description: "Actualiza el estado de una alerta (REVISADA, CONFIRMADA, DESCARTADA)",
	}, s.handleActualizarEstadoAlerta)

	log.Println("✅ 7 herramientas registradas")
}

// internal/mcp/server.go

// registrarResources registra todos los recursos disponibles
func (s *Server) registrarResources() {
	log.Println("📚 Registrando recursos MCP...")

	// ✅ Resource 1: Configuración de la aplicación
	// Usar método del servidor: s.mcpServer.AddResource (NO mcp.AddResource)
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "config://app/settings",
		Name:        "Configuración de la Aplicación",
		Description: "Settings y configuración actual del servidor MCP Bike Finder",
		//MimeType:    "application/json",
	}, s.handleLeerConfiguracion)

	// ✅ Resource 2: Estadísticas de búsquedas
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "stats://searches/daily",
		Name:        "Estadísticas Diarias de Búsquedas",
		Description: "Número de búsquedas realizadas en las últimas 24 horas",
		//MimeType:    "application/json",
	}, s.handleLeerEstadisticas)

	log.Println("✅ 2 recursos registrados")
}

// registrarPrompts registra plantillas de prompts
func (s *Server) registrarPrompts() {
	log.Println("💬 Registrando prompts MCP...")

	// ✅ Usar método del servidor: s.mcpServer.AddPrompt (NO mcp.AddPrompt)
	// ✅ Arguments debe ser []*mcp.PromptArgument (slice de PUNTEROS), no []mcp.PromptArgument
	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "analizar_compra_bici",
		Description: "Guía al usuario para analizar si una bicicleta es una buena compra comparando con el mercado",
		Arguments: []*mcp.PromptArgument{ // ← ✅ Slice de PUNTEROS
			{Name: "imagen_s3_url", Description: "URL de la imagen en S3"},
			{Name: "precio_oferta", Description: "Precio de la bicicleta en venta"},
			{Name: "presupuesto_max", Description: "Presupuesto máximo del usuario"},
		},
	}, s.handlePromptAnalizarCompra)

	log.Println("✅ 1 prompt registrado")
}
