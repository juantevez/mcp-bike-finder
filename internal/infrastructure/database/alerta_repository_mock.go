package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

// AlertaRepositoryMock implementa domain.AlertaRepository en memoria para el POC.
type AlertaRepositoryMock struct {
	mu       sync.RWMutex
	alertas  map[string]*domain.Alerta
	urlIndex map[string]bool // key: bicicletaID+":"+url
}

func NewAlertaRepositoryMock() *AlertaRepositoryMock {
	return &AlertaRepositoryMock{
		alertas:  make(map[string]*domain.Alerta),
		urlIndex: make(map[string]bool),
	}
}

func (r *AlertaRepositoryMock) Guardar(_ context.Context, alerta *domain.Alerta) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copia := *alerta
	r.alertas[alerta.ID] = &copia
	r.urlIndex[alerta.BicicletaID+":"+alerta.URL] = true
	return nil
}

func (r *AlertaRepositoryMock) ExisteParaBicicletaYURL(_ context.Context, bicicletaID, url string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.urlIndex[bicicletaID+":"+url], nil
}

func (r *AlertaRepositoryMock) ObtenerPorBicicleta(_ context.Context, bicicletaID string) ([]*domain.Alerta, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var resultado []*domain.Alerta
	for _, a := range r.alertas {
		if a.BicicletaID == bicicletaID {
			copia := *a
			resultado = append(resultado, &copia)
		}
	}
	return resultado, nil
}

func (r *AlertaRepositoryMock) ObtenerPorUsuario(_ context.Context, usuarioID string, status *string) ([]*domain.Alerta, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var resultado []*domain.Alerta
	for _, a := range r.alertas {
		if a.UsuarioID != usuarioID {
			continue
		}
		if status != nil && a.Status != *status {
			continue
		}
		copia := *a
		resultado = append(resultado, &copia)
	}
	return resultado, nil
}

func (r *AlertaRepositoryMock) ActualizarStatus(_ context.Context, id string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	alerta, ok := r.alertas[id]
	if !ok {
		return fmt.Errorf("alerta no encontrada: %s", id)
	}
	alerta.Status = status
	alerta.UpdatedAt = time.Now()
	return nil
}
