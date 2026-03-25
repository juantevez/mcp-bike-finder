package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

const umbralAlerta = 60.0 // score mínimo (0-100) para generar una alerta

// AlertaService evalúa resultados de búsqueda y genera alertas de posibles coincidencias.
type AlertaService struct {
	alertaRepo domain.AlertaRepository
}

func NewAlertaService(alertaRepo domain.AlertaRepository) *AlertaService {
	return &AlertaService{alertaRepo: alertaRepo}
}

// EvaluarYGenerarAlertas compara los resultados del scraper contra la bici registrada
// y persiste una alerta por cada resultado que supere el umbral de similitud.
func (s *AlertaService) EvaluarYGenerarAlertas(
	ctx context.Context,
	bici *domain.Bicicleta,
	resultados []domain.ListadoMarketplace,
) ([]*domain.Alerta, error) {

	var alertasGeneradas []*domain.Alerta

	for _, listado := range resultados {
		score := calcularScoreSimilitud(bici, listado)
		if score < umbralAlerta {
			continue
		}

		existe, err := s.alertaRepo.ExisteParaBicicletaYURL(ctx, bici.ID, listado.URL)
		if err != nil {
			log.Printf("⚠️ Error verificando duplicado para %s: %v", listado.URL, err)
			continue
		}
		if existe {
			log.Printf("⏭️ Alerta duplicada ignorada: %s", listado.URL)
			continue
		}

		alerta := &domain.Alerta{
			ID:             fmt.Sprintf("alerta_%d", time.Now().UnixNano()),
			BicicletaID:    bici.ID,
			UsuarioID:      bici.UserID,
			Titulo:         listado.Titulo,
			URL:            listado.URL,
			Marketplace:    listado.Marketplace,
			Precio:         listado.Precio,
			ScoreSimilitud: score,
			Status:         domain.AlertaStatusNueva,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := s.alertaRepo.Guardar(ctx, alerta); err != nil {
			log.Printf("⚠️ Error guardando alerta para %s: %v", listado.URL, err)
			continue
		}

		alertasGeneradas = append(alertasGeneradas, alerta)
		log.Printf("🚨 Alerta generada: score=%.0f - %s", score, listado.Titulo)
	}

	return alertasGeneradas, nil
}

// ObtenerAlertasPorUsuario devuelve las alertas de un usuario, opcionalmente filtradas por estado.
func (s *AlertaService) ObtenerAlertasPorUsuario(ctx context.Context, usuarioID string, status *string) ([]*domain.Alerta, error) {
	if usuarioID == "" {
		return nil, fmt.Errorf("usuario_id es requerido")
	}
	return s.alertaRepo.ObtenerPorUsuario(ctx, usuarioID, status)
}

// ObtenerAlertasPorBicicleta devuelve las alertas de una bicicleta específica.
func (s *AlertaService) ObtenerAlertasPorBicicleta(ctx context.Context, bicicletaID string) ([]*domain.Alerta, error) {
	if bicicletaID == "" {
		return nil, fmt.Errorf("bicicleta_id es requerido")
	}
	return s.alertaRepo.ObtenerPorBicicleta(ctx, bicicletaID)
}

// ActualizarStatus actualiza el estado de una alerta (revisada, confirmada, descartada).
func (s *AlertaService) ActualizarStatus(ctx context.Context, id, status string) error {
	estados := map[string]bool{
		domain.AlertaStatusNueva:      true,
		domain.AlertaStatusRevisada:   true,
		domain.AlertaStatusDescartada: true,
		domain.AlertaStatusConfirmada: true,
	}
	if !estados[status] {
		return fmt.Errorf("estado inválido: %s", status)
	}
	return s.alertaRepo.ActualizarStatus(ctx, id, status)
}

// ===========================================
// Scoring de similitud
// ===========================================

// calcularScoreSimilitud devuelve un score de 0-100 entre una bici registrada y un listado.
// Pesos: marca(25) + modelo(35) + color(20) + talle(10) + componente específico(10)
func calcularScoreSimilitud(bici *domain.Bicicleta, listado domain.ListadoMarketplace) float64 {
	titulo := strings.ToLower(listado.Titulo)
	score := 0.0

	if bici.Marca != "" && strings.Contains(titulo, strings.ToLower(bici.Marca)) {
		score += 25
	}
	if bici.Modelo != "" && strings.Contains(titulo, strings.ToLower(bici.Modelo)) {
		score += 35
	}
	if bici.Color != "" && strings.Contains(titulo, strings.ToLower(bici.Color)) {
		score += 20
	}
	if bici.Talle != "" && strings.Contains(titulo, strings.ToLower(bici.Talle)) {
		score += 10
	}

	// Componente modificado específico — aumenta la confianza si aparece en el título
	score += scoreComponentes(bici.Components, titulo)

	return score
}

// scoreComponentes suma hasta 10 puntos si algún componente modificado aparece en el título.
func scoreComponentes(components map[string]interface{}, titulo string) float64 {
	if len(components) == 0 {
		return 0
	}
	for _, v := range components {
		val, ok := v.(string)
		if !ok || val == "" {
			continue
		}
		// Buscar al menos una palabra significativa del componente (>4 chars)
		for _, palabra := range strings.Fields(val) {
			if len(palabra) > 4 && strings.Contains(titulo, strings.ToLower(palabra)) {
				return 10
			}
		}
	}
	return 0
}
