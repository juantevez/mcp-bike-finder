package service

import (
	"context"
	"log"
	"time"

	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

// SchedulerService corre búsquedas periódicas para todas las bicis activas
// y genera alertas cuando encuentra coincidencias en marketplaces.
type SchedulerService struct {
	biciRepo     domain.BicicletaRepository
	busquedaSvc  *BusquedaService
	alertaSvc    *AlertaService
	intervalo    time.Duration
	limiteBicis  int
}

func NewSchedulerService(
	biciRepo domain.BicicletaRepository,
	busquedaSvc *BusquedaService,
	alertaSvc *AlertaService,
	intervalo time.Duration,
	limiteBicis int,
) *SchedulerService {
	return &SchedulerService{
		biciRepo:    biciRepo,
		busquedaSvc: busquedaSvc,
		alertaSvc:   alertaSvc,
		intervalo:   intervalo,
		limiteBicis: limiteBicis,
	}
}

// Start inicia el scheduler en background. Debe llamarse en una goroutine.
// Se detiene cuando el contexto se cancela.
func (s *SchedulerService) Start(ctx context.Context) {
	log.Printf("⏰ Scheduler iniciado — intervalo: %s, límite: %d bicis/ronda", s.intervalo, s.limiteBicis)

	// Primera ronda inmediata al arrancar
	s.ejecutarRonda(ctx)

	ticker := time.NewTicker(s.intervalo)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.ejecutarRonda(ctx)
		case <-ctx.Done():
			log.Println("⏰ Scheduler detenido")
			return
		}
	}
}

// ejecutarRonda procesa todas las bicis activas en una sola pasada.
func (s *SchedulerService) ejecutarRonda(ctx context.Context) {
	log.Println("⏰ Iniciando ronda de búsquedas automáticas...")

	bicis, err := s.biciRepo.ListarActivas(ctx, s.limiteBicis)
	if err != nil {
		log.Printf("⚠️ Error listando bicicletas activas: %v", err)
		return
	}

	if len(bicis) == 0 {
		log.Println("⏰ Sin bicicletas activas para procesar")
		return
	}

	log.Printf("⏰ Procesando %d bicicletas...", len(bicis))

	totalAlertas := 0
	for _, bici := range bicis {
		if ctx.Err() != nil {
			log.Println("⏰ Ronda interrumpida por cancelación de contexto")
			return
		}
		alertas := s.procesarBicicleta(ctx, bici)
		totalAlertas += alertas
	}

	log.Printf("✅ Ronda completada — %d bicis procesadas, %d alertas nuevas", len(bicis), totalAlertas)
}

// procesarBicicleta corre la búsqueda y evaluación para una bicicleta.
// Devuelve la cantidad de alertas nuevas generadas.
func (s *SchedulerService) procesarBicicleta(ctx context.Context, bici *domain.Bicicleta) int {
	biciInfo := bicicletaToInfo(bici)

	resultados, err := s.busquedaSvc.BuscarBicisSimilares(ctx, biciInfo, 0)
	if err != nil {
		log.Printf("⚠️ [%s %s] Error en búsqueda: %v", bici.Marca, bici.Modelo, err)
		return 0
	}

	alertas, err := s.alertaSvc.EvaluarYGenerarAlertas(ctx, bici, resultados)
	if err != nil {
		log.Printf("⚠️ [%s %s] Error generando alertas: %v", bici.Marca, bici.Modelo, err)
		return 0
	}

	if len(alertas) > 0 {
		log.Printf("🚨 [%s %s] %d alerta(s) nueva(s)", bici.Marca, bici.Modelo, len(alertas))
	}

	return len(alertas)
}
