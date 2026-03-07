package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

// ===========================================
// BikeParser
// ===========================================

type BikeParser struct {
	marcaPatterns     []*regexp.Regexp
	modeloPatterns    []*regexp.Regexp
	tallePatterns     []*regexp.Regexp
	colorPatterns     []*regexp.Regexp
	componentPatterns map[string]*regexp.Regexp
}

// NewBikeParser crea un nuevo parser con patrones predefinidos
// pkg/parser/bike_parser.go

// NewBikeParser crea un nuevo parser con patrones predefinidos
func NewBikeParser() *BikeParser {
	return &BikeParser{
		// ✅ Patrones más flexibles y case-insensitive
		marcaPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(trek|giant|specialized|cannondale|scott|bianchi|merida|cube|orbea|ghost|santa cruz|yetis|pivot)\b`),
		},
		modeloPatterns: []*regexp.Regexp{
			// ✅ Más modelos + patrón genérico para "Palabra Número"
			regexp.MustCompile(`(?i)\b(marlin\s*\d+|escape\s*\d+|rockhopper\s*\d+|stance\s*\d+|spark\s*\d+|talent\s*\d+|contend\s*\d+)\b`),
			regexp.MustCompile(`(?i)\b([A-Z][a-z]+\s+\d+)\b`), // Patrón genérico: "Palabra 7"
		},
		tallePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(xs|s|m|l|xl|xxl)\b`),
			regexp.MustCompile(`(?i)\b(\d{2})\s*(cm|")\b`),
			regexp.MustCompile(`(?i)\b(\d{2})\s*pulg\b`),
			regexp.MustCompile(`(?i)\b(\d+\.?\d*)\s*"`), // ✅ Para "17.5""
		},
		colorPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(azul|rojo|verde|negro|blanco|gris|amarillo|naranja|plateado|dorado|metallic|blue|red|green|black|white)\b`),
		},
		componentPatterns: map[string]*regexp.Regexp{
			// ✅ Patrones más flexibles para componentes
			"asiento":     regexp.MustCompile(`(?i)(saddle|asiento|sillin)\s*[:\-]?\s*([a-zA-Z0-9\s]+)`),
			"manubrio":    regexp.MustCompile(`(?i)(handlebar|manubrio|manillar)\s*[:\-]?\s*([a-zA-Z0-9\s]+)`),
			"tija":        regexp.MustCompile(`(?i)(seatpost|tija)\s*[:\-]?\s*([a-zA-Z0-9\s]+)`),
			"suspension":  regexp.MustCompile(`(?i)(fork|suspension|horquilla)\s*[:\-]?\s*([a-zA-Z0-9\s]+)`),
			"transmision": regexp.MustCompile(`(?i)(drivetrain|transmision|cambio|shifter)\s*[:\-]?\s*([a-zA-Z0-9\s]+)`),
			"frenos":      regexp.MustCompile(`(?i)(brakes|frenos)\s*[:\-]?\s*([a-zA-Z0-9\s]+)`),
		},
	}
}

// Parsear convierte texto OCR y visión en estructura de dominio
func (p *BikeParser) Parsear(textoOCR string, vision *domain.VisionResult) *domain.BicicletaInfo {

	info := &domain.BicicletaInfo{}

	// 1. Extraer marca
	info.Marca = p.extraerMarca(textoOCR)

	// 2. Extraer modelo
	info.Modelo = p.extraerModelo(textoOCR)

	// 3. Extraer año
	info.Anio = p.extraerAnio(textoOCR)

	// 4. Extraer color
	info.Color = p.extraerColor(textoOCR)
	if info.Color == "" && vision != nil {
		info.Color = vision.ColorDominante
	}

	// 5. Extraer talle
	info.Talle = p.extraerTalle(textoOCR)

	// 6. Extraer componentes
	info.Componentes = p.extraerComponentes(textoOCR)

	// 7. Determinar tipo de bicicleta
	info.Tipo = p.determinarTipo(info, vision)

	return info
}

func (p *BikeParser) extraerMarca(texto string) string {
	// ✅ Buscar en todo el texto, no solo líneas completas
	texto = strings.ToUpper(texto)

	for _, pattern := range p.marcaPatterns {
		match := pattern.FindString(texto)
		if match != "" {
			return strings.Title(strings.ToLower(strings.TrimSpace(match)))
		}
	}

	return ""
}

func (p *BikeParser) extraerModelo(texto string) string {
	// ✅ Buscar patrones como "MARLIN 7", "Escape 3", etc.
	for _, pattern := range p.modeloPatterns {
		match := pattern.FindString(texto)
		if match != "" {
			return strings.TrimSpace(match)
		}
	}

	// ✅ Fallback: buscar después de la marca
	// Ej: "TREK MARLIN 7" → extraer "MARLIN 7"
	return ""
}

func (p *BikeParser) extraerAnio(texto string) int {
	// Buscar años entre 1980 y año actual + 1
	pattern := regexp.MustCompile(`\b(19[8-9]\d|20[0-2]\d)\b`)
	match := pattern.FindString(texto)
	if match != "" {
		var anio int
		fmt.Sscanf(match, "%d", &anio)
		return anio
	}

	return 0
}

func (p *BikeParser) extraerColor(texto string) string {
	for _, pattern := range p.colorPatterns {
		match := pattern.FindString(texto)
		if match != "" {
			return strings.Title(strings.ToLower(match))
		}
	}

	return ""
}

func (p *BikeParser) extraerTalle(texto string) string {
	// Priorizar tallas en letras
	for _, pattern := range p.tallePatterns {
		match := pattern.FindString(texto)
		if match != "" {
			return strings.ToUpper(strings.TrimSpace(match))
		}
	}

	return ""
}

func (p *BikeParser) extraerComponentes(texto string) domain.ComponentesBici {
	comps := domain.ComponentesBici{}

	if match := p.componentPatterns["asiento"].FindStringSubmatch(texto); match != nil && len(match) > 2 {
		comps.Asiento = strings.TrimSpace(match[2])
	}
	if match := p.componentPatterns["manubrio"].FindStringSubmatch(texto); match != nil && len(match) > 2 {
		comps.Manubrio = strings.TrimSpace(match[2])
	}
	if match := p.componentPatterns["tija"].FindStringSubmatch(texto); match != nil && len(match) > 2 {
		comps.Tija = strings.TrimSpace(match[2])
	}
	if match := p.componentPatterns["suspension"].FindStringSubmatch(texto); match != nil && len(match) > 2 {
		comps.Suspension = strings.TrimSpace(match[2])
	}
	if match := p.componentPatterns["transmision"].FindStringSubmatch(texto); match != nil && len(match) > 2 {
		comps.Transmision = strings.TrimSpace(match[2])
	}
	if match := p.componentPatterns["frenos"].FindStringSubmatch(texto); match != nil && len(match) > 2 {
		comps.Frenos = strings.TrimSpace(match[2])
	}

	return comps
}

func (p *BikeParser) determinarTipo(info *domain.BicicletaInfo, vision *domain.VisionResult) string {

	// Si visión detectó tipo, usarlo
	if vision != nil && vision.TipoBicicleta != "" {
		return vision.TipoBicicleta
	}

	// Determinar por modelo
	modeloLower := strings.ToLower(info.Modelo)

	if strings.Contains(modeloLower, "marlin") ||
		strings.Contains(modeloLower, "rockhopper") ||
		strings.Contains(modeloLower, "mountain") {
		return "mountain_bike"
	}

	if strings.Contains(modeloLower, "escape") ||
		strings.Contains(modeloLower, "road") ||
		strings.Contains(modeloLower, "contend") {
		return "road_bike"
	}

	if strings.Contains(modeloLower, "dual") ||
		strings.Contains(modeloLower, "hybrid") {
		return "hybrid_bike"
	}

	return "unknown"
}
