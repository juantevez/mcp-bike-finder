package service

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/juantevez/mcp-bike-finder/internal/domain"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/scraper"
)

// ===========================================
// BusquedaService
// ===========================================

type BusquedaService struct {
	scraperClient *scraper.Client
	config        scraper.ScraperConfig // ✅ Usar el tipo del paquete scraper
}

type ScraperConfig struct {
	UserAgent    string
	DelayMs      int
	MaxRetries   int
	Marketplaces []string
}

// NewBusquedaService crea una nueva instancia del servicio de búsqueda
func NewBusquedaService(config scraper.ScraperConfig) *BusquedaService { // ✅ Mismo tipo
	return &BusquedaService{
		scraperClient: scraper.NewClient(config), // ✅ Ahora los tipos coinciden
		config:        config,
	}
}

// BuscarBicisSimilares busca bicicletas similares en múltiples marketplaces
func (s *BusquedaService) BuscarBicisSimilares(ctx context.Context, bici *domain.BicicletaInfo, presupuestoMax float64) ([]domain.ListadoMarketplace, error) {

	log.Printf("🔍 Iniciando búsqueda para: %s %s (%s) - Presupuesto: $%.2f",
		bici.Marca, bici.Modelo, bici.Talle, presupuestoMax)

	var todosResultados []domain.ListadoMarketplace
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(s.config.Marketplaces))

	// Generar queries de búsqueda
	queries := s.generarQueries(bici)

	// Buscar en cada marketplace en paralelo
	for _, marketplace := range s.config.Marketplaces {
		wg.Add(1)
		go func(mp string) {
			defer wg.Done()

			for _, query := range queries {
				resultados, err := s.scraperClient.Buscar(ctx, mp, query, presupuestoMax)
				if err != nil {
					log.Printf("⚠️ Error en %s con query '%s': %v", mp, query, err)
					continue // Continuar con siguiente query
				}

				mu.Lock()
				todosResultados = append(todosResultados, resultados...)
				mu.Unlock()
			}
		}(marketplace)
	}

	wg.Wait()
	close(errChan)

	// Verificar si hubo resultados
	if len(todosResultados) == 0 {
		log.Println("⚠️ No se encontraron resultados en ningún marketplace")
		return []domain.ListadoMarketplace{}, nil
	}

	// Procesar resultados
	resultadosProcesados := s.procesarResultados(todosResultados, bici, presupuestoMax)

	log.Printf("✅ Búsqueda completada: %d resultados", len(resultadosProcesados))

	return resultadosProcesados, nil
}

// generarQueries genera múltiples queries de búsqueda para maximizar resultados
func (s *BusquedaService) generarQueries(bici *domain.BicicletaInfo) []string {

	var queries []string

	// Query 1: Exacto (marca + modelo + talle)
	queries = append(queries, fmt.Sprintf("%s %s talle %s", bici.Marca, bici.Modelo, bici.Talle))

	// Query 2: Marca + modelo (sin talle)
	queries = append(queries, fmt.Sprintf("%s %s", bici.Marca, bici.Modelo))

	// Query 3: Solo modelo (más amplio)
	queries = append(queries, bici.Modelo)

	// Query 4: Con componentes si están disponibles
	if bici.Componentes.Asiento != "" {
		queries = append(queries, fmt.Sprintf("%s %s %s", bici.Marca, bici.Modelo, bici.Componentes.Asiento))
	}

	// Query 5: Con tipo de bicicleta
	if bici.Tipo != "" && bici.Tipo != "unknown" {
		queries = append(queries, fmt.Sprintf("%s %s %s", bici.Tipo, bici.Marca, bici.Modelo))
	}

	// Eliminar duplicados
	queries = eliminarDuplicados(queries)

	log.Printf("📝 Queries generadas: %v", queries)

	return queries
}

// procesarResultados procesa, filtra y ordena los resultados
func (s *BusquedaService) procesarResultados(
	resultados []domain.ListadoMarketplace,
	bici *domain.BicicletaInfo,
	presupuestoMax float64,
) []domain.ListadoMarketplace {

	// 1. Filtrar por presupuesto
	var filtrados []domain.ListadoMarketplace
	for _, r := range resultados {
		if r.Precio <= presupuestoMax {
			filtrados = append(filtrados, r)
		}
	}

	// 2. Eliminar duplicados (por URL)
	filtrados = s.eliminarDuplicadosPorURL(filtrados)

	// 3. Ordenar por relevancia (precio + fecha)
	s.ordenarPorRelevancia(filtrados, bici)

	// 4. Limitar resultados (máximo 50)
	if len(filtrados) > 50 {
		filtrados = filtrados[:50]
	}

	return filtrados
}

// eliminarDuplicadosPorURL elimina listados duplicados basándose en la URL
func (s *BusquedaService) eliminarDuplicadosPorURL(resultados []domain.ListadoMarketplace) []domain.ListadoMarketplace {

	vistos := make(map[string]bool)
	var únicos []domain.ListadoMarketplace

	for _, r := range resultados {
		if !vistos[r.URL] {
			vistos[r.URL] = true
			únicos = append(únicos, r)
		}
	}

	return únicos
}

// ordenarPorRelevancia ordena los resultados por cercanía al modelo original y precio
func (s *BusquedaService) ordenarPorRelevancia(resultados []domain.ListadoMarketplace, bici *domain.BicicletaInfo) {

	sort.Slice(resultados, func(i, j int) bool {
		// Score de relevancia
		scoreI := s.calcularScore(resultados[i], bici)
		scoreJ := s.calcularScore(resultados[j], bici)

		// Mayor score = más relevante
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}

		// Si mismo score, ordenar por precio (menor primero)
		return resultados[i].Precio < resultados[j].Precio
	})
}

// calcularScore calcula un score de relevancia para un listado
func (s *BusquedaService) calcularScore(listado domain.ListadoMarketplace, bici *domain.BicicletaInfo) int {

	score := 0

	// Título coincide con marca
	if strings.Contains(strings.ToLower(listado.Titulo), strings.ToLower(bici.Marca)) {
		score += 30
	}

	// Título coincide con modelo
	if strings.Contains(strings.ToLower(listado.Titulo), strings.ToLower(bici.Modelo)) {
		score += 40
	}

	// Título coincide con talle
	if strings.Contains(strings.ToLower(listado.Titulo), strings.ToLower(bici.Talle)) {
		score += 20
	}

	// Precio cercano al promedio de mercado (estimado)
	// score += 10 // Implementar lógica de precio justo

	return score
}

// BuscarPorQuery realiza una búsqueda con un query específico
func (s *BusquedaService) BuscarPorQuery(ctx context.Context, marketplace, query string, presupuestoMax float64) ([]domain.ListadoMarketplace, error) {

	return s.scraperClient.Buscar(ctx, marketplace, query, presupuestoMax)
}

// ObtenerDetalleListado obtiene detalles completos de un listado específico
func (s *BusquedaService) ObtenerDetalleListado(ctx context.Context, url string) (*domain.ListadoMarketplace, error) {

	return s.scraperClient.ObtenerDetalle(ctx, url)
}

// ===========================================
// Funciones Auxiliares
// ===========================================

func eliminarDuplicados(slice []string) []string {
	vistos := make(map[string]bool)
	var resultado []string

	for _, item := range slice {
		if !vistos[item] {
			vistos[item] = true
			resultado = append(resultado, item)
		}
	}

	return resultado
}
