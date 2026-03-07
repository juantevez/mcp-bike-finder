package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

// ===========================================
// BicicletaService
// ===========================================

type BicicletaService struct {
	biciRepo     domain.BicicletaRepository
	busquedaRepo domain.BusquedaRepository
	refRepo      domain.ReferenceRepository // ✅ AGREGAR ESTE CAMPO
	extractorSvc *ExtractorService
	busquedaSvc  *BusquedaService
}

// NewBicicletaService crea una nueva instancia del servicio
func NewBicicletaService(
	biciRepo domain.BicicletaRepository,
	busquedaRepo domain.BusquedaRepository,
	refRepo domain.ReferenceRepository, // ✅ Nuevo parámetro
	extractorSvc *ExtractorService,
	busquedaSvc *BusquedaService,
) *BicicletaService {
	return &BicicletaService{
		biciRepo:     biciRepo,
		busquedaRepo: busquedaRepo,
		refRepo:      refRepo, // ✅ Asignar el repositorio
		extractorSvc: extractorSvc,
		busquedaSvc:  busquedaSvc,
	}
}

// GuardarBicicleta guarda una bicicleta en la base de datos
func (s *BicicletaService) GuardarBicicleta(ctx context.Context, bici *domain.Bicicleta) error {

	// Validaciones de negocio
	if err := s.validarBicicleta(bici); err != nil {
		return fmt.Errorf("validación fallida: %w", err)
	}

	// Verificar si ya existe (por URL de imagen)
	existente, err := s.biciRepo.ObtenerPorImagenURL(ctx, bici.ImagenS3URL)
	if err == nil && existente != nil {
		// Si ya existe, actualizar en lugar de crear
		bici.ID = existente.ID
		bici.CreatedAt = existente.CreatedAt
		return s.biciRepo.Actualizar(ctx, bici)
	}

	// Guardar nueva bicicleta
	return s.biciRepo.Guardar(ctx, bici)
}

// ObtenerBicicletaPorID obtiene una bicicleta por su ID
func (s *BicicletaService) ObtenerBicicletaPorID(ctx context.Context, id string) (*domain.Bicicleta, error) {

	if id == "" {
		return nil, fmt.Errorf("ID no puede estar vacío")
	}

	return s.biciRepo.ObtenerPorID(ctx, id)
}

// BuscarPorMarcaModelo busca bicicletas por marca y modelo
func (s *BicicletaService) BuscarPorMarcaModelo(ctx context.Context, marca, modelo string) ([]*domain.Bicicleta, error) {

	if marca == "" {
		return nil, fmt.Errorf("marca es requerida")
	}

	return s.biciRepo.ObtenerPorMarcaModelo(ctx, marca, modelo)
}

// ProcesarImagenYGuardar analiza una imagen y guarda la bicicleta resultante
func (s *BicicletaService) ProcesarImagenYGuardar(ctx context.Context, imagenS3URL string) (*domain.Bicicleta, error) {

	// 1. Extraer información de la imagen
	biciInfo, err := s.extractorSvc.ExtraerInfoBicicleta(ctx, imagenS3URL)
	if err != nil {
		return nil, fmt.Errorf("error extrayendo información: %w", err)
	}

	// ✅ Helper: convertir ComponentesBici → map[string]interface{}
	componentsMap := componentesToMap(biciInfo.Componentes)

	// 2. Crear entidad de dominio
	bicicleta := &domain.Bicicleta{
		ID:          generarUUID(),
		Marca:       biciInfo.Marca,
		Modelo:      biciInfo.Modelo,
		Anio:        biciInfo.Anio,
		Color:       biciInfo.Color,
		Talle:       biciInfo.Talle,
		Tipo:        biciInfo.Tipo, // ✅ Ahora sí existe como alias
		Components:  componentsMap, // ✅ Convertido a map[string]interface{}
		ImagenS3URL: imagenS3URL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 3. Guardar en base de datos
	if err := s.biciRepo.Guardar(ctx, bicicleta); err != nil {
		return nil, fmt.Errorf("error guardando bicicleta: %w", err)
	}

	return bicicleta, nil
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

// ObtenerHistorialBusquedas obtiene el historial de búsquedas de un usuario
func (s *BicicletaService) ObtenerHistorialBusquedas(ctx context.Context, usuarioID string, limite int) ([]*domain.BusquedaHistorial, error) {

	if usuarioID == "" {
		return nil, fmt.Errorf("usuario_id es requerido")
	}

	if limite <= 0 {
		limite = 10
	}

	if limite > 100 {
		limite = 100 // Límite máximo de seguridad
	}

	return s.busquedaRepo.ObtenerHistorial(ctx, usuarioID, limite)
}

// RegistrarBusqueda registra una búsqueda en el historial
func (s *BicicletaService) RegistrarBusqueda(ctx context.Context, usuarioID, bicicletaID, criterios string, resultados int) error {

	historial := &domain.BusquedaHistorial{
		ID:          generarUUID(),
		UsuarioID:   usuarioID,
		BicicletaID: bicicletaID,
		Criterios:   criterios,
		Resultados:  resultados,
		CreatedAt:   time.Now(),
	}

	return s.busquedaRepo.GuardarHistorial(ctx, historial)
}

// validarBicicleta valida las reglas de negocio de una bicicleta
func (s *BicicletaService) validarBicicleta(bici *domain.Bicicleta) error {

	if bici.Marca == "" {
		return fmt.Errorf("la marca es requerida")
	}

	if bici.Modelo == "" {
		return fmt.Errorf("el modelo es requerido")
	}

	if bici.ImagenS3URL == "" {
		return fmt.Errorf("la URL de la imagen es requerida")
	}

	// Validar formato de URL de S3
	if !esURLS3Valida(bici.ImagenS3URL) {
		return fmt.Errorf("URL de S3 inválida")
	}

	// Validar talle (si se proporciona)
	if bici.Talle != "" && !esTalleValido(bici.Talle) {
		return fmt.Errorf("talle inválido: %s", bici.Talle)
	}

	// Validar año (si se proporciona)
	añoActual := time.Now().Year()
	if bici.Anio > 0 {
		if bici.Anio < 1980 || bici.Anio > añoActual+1 {
			return fmt.Errorf("año inválido: %d", bici.Anio)
		}
	}

	return nil
}

// ===========================================
// Funciones Auxiliares
// ===========================================

func esURLS3Valida(url string) bool {
	// Validaciones básicas de URL S3
	if len(url) < 10 {
		return false
	}
	// Acepta formatos: s3://bucket/key o https://s3.region.amazonaws.com/bucket/key
	return true // Implementar validación más estricta según necesidad
}

func esTalleValido(talle string) bool {
	tallesValidos := []string{"XS", "S", "M", "L", "XL", "XXL",
		"13", "15", "17", "19", "21", "23", // pulgadas
		"48", "50", "52", "54", "56", "58", "60"} // cm

	for _, valido := range tallesValidos {
		if talle == valido {
			return true
		}
	}
	return false
}

func generarUUID() string {
	// En producción usar: github.com/google/uuid
	// return uuid.New().String()
	return fmt.Sprintf("uuid_%d", time.Now().UnixNano())
}

// internal/service/bicicleta.go - Agregar al final del archivo

// BuscarCoincidenciasEnCatalogo busca modelos similares en el catálogo maestro
func (s *BicicletaService) BuscarCoincidenciasEnCatalogo(
	ctx context.Context,
	marcaOCR, modeloOCR string,
	anioOCR *int, // ← *int, no *string
) ([]*domain.BikeCatalog, error) {

	// Normalizar inputs del OCR (convertir a pointers)
	var marca, modelo *string
	if marcaOCR != "" {
		m := s.normalizarTexto(marcaOCR)
		marca = &m
	}
	if modeloOCR != "" {
		mo := s.normalizarTexto(modeloOCR)
		modelo = &mo
	}

	return s.refRepo.BuscarEnCatalogo(ctx, marca, modelo, anioOCR, 10)
}

// Helper para normalizar texto de OCR
func (s *BicicletaService) normalizarTexto(texto string) string {
	texto = strings.TrimSpace(texto)
	texto = strings.ToLower(texto)
	texto = regexp.MustCompile(`[^a-z0-9\s]`).ReplaceAllString(texto, "")
	return strings.Join(strings.Fields(texto), " ")
}
