package service

import (
	"context"
	"fmt"

	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

// ExtractorService convierte una bicicleta registrada en la DB en un BicicletaInfo
// listo para ser usado como criterio de búsqueda en marketplaces.
type ExtractorService struct {
	biciRepo domain.BicicletaRepository
}

func NewExtractorService(biciRepo domain.BicicletaRepository) *ExtractorService {
	return &ExtractorService{biciRepo: biciRepo}
}

// ExtraerInfoBicicleta carga los datos de una bicicleta desde la DB por su URL de imagen
// y los convierte en BicicletaInfo para usarlos como criterio de búsqueda.
func (s *ExtractorService) ExtraerInfoBicicleta(ctx context.Context, imagenS3URL string) (*domain.BicicletaInfo, error) {
	bici, err := s.biciRepo.ObtenerPorImagenURL(ctx, imagenS3URL)
	if err != nil {
		return nil, fmt.Errorf("error buscando bicicleta: %w", err)
	}
	if bici == nil {
		return nil, fmt.Errorf("no hay bicicleta registrada con imagen: %s", imagenS3URL)
	}
	return bicicletaToInfo(bici), nil
}

// ExtraerInfoPorID carga los datos de una bicicleta desde la DB por su ID.
func (s *ExtractorService) ExtraerInfoPorID(ctx context.Context, id string) (*domain.BicicletaInfo, error) {
	bici, err := s.biciRepo.ObtenerPorID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("bicicleta no encontrada: %w", err)
	}
	return bicicletaToInfo(bici), nil
}

// bicicletaToInfo convierte una entidad Bicicleta en un BicicletaInfo para búsqueda.
func bicicletaToInfo(b *domain.Bicicleta) *domain.BicicletaInfo {
	info := &domain.BicicletaInfo{
		Marca:  b.Marca,
		Modelo: b.Modelo,
		Anio:   b.Anio,
		Color:  b.Color,
		Talle:  b.Talle,
		Tipo:   b.Tipo,
	}

	// Mapear componentes desde el JSONB si existen
	if b.Components != nil {
		comps := domain.ComponentesBici{}
		if v, ok := b.Components["asiento"].(string); ok {
			comps.Asiento = v
		}
		if v, ok := b.Components["tija"].(string); ok {
			comps.Tija = v
		}
		if v, ok := b.Components["manubrio"].(string); ok {
			comps.Manubrio = v
		}
		if v, ok := b.Components["suspension"].(string); ok {
			comps.Suspension = v
		}
		if v, ok := b.Components["transmision"].(string); ok {
			comps.Transmision = v
		}
		if v, ok := b.Components["frenos"].(string); ok {
			comps.Frenos = v
		}
		if v, ok := b.Components["ruedas"].(string); ok {
			comps.Ruedas = v
		}
		info.Componentes = comps
	}

	return info
}
