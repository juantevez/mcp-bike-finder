package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/juantevez/mcp-bike-finder/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ===========================================
// Estructuras de Input para las Tools
// ===========================================

type AnalizarImagenInput struct {
	ImagenS3URL string `json:"imagen_s3_url" jsonschema:"URL completa de la imagen en AWS S3 (ej: s3://bucket/path/image.jpg o https://s3.region.amazonaws.com/bucket/path/image.jpg)"`
}

type BuscarSimilaresInput struct {
	ImagenS3URL string  `json:"imagen_s3_url" jsonschema:"URL de la imagen de referencia en S3"`
	Presupuesto float64 `json:"presupuesto" jsonschema:"Presupuesto máximo en USD para la búsqueda"`
	Marca       string  `json:"marca,omitempty" jsonschema:"Marca específica para filtrar (opcional)"`
	Modelo      string  `json:"modelo,omitempty" jsonschema:"Modelo específico para filtrar (opcional)"`
}

type GuardarBicicletaInput struct {
	Marca       string                 `json:"marca" jsonschema:"Marca de la bicicleta (ej: Trek, Giant, Specialized)"`
	Modelo      string                 `json:"modelo" jsonschema:"Modelo de la bicicleta (ej: Marlin 7, Escape 3)"`
	Anio        int                    `json:"anio,omitempty" jsonschema:"Año de fabricación"`
	Color       string                 `json:"color" jsonschema:"Color principal"`
	Talle       string                 `json:"talle" jsonschema:"Talle (S, M, L, XL o pulgadas)"`
	Componentes domain.ComponentesBici `json:"componentes,omitempty" jsonschema:"Componentes personalizados si los hubiera"`
	ImagenS3URL string                 `json:"imagen_s3_url" jsonschema:"URL de la imagen en S3"`
	Precio      float64                `json:"precio,omitempty" jsonschema:"Precio de compra"`
}

type ObtenerHistorialInput struct {
	UsuarioID string `json:"usuario_id" jsonschema:"ID del usuario para obtener su historial"`
	Limite    int    `json:"limite,omitempty" jsonschema:"Cantidad máxima de resultados a devolver (default: 10)"`
}

type ListarMarcasInput struct {
	SoloActivas bool `json:"solo_activas,omitempty" jsonschema:"Filtrar solo marcas activas (default: true)"`
}

// ===========================================
// Handler: Analizar Imagen de Bicicleta
// ===========================================

func (s *Server) handleAnalizarImagen(ctx context.Context, req *mcp.CallToolRequest, input AnalizarImagenInput) (*mcp.CallToolResult, any, error) {

	// Validación básica
	if input.ImagenS3URL == "" {
		return nil, nil, fmt.Errorf("el campo 'imagen_s3_url' es requerido")
	}

	// Llamar al servicio de extracción
	biciInfo, err := s.extractorSvc.ExtraerInfoBicicleta(ctx, input.ImagenS3URL)
	if err != nil {
		return nil, nil, fmt.Errorf("error analizando imagen: %w", err)
	}

	// Formatear respuesta legible para el LLM
	var sb strings.Builder
	sb.WriteString("## 🚴 Información Extraída de la Imagen\n\n")
	sb.WriteString(fmt.Sprintf("**Marca:** %s\n", biciInfo.Marca))
	sb.WriteString(fmt.Sprintf("**Modelo:** %s\n", biciInfo.Modelo))

	if biciInfo.Anio > 0 {
		sb.WriteString(fmt.Sprintf("**Año:** %d\n", biciInfo.Anio))
	}

	sb.WriteString(fmt.Sprintf("**Color:** %s\n", biciInfo.Color))
	sb.WriteString(fmt.Sprintf("**Talle:** %s\n", biciInfo.Talle))
	sb.WriteString(fmt.Sprintf("**Tipo:** %s\n", biciInfo.Tipo))

	if biciInfo.Componentes.Asiento != "" {
		sb.WriteString(fmt.Sprintf("**Asiento:** %s\n", biciInfo.Componentes.Asiento))
	}
	if biciInfo.Componentes.Manubrio != "" {
		sb.WriteString(fmt.Sprintf("**Manubrio:** %s\n", biciInfo.Componentes.Manubrio))
	}
	if biciInfo.Componentes.Suspension != "" {
		sb.WriteString(fmt.Sprintf("**Suspensión:** %s\n", biciInfo.Componentes.Suspension))
	}
	if biciInfo.Componentes.Transmision != "" {
		sb.WriteString(fmt.Sprintf("**Transmisión:** %s\n", biciInfo.Componentes.Transmision))
	}

	sb.WriteString("\n---\n✅ Análisis completado exitosamente")

	// Retornar resultado con contenido de texto
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: sb.String()},
		},
		// Structured output para consumo programático
		StructuredContent: biciInfo,
	}, nil, nil
}

// ===========================================
// Handler: Buscar Bicicletas Similares
// ===========================================

func (s *Server) handleBuscarSimilares(ctx context.Context, req *mcp.CallToolRequest, input BuscarSimilaresInput) (*mcp.CallToolResult, any, error) {

	// Validación
	if input.ImagenS3URL == "" {
		return nil, nil, fmt.Errorf("el campo 'imagen_s3_url' es requerido")
	}
	if input.Presupuesto <= 0 {
		return nil, nil, fmt.Errorf("el presupuesto debe ser mayor a 0")
	}

	// Paso 1: Analizar la imagen para obtener criterios
	biciInfo, err := s.extractorSvc.ExtraerInfoBicicleta(ctx, input.ImagenS3URL)
	if err != nil {
		return nil, nil, fmt.Errorf("error analizando imagen: %w", err)
	}

	// Paso 2: Sobrescribir con filtros manuales si se proporcionaron
	if input.Marca != "" {
		biciInfo.Marca = input.Marca
	}
	if input.Modelo != "" {
		biciInfo.Modelo = input.Modelo
	}

	// Paso 3: Buscar en marketplaces
	resultados, err := s.busquedaSvc.BuscarBicisSimilares(ctx, biciInfo, input.Presupuesto)
	if err != nil {
		return nil, nil, fmt.Errorf("error buscando bicicletas: %w", err)
	}

	// Formatear respuesta
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 🔍 Búsqueda de Bicicletas Similares\n\n"))
	sb.WriteString(fmt.Sprintf("Basado en: **%s %s** (%s)\n", biciInfo.Marca, biciInfo.Modelo, biciInfo.Color))
	sb.WriteString(fmt.Sprintf("Presupuesto máximo: **$%.2f USD**\n\n", input.Presupuesto))
	sb.WriteString(fmt.Sprintf("✅ **%d resultados encontrados**\n\n", len(resultados)))

	if len(resultados) == 0 {
		sb.WriteString("⚠️ No se encontraron bicicletas que coincidan con los criterios.")
		sb.WriteString("\n\n**Sugerencias:**")
		sb.WriteString("\n- Amplía el presupuesto")
		sb.WriteString("\n- Busca por marca solamente")
		sb.WriteString("\n- Verifica otros marketplaces")
	} else {
		sb.WriteString("| # | Marketplace | Precio | Ubicación | URL |\n")
		sb.WriteString("|---|-------------|--------|-----------|-----|\n")

		for i, listado := range resultados {
			if i >= 10 { // Limitar a 10 resultados en la tabla
				break
			}
			sb.WriteString(fmt.Sprintf("| %d | %s | $%.2f | %s | [Ver](%s) |\n",
				i+1, listado.Marketplace, listado.Precio, listado.Ubicacion, listado.URL))
		}

		if len(resultados) > 10 {
			sb.WriteString(fmt.Sprintf("\n... y %d resultados más disponibles.\n", len(resultados)-10))
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: sb.String()},
		},
		StructuredContent: resultados,
	}, nil, nil
}

// ===========================================
// Handler: Guardar Bicicleta en DB
// ===========================================

func (s *Server) handleGuardarBicicleta(ctx context.Context, req *mcp.CallToolRequest, input GuardarBicicletaInput) (*mcp.CallToolResult, any, error) {

	// Validación
	if input.Marca == "" || input.Modelo == "" {
		return nil, nil, fmt.Errorf("marca y modelo son requeridos")
	}
	if input.ImagenS3URL == "" {
		return nil, nil, fmt.Errorf("imagen_s3_url es requerido")
	}

	// ✅ Helper: convertir ComponentesBici → map[string]interface{}
	componentsMap := componentesToMap(input.Componentes)

	// Crear entidad de dominio
	bicicleta := &domain.Bicicleta{
		ID:          generarID(), // Función auxiliar para UUID
		Marca:       input.Marca,
		Modelo:      input.Modelo,
		Anio:        input.Anio,
		Color:       input.Color,
		Talle:       input.Talle,
		Tipo:        determinarTipoBici(input.Modelo),
		Components:  componentsMap, // ✅ Convertido a map[string]interface{}
		ImagenS3URL: input.ImagenS3URL,
		//PrecioOriginal: input.Precio,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Guardar en base de datos
	err := s.bicicletaSvc.GuardarBicicleta(ctx, bicicleta)
	if err != nil {
		return nil, nil, fmt.Errorf("error guardando bicicleta: %w", err)
	}

	mensaje := fmt.Sprintf("✅ Bicicleta guardada exitosamente\n\n**ID:** `%s`\n**Marca:** %s\n**Modelo:** %s\n**Talle:** %s",
		bicicleta.ID, bicicleta.Marca, bicicleta.Modelo, bicicleta.Talle)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: mensaje},
		},
		StructuredContent: bicicleta,
	}, nil, nil
}

// ===========================================
// Handler: Obtener Historial de Búsquedas
// ===========================================

func (s *Server) handleObtenerHistorial(ctx context.Context, req *mcp.CallToolRequest, input ObtenerHistorialInput) (*mcp.CallToolResult, any, error) {

	// Validación
	if input.UsuarioID == "" {
		return nil, nil, fmt.Errorf("usuario_id es requerido")
	}

	limite := input.Limite
	if limite <= 0 {
		limite = 10
	}

	// Obtener historial del servicio
	historial, err := s.bicicletaSvc.ObtenerHistorialBusquedas(ctx, input.UsuarioID, limite)
	if err != nil {
		return nil, nil, fmt.Errorf("error obteniendo historial: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 📜 Historial de Búsquedas\n\n"))
	sb.WriteString(fmt.Sprintf("Usuario: `%s`\n\n", input.UsuarioID))

	if len(historial) == 0 {
		sb.WriteString("No hay búsquedas registradas para este usuario.")
	} else {
		sb.WriteString("| Fecha | Bicicleta | Resultados | Criterios |\n")
		sb.WriteString("|-------|-----------|------------|----------|\n")

		for _, h := range historial {
			sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s |\n",
				h.CreatedAt.Format("2006-01-02 15:04"),
				h.BicicletaID,
				h.Resultados,
				h.Criterios))
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: sb.String()},
		},
		StructuredContent: historial,
	}, nil, nil
}

// ===========================================
// Handlers de Resources (CORREGIDO con ejemplo oficial)
// ===========================================

// handleLeerConfiguracion
func (s *Server) handleLeerConfiguracion(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {

	configData := map[string]interface{}{
		"server_name":    s.cfg.Name,
		"server_version": s.cfg.Version,
		"transport":      "stdio",
		"timestamp":      time.Now().Format(time.RFC3339),
		"status":         "healthy",
	}

	configJSON, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializando configuración: %w", err)
	}

	// ✅ CORRECCIÓN: Struct literal directo, NO TextResource ni conversión de interfaz
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(configJSON),
			},
		},
	}, nil
}

// handleLeerEstadisticas
func (s *Server) handleLeerEstadisticas(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {

	statsData := map[string]interface{}{
		"busquedas_24h":        0,
		"imagenes_procesadas":  0,
		"marketplaces_activos": 3,
		"timestamp":            time.Now().Format(time.RFC3339),
	}

	statsJSON, err := json.MarshalIndent(statsData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializando estadísticas: %w", err)
	}

	// ✅ CORRECCIÓN: Struct literal directo
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(statsJSON),
			},
		},
	}, nil
}

// ===========================================
// Handler: Prompt - Analizar Compra
// ===========================================

func (s *Server) handlePromptAnalizarCompra(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {

	// Extraer argumentos (con valores por defecto si faltan)
	imagenURL := req.Params.Arguments["imagen_s3_url"]
	precioOferta := req.Params.Arguments["precio_oferta"]
	presupuestoMax := req.Params.Arguments["presupuesto_max"]

	promptText := fmt.Sprintf(`Eres un experto en bicicletas de segunda mano. Analiza la siguiente situación:

**Imagen de referencia:** %s
**Precio de oferta:** $%s USD
**Presupuesto máximo del usuario:** $%s USD

Tareas:
1. Analiza la imagen para extraer marca, modelo, año, talle y componentes
2. Compara el precio de oferta con el mercado actual
3. Evalúa si es una buena compra considerando:
   - Precio vs mercado
   - Estado de componentes
   - Antigüedad del modelo
4. Recomienda si comprar, negociar o buscar alternativas

Proporciona un análisis detallado con recomendaciones específicas.`,
		imagenURL, precioOferta, presupuestoMax)

	// ✅ CAMBIO CLAVE: []*mcp.PromptMessage (slice de PUNTEROS)
	// Y cada PromptMessage también debe ser puntero: &mcp.PromptMessage{...}
	return &mcp.GetPromptResult{
		Description: "Análisis de compra de bicicleta",
		Messages: []*mcp.PromptMessage{ // ← ✅ Slice de PUNTEROS
			&mcp.PromptMessage{ // ← ✅ Cada elemento es puntero
				Role:    "user",
				Content: &mcp.TextContent{Text: promptText},
			},
		},
	}, nil
}

// ===========================================
// Handler: Buscar y generar alertas
// ===========================================

type BuscarYAlertarInput struct {
	BicicletaID string  `json:"bicicleta_id" jsonschema:"ID de la bicicleta registrada a buscar"`
	Presupuesto float64 `json:"presupuesto" jsonschema:"Precio máximo de referencia para filtrar resultados"`
}

func (s *Server) handleBuscarYAlertar(ctx context.Context, req *mcp.CallToolRequest, input BuscarYAlertarInput) (*mcp.CallToolResult, any, error) {
	if input.BicicletaID == "" {
		return nil, nil, fmt.Errorf("bicicleta_id es requerido")
	}

	// 1. Cargar bicicleta desde DB
	bici, err := s.bicicletaSvc.ObtenerBicicletaPorID(ctx, input.BicicletaID)
	if err != nil {
		return nil, nil, fmt.Errorf("bicicleta no encontrada: %w", err)
	}

	// 2. Convertir a BicicletaInfo para búsqueda
	biciInfo, err := s.extractorSvc.ExtraerInfoPorID(ctx, input.BicicletaID)
	if err != nil {
		return nil, nil, fmt.Errorf("error cargando info de bicicleta: %w", err)
	}

	// 3. Buscar en marketplaces
	presupuesto := input.Presupuesto
	if presupuesto <= 0 {
		presupuesto = 999999 // sin límite de precio para búsqueda de bici robada
	}
	resultados, err := s.busquedaSvc.BuscarBicisSimilares(ctx, biciInfo, presupuesto)
	if err != nil {
		return nil, nil, fmt.Errorf("error buscando en marketplaces: %w", err)
	}

	// 4. Evaluar y generar alertas
	alertas, err := s.alertaSvc.EvaluarYGenerarAlertas(ctx, bici, resultados)
	if err != nil {
		return nil, nil, fmt.Errorf("error generando alertas: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 🔍 Búsqueda: %s %s\n\n", bici.Marca, bici.Modelo))
	sb.WriteString(fmt.Sprintf("Resultados encontrados: **%d** | Alertas generadas: **%d**\n\n", len(resultados), len(alertas)))

	if len(alertas) == 0 {
		sb.WriteString("✅ Sin coincidencias sospechosas en esta búsqueda.")
	} else {
		sb.WriteString("### 🚨 Posibles Coincidencias\n\n")
		sb.WriteString("| Score | Marketplace | Precio | Título | URL |\n")
		sb.WriteString("|-------|-------------|--------|--------|-----|\n")
		for _, a := range alertas {
			sb.WriteString(fmt.Sprintf("| %.0f%% | %s | $%.2f | %s | [Ver](%s) |\n",
				a.ScoreSimilitud, a.Marketplace, a.Precio, a.Titulo, a.URL))
		}
	}

	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: sb.String()}},
		StructuredContent: alertas,
	}, nil, nil
}

// ===========================================
// Handler: Listar alertas
// ===========================================

type ListarAlertasInput struct {
	UsuarioID   string `json:"usuario_id" jsonschema:"ID del usuario"`
	BicicletaID string `json:"bicicleta_id,omitempty" jsonschema:"Filtrar por bicicleta específica (opcional)"`
	Status      string `json:"status,omitempty" jsonschema:"Filtrar por estado: NUEVA, REVISADA, CONFIRMADA, DESCARTADA"`
}

func (s *Server) handleListarAlertas(ctx context.Context, req *mcp.CallToolRequest, input ListarAlertasInput) (*mcp.CallToolResult, any, error) {
	if input.UsuarioID == "" && input.BicicletaID == "" {
		return nil, nil, fmt.Errorf("usuario_id o bicicleta_id es requerido")
	}

	var alertas []*domain.Alerta
	var err error

	if input.BicicletaID != "" {
		alertas, err = s.alertaSvc.ObtenerAlertasPorBicicleta(ctx, input.BicicletaID)
	} else {
		var status *string
		if input.Status != "" {
			status = &input.Status
		}
		alertas, err = s.alertaSvc.ObtenerAlertasPorUsuario(ctx, input.UsuarioID, status)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("error obteniendo alertas: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 🚨 Alertas (%d)\n\n", len(alertas)))

	if len(alertas) == 0 {
		sb.WriteString("Sin alertas para los criterios indicados.")
	} else {
		sb.WriteString("| ID | Score | Estado | Marketplace | Precio | Título |\n")
		sb.WriteString("|----|-------|--------|-------------|--------|--------|\n")
		for _, a := range alertas {
			sb.WriteString(fmt.Sprintf("| `%s` | %.0f%% | %s | %s | $%.2f | [%s](%s) |\n",
				a.ID, a.ScoreSimilitud, a.Status, a.Marketplace, a.Precio, a.Titulo, a.URL))
		}
	}

	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: sb.String()}},
		StructuredContent: alertas,
	}, nil, nil
}

// ===========================================
// Handler: Actualizar estado de alerta
// ===========================================

type ActualizarEstadoAlertaInput struct {
	ID     string `json:"id" jsonschema:"ID de la alerta"`
	Status string `json:"status" jsonschema:"Nuevo estado: REVISADA, CONFIRMADA o DESCARTADA"`
}

func (s *Server) handleActualizarEstadoAlerta(ctx context.Context, req *mcp.CallToolRequest, input ActualizarEstadoAlertaInput) (*mcp.CallToolResult, any, error) {
	if input.ID == "" || input.Status == "" {
		return nil, nil, fmt.Errorf("id y status son requeridos")
	}
	if err := s.alertaSvc.ActualizarStatus(ctx, input.ID, input.Status); err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: fmt.Sprintf("✅ Alerta `%s` actualizada a **%s**", input.ID, input.Status),
		}},
	}, nil, nil
}

// ===========================================
// Funciones Auxiliares
// ===========================================

func generarID() string {
	// En producción usar github.com/google/uuid
	return fmt.Sprintf("bici_%d", time.Now().UnixNano())
}

func determinarTipoBici(modelo string) string {
	modeloLower := strings.ToLower(modelo)
	if strings.Contains(modeloLower, "marlin") || strings.Contains(modeloLower, "mountain") {
		return "mountain_bike"
	}
	if strings.Contains(modeloLower, "escape") || strings.Contains(modeloLower, "road") {
		return "road_bike"
	}
	if strings.Contains(modeloLower, "dual") || strings.Contains(modeloLower, "hybrid") {
		return "hybrid_bike"
	}
	return "unknown"
}

func (s *Server) handleListarMarcas(ctx context.Context, req *mcp.CallToolRequest, input ListarMarcasInput) (*mcp.CallToolResult, any, error) {

	soloActivas := input.SoloActivas || true

	marcas, err := s.refRepo.ObtenerTodasLasMarcas(ctx, soloActivas)
	if err != nil {
		return nil, nil, fmt.Errorf("error obteniendo marcas: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 🏷️ Marcas Disponibles (%d)\n\n", len(marcas)))

	for _, m := range marcas {
		sb.WriteString(fmt.Sprintf("- **%s**", m.Name))
		if m.Country != nil && *m.Country != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", *m.Country))
		}
		if m.Website != nil && *m.Website != "" {
			sb.WriteString(fmt.Sprintf(" - [Web](%s)", *m.Website))
		}
		sb.WriteString("\n")
	}

	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: sb.String()}},
		StructuredContent: marcas,
	}, nil, nil
}

// ===========================================
// Helper: Conversión de tipos
// ===========================================

// componentesToMap convierte ComponentesBici a map[string]interface{} para JSONB
func componentesToMap(c domain.ComponentesBici) map[string]interface{} {
	m := make(map[string]interface{})

	if c.Asiento != "" {
		m["asiento"] = c.Asiento
	}
	if c.Tija != "" {
		m["tija"] = c.Tija
	}
	if c.Manubrio != "" {
		m["manubrio"] = c.Manubrio
	}
	if c.Pedales != "" {
		m["pedales"] = c.Pedales
	}
	if c.Suspension != "" {
		m["suspension"] = c.Suspension
	}
	if c.Transmision != "" {
		m["transmision"] = c.Transmision
	}
	if c.Frenos != "" {
		m["frenos"] = c.Frenos
	}
	if c.Ruedas != "" {
		m["ruedas"] = c.Ruedas
	}

	// Si está vacío, retornar nil para que JSONB sea {} en PostgreSQL
	if len(m) == 0 {
		return nil
	}

	return m
}
