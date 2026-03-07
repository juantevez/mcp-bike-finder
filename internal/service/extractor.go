package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/juantevez/mcp-bike-finder/internal/domain"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/ocr"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/s3"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/vision"
	"github.com/juantevez/mcp-bike-finder/pkg/parser"
)

// ===========================================
// ExtractorService
// ===========================================

type ExtractorService struct {
	s3Client     *s3.Client
	ocrClient    *ocr.Client
	visionClient *vision.Client
	bikeParser   *parser.BikeParser
}

// NewExtractorService crea una nueva instancia del servicio extractor
func NewExtractorService(
	s3Client *s3.Client,
	ocrClient *ocr.Client,
	visionClient *vision.Client,
) *ExtractorService {
	return &ExtractorService{
		s3Client:     s3Client,
		ocrClient:    ocrClient,
		visionClient: visionClient,
		bikeParser:   parser.NewBikeParser(),
	}
}

// ExtraerInfoBicicleta extrae toda la información de una imagen de bicicleta
func (s *ExtractorService) ExtraerInfoBicicleta(ctx context.Context, s3URL string) (*domain.BicicletaInfo, error) {

	log.Printf("🔍 Extrayendo información de: %s", s3URL)

	// 1. Descargar imagen desde S3
	imgData, err := s.s3Client.Download(ctx, s3URL)
	if err != nil {
		return nil, fmt.Errorf("error descargando imagen de S3: %w", err)
	}

	if len(imgData) == 0 {
		return nil, fmt.Errorf("la imagen está vacía")
	}

	log.Printf("📥 Imagen descargada: %d bytes", len(imgData))

	// 2. Ejecutar OCR y Visión en paralelo
	var wg sync.WaitGroup
	var textoOCR string
	var visionResult *domain.VisionResult
	var ocrErr, visionErr error
	errChan := make(chan error, 2)

	wg.Add(2)

	// Goroutine 1: OCR (extraer texto)
	go func() {
		defer wg.Done()
		textoOCR, ocrErr = s.ocrClient.DetectText(ctx, imgData)
		if ocrErr != nil {
			log.Printf("⚠️ OCR falló: %v", ocrErr)
			errChan <- ocrErr
			return
		}
		log.Printf("📝 Texto OCR extraído: %d caracteres", len(textoOCR))
		errChan <- nil
	}()

	// Goroutine 2: Visión por computadora (detectar objetos, colores, etc.)
	go func() {
		defer wg.Done()
		visionResult, visionErr = s.visionClient.AnalyzeBike(ctx, imgData)
		if visionErr != nil {
			log.Printf("⚠️ Visión falló: %v", visionErr)
			errChan <- visionErr
			return
		}
		log.Printf("👁️ Análisis de visión completado")
		errChan <- nil
	}()

	wg.Wait()
	close(errChan)

	// Verificar errores (continuar si al menos uno funcionó)
	// OCR es crítico, Visión es complementaria
	if ocrErr != nil && textoOCR == "" {
		return nil, fmt.Errorf("OCR falló y es requerido: %w", ocrErr)
	}

	// 3. Parsear y estructurar la información
	biciInfo := s.bikeParser.Parsear(textoOCR, visionResult)

	// 4. Validación cruzada (OCR vs Visión)
	s.validacionCruzada(biciInfo, visionResult)

	log.Printf("✅ Extracción completada: %s %s", biciInfo.Marca, biciInfo.Modelo)

	return biciInfo, nil
}

// validacionCruzada compara resultados de OCR y Visión para mejorar precisión
func (s *ExtractorService) validacionCruzada(info *domain.BicicletaInfo, vision *domain.VisionResult) {

	if vision == nil {
		return
	}

	// Si OCR no detectó color pero Visión sí, usar Visión
	if info.Color == "" && vision.ColorDominante != "" {
		info.Color = vision.ColorDominante
		log.Printf("🎨 Color completado desde visión: %s", info.Color)
	}

	// Si OCR no detectó tipo pero Visión sí, usar Visión
	if info.Tipo == "" && vision.TipoBicicleta != "" {
		info.Tipo = vision.TipoBicicleta
		log.Printf("🚴 Tipo completado desde visión: %s", info.Tipo)
	}

	// Si hay conflicto de color, priorizar Visión (más confiable para colores)
	if info.Color != "" && vision.ColorDominante != "" {
		if !strings.EqualFold(info.Color, vision.ColorDominante) {
			log.Printf("⚠️ Conflicto de color - OCR: %s, Visión: %s. Usando Visión.",
				info.Color, vision.ColorDominante)
			info.Color = vision.ColorDominante
		}
	}
}

// ExtraerSoloTexto extrae únicamente el texto de una imagen (sin parsear)
func (s *ExtractorService) ExtraerSoloTexto(ctx context.Context, s3URL string) (string, error) {

	imgData, err := s.s3Client.Download(ctx, s3URL)
	if err != nil {
		return "", fmt.Errorf("error descargando imagen: %w", err)
	}

	return s.ocrClient.DetectText(ctx, imgData)
}

// ValidarImagen verifica que la imagen sea válida y contenga una bicicleta
func (s *ExtractorService) ValidarImagen(ctx context.Context, s3URL string) (*domain.ImageValidation, error) {

	imgData, err := s.s3Client.Download(ctx, s3URL)
	if err != nil {
		return nil, fmt.Errorf("error descargando imagen: %w", err)
	}

	return s.visionClient.ValidateImage(ctx, imgData)
}
