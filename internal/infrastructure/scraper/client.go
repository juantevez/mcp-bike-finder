// internal/infrastructure/scraper/client.go
package scraper

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

// ===========================================
// Client - Cliente de scraping para marketplaces
// ===========================================

type Client struct {
	httpClient *http.Client
	config     ScraperConfig
	userAgent  string
}

type ScraperConfig struct {
	UserAgent    string
	DelayMs      int
	MaxRetries   int
	Marketplaces []string
}

// NewClient crea una nueva instancia del cliente de scraping
func NewClient(config ScraperConfig) *Client {

	// Configurar HTTP client con timeouts razonables
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// User agent por defecto si no se proporciona
	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}

	return &Client{
		httpClient: httpClient,
		config:     config,
		userAgent:  userAgent,
	}
}

// ===========================================
// Método Principal: Buscar
// ===========================================

// Buscar realiza una búsqueda en un marketplace específico
func (c *Client) Buscar(ctx context.Context, marketplace, query string, presupuestoMax float64) ([]domain.ListadoMarketplace, error) {

	log.Printf("🔍 Scraping: %s - query: '%s' - presupuesto: $%.2f", marketplace, query, presupuestoMax)

	// Normalizar query para URL
	queryURL := url.QueryEscape(query)

	// Construir URL de búsqueda según el marketplace
	searchURL, err := c.construirURLBusqueda(marketplace, queryURL)
	if err != nil {
		return nil, fmt.Errorf("error construyendo URL: %w", err)
	}

	// Ejecutar request con reintentos
	var doc *goquery.Document
	for intento := 0; intento < c.config.MaxRetries; intento++ {
		doc, err = c.fetchDocument(ctx, searchURL)
		if err == nil {
			break // Éxito
		}
		if intento < c.config.MaxRetries-1 {
			log.Printf("⚠️ Reintentando scraping (%d/%d): %v", intento+1, c.config.MaxRetries, err)
			time.Sleep(time.Duration(c.config.DelayMs) * time.Millisecond)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("error fetching página: %w", err)
	}

	// Parsear resultados según el marketplace
	resultados, err := c.parsearResultados(marketplace, doc, presupuestoMax)
	if err != nil {
		return nil, fmt.Errorf("error parseando resultados: %w", err)
	}

	log.Printf("✅ Scraping completado: %d resultados de %s", len(resultados), marketplace)

	return resultados, nil
}

// ===========================================
// Helpers por Marketplace
// ===========================================

// construirURLBusqueda genera la URL de búsqueda según el marketplace
func (c *Client) construirURLBusqueda(marketplace, queryEscaped string) (string, error) {

	switch {
	case strings.Contains(marketplace, "mercadolibre"):
		return fmt.Sprintf("https://listado.mercadolibre.com.ar/bicicletas#D[A:%s]", queryEscaped), nil

	case strings.Contains(marketplace, "olx"):
		return fmt.Sprintf("https://www.olx.com.ar/ad/q-bicicleta-%s/", strings.ToLower(queryEscaped)), nil

	case strings.Contains(marketplace, "facebook") || strings.Contains(marketplace, "marketplace"):
		// Facebook Marketplace requiere sesión, usamos búsqueda genérica
		return fmt.Sprintf("https://www.facebook.com/marketplace/search/?query=bicicleta+%s", queryEscaped), nil

	default:
		// Fallback: búsqueda genérica
		return fmt.Sprintf("https://duckduckgo.com/?q=bicicleta+%s+site:%s", queryEscaped, marketplace), nil
	}
}

// fetchDocument descarga y parsea el HTML de una URL
func (c *Client) fetchDocument(ctx context.Context, url string) (*goquery.Document, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-AR,es;q=0.9,en;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// parsearResultados extrae listados del HTML según el marketplace
func (c *Client) parsearResultados(marketplace string, doc *goquery.Document, presupuestoMax float64) ([]domain.ListadoMarketplace, error) {

	var resultados []domain.ListadoMarketplace

	switch {
	case strings.Contains(marketplace, "mercadolibre"):
		resultados = c.parsearMercadoLibre(doc, presupuestoMax)

	case strings.Contains(marketplace, "olx"):
		resultados = c.parsearOLX(doc, presupuestoMax)

	default:
		// Fallback: intentar encontrar enlaces genéricos
		resultados = c.parsearGenerico(doc, presupuestoMax, marketplace)
	}

	return resultados, nil
}

// ===========================================
// Parsers Específicos por Marketplace
// ===========================================

// parsearMercadoLibre extrae resultados de MercadoLibre Argentina
func (c *Client) parsearMercadoLibre(doc *goquery.Document, presupuestoMax float64) []domain.ListadoMarketplace {

	var resultados []domain.ListadoMarketplace

	// Selector de items en MercadoLibre (puede cambiar, verificar en producción)
	doc.Find("li.ui-search-layout__item").Each(func(i int, s *goquery.Selection) {

		// Extraer título
		titulo := s.Find("h2.ui-search-item__title").Text()
		titulo = strings.TrimSpace(titulo)
		if titulo == "" {
			return
		}

		// Extraer precio
		precioText := s.Find("span.andes-money-amount__fraction").First().Text()
		precioText = strings.ReplaceAll(precioText, ".", "")  // Remover separador de miles
		precioText = strings.ReplaceAll(precioText, ",", ".") // Convertir decimal

		var precio float64
		fmt.Sscanf(precioText, "%f", &precio)

		// Filtrar por presupuesto
		if precio > presupuestoMax && presupuestoMax > 0 {
			return
		}

		// Extraer URL
		link, exists := s.Find("a.ui-search-link").Attr("href")
		if !exists {
			return
		}

		// Extraer ubicación
		ubicacion := s.Find("span.ui-search-item__location").Text()
		ubicacion = strings.TrimSpace(ubicacion)

		// Extraer imagen (opcional)
		imagenURL, _ := s.Find("img.ui-search-result-image__element").Attr("src")

		resultado := domain.ListadoMarketplace{
			ID:          fmt.Sprintf("ml_%d_%d", time.Now().UnixNano(), i),
			Titulo:      titulo,
			Precio:      precio,
			Moneda:      "ARS",
			URL:         link,
			Marketplace: "MercadoLibre",
			Ubicacion:   ubicacion,
			ImagenURL:   imagenURL,
			Fecha:       time.Now(),
		}

		resultados = append(resultados, resultado)
	})

	return resultados
}

// parsearOLX extrae resultados de OLX Argentina
func (c *Client) parsearOLX(doc *goquery.Document, presupuestoMax float64) []domain.ListadoMarketplace {

	var resultados []domain.ListadoMarketplace

	// Selector de items en OLX (puede cambiar)
	doc.Find("article[data-cy='ad-card']").Each(func(i int, s *goquery.Selection) {

		titulo := s.Find("h6").Text()
		titulo = strings.TrimSpace(titulo)
		if titulo == "" {
			return
		}

		// OLX muestra precio en formato "$ 150.000"
		precioText := s.Find("h3").Text()
		precioText = strings.ReplaceAll(precioText, "$", "")
		precioText = strings.ReplaceAll(precioText, ".", "")
		precioText = strings.TrimSpace(precioText)

		var precio float64
		fmt.Sscanf(precioText, "%f", &precio)

		if precio > presupuestoMax && presupuestoMax > 0 {
			return
		}

		link, exists := s.Find("a").Attr("href")
		if !exists {
			return
		}

		ubicacion := s.Find("span[data-testid='ad-location']").Text()

		resultado := domain.ListadoMarketplace{
			ID:          fmt.Sprintf("olx_%d_%d", time.Now().UnixNano(), i),
			Titulo:      titulo,
			Precio:      precio,
			Moneda:      "ARS",
			URL:         link,
			Marketplace: "OLX",
			Ubicacion:   ubicacion,
			Fecha:       time.Now(),
		}

		resultados = append(resultados, resultado)
	})

	return resultados
}

// parsearGenerico: fallback para marketplaces no específicos
func (c *Client) parsearGenerico(doc *goquery.Document, presupuestoMax float64, marketplace string) []domain.ListadoMarketplace {

	var resultados []domain.ListadoMarketplace

	// Intentar encontrar enlaces que parezcan listados
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		text := strings.TrimSpace(s.Text())
		if len(text) < 10 || len(text) > 200 {
			return // Muy corto o muy largo, probablemente no es un título
		}

		// Filtrar enlaces que parezcan de listado
		if !strings.Contains(strings.ToLower(href), "bicicleta") &&
			!strings.Contains(strings.ToLower(text), "bicicleta") {
			return
		}

		resultado := domain.ListadoMarketplace{
			ID:          fmt.Sprintf("gen_%d_%d", time.Now().UnixNano(), i),
			Titulo:      text,
			Precio:      0, // No pudimos extraer precio
			Moneda:      "ARS",
			URL:         href,
			Marketplace: marketplace,
			Fecha:       time.Now(),
		}

		resultados = append(resultados, resultado)
	})

	// Limitar resultados genéricos para no saturar
	if len(resultados) > 20 {
		resultados = resultados[:20]
	}

	return resultados
}

// ===========================================
// Métodos Adicionales
// ===========================================

// ObtenerDetalle obtiene información detallada de un listado específico
func (c *Client) ObtenerDetalle(ctx context.Context, url string) (*domain.ListadoMarketplace, error) {

	doc, err := c.fetchDocument(ctx, url)
	if err != nil {
		return nil, err
	}

	// Implementación específica por marketplace (simplificada)
	titulo := doc.Find("h1").First().Text()
	precioText := doc.Find("[data-testid='price']").First().Text()

	var precio float64
	fmt.Sscanf(precioText, "%f", &precio)

	return &domain.ListadoMarketplace{
		Titulo:      strings.TrimSpace(titulo),
		Precio:      precio,
		URL:         url,
		Marketplace: "unknown",
		Fecha:       time.Now(),
	}, nil
}

// ValidateURL verifica si una URL es válida para scraping
func (c *Client) ValidateURL(marketplace, targetURL string) bool {

	// Verificar que la URL pertenezca al dominio del marketplace
	domain, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	return strings.Contains(domain.Host, marketplace)
}

// ===========================================
// Helpers
// ===========================================

// extraerPrecio intenta parsear un string de precio a float64
func extraerPrecio(texto string) float64 {
	// Remover símbolos y separadores
	texto = regexp.MustCompile(`[^\d,.\-]`).ReplaceAllString(texto, "")
	texto = strings.ReplaceAll(texto, ".", "")
	texto = strings.ReplaceAll(texto, ",", ".")

	var precio float64
	fmt.Sscanf(texto, "%f", &precio)
	return precio
}

// normalizeText limpia y normaliza texto
func normalizeText(texto string) string {
	texto = strings.TrimSpace(texto)
	texto = regexp.MustCompile(`\s+`).ReplaceAllString(texto, " ")
	return texto
}
