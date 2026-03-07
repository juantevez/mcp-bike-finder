// internal/infrastructure/database/busqueda_repository_mock.go
package database

import (
	"context"

	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

// BusquedaRepositoryMock - Implementación en memoria para desarrollo
type BusquedaRepositoryMock struct {
	historial []*domain.BusquedaHistorial
}

func NewBusquedaRepositoryMock() *BusquedaRepositoryMock {
	return &BusquedaRepositoryMock{
		historial: make([]*domain.BusquedaHistorial, 0),
	}
}

func (r *BusquedaRepositoryMock) GuardarHistorial(ctx context.Context, h *domain.BusquedaHistorial) error {
	r.historial = append(r.historial, h)
	return nil
}

func (r *BusquedaRepositoryMock) ObtenerHistorial(ctx context.Context, usuarioID string, limite int) ([]*domain.BusquedaHistorial, error) {
	var resultados []*domain.BusquedaHistorial
	for _, h := range r.historial {
		if h.UsuarioID == usuarioID {
			resultados = append(resultados, h)
		}
	}
	// Limitar resultados
	if len(resultados) > limite {
		resultados = resultados[:limite]
	}
	return resultados, nil
}
