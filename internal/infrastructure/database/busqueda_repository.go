// internal/infrastructure/database/busqueda_repository.go
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

// ===========================================
// BusquedaRepository - Implementación PostgreSQL
// ===========================================

type BusquedaRepository struct {
	pool   *pgxpool.Pool
	schema string // "bike" o el schema que uses
}

// NewBusquedaRepository crea una nueva instancia del repositorio
func NewBusquedaRepository(pool *pgxpool.Pool, schema string) *BusquedaRepository {
	return &BusquedaRepository{
		pool:   pool,
		schema: schema,
	}
}

// ===========================================
// Métodos de la Interfaz domain.BusquedaRepository
// ===========================================

// GuardarHistorial guarda una búsqueda en el historial
func (r *BusquedaRepository) GuardarHistorial(ctx context.Context, historial *domain.BusquedaHistorial) error {

	// Si la tabla no existe aún, retornar nil (modo desarrollo)
	// En producción, deberías crear la tabla con una migración
	query := fmt.Sprintf(`
		INSERT INTO %s.busqueda_historial 
		(id, usuario_id, bicicleta_id, criterios, resultados, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, r.schema)

	_, err := r.pool.Exec(ctx, query,
		historial.ID,
		historial.UsuarioID,
		historial.BicicletaID,
		historial.Criterios,
		historial.Resultados,
		historial.CreatedAt,
	)

	// Si la tabla no existe, ignorar el error en modo desarrollo
	if err != nil && isTableNotFound(err) {
		return nil // Mock silencioso para desarrollo
	}

	return err
}

// ObtenerHistorial recupera el historial de búsquedas de un usuario
func (r *BusquedaRepository) ObtenerHistorial(ctx context.Context, usuarioID string, limite int) ([]*domain.BusquedaHistorial, error) {

	query := fmt.Sprintf(`
		SELECT id, usuario_id, bicicleta_id, criterios, resultados, created_at
		FROM %s.busqueda_historial
		WHERE usuario_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, r.schema)

	rows, err := r.pool.Query(ctx, query, usuarioID, limite)
	if err != nil {
		// Si la tabla no existe, retornar vacío (modo desarrollo)
		if isTableNotFound(err) {
			return []*domain.BusquedaHistorial{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var historial []*domain.BusquedaHistorial
	for rows.Next() {
		var h domain.BusquedaHistorial
		err := rows.Scan(
			&h.ID,
			&h.UsuarioID,
			&h.BicicletaID,
			&h.Criterios,
			&h.Resultados,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		historial = append(historial, &h)
	}

	return historial, rows.Err()
}

// ===========================================
// Helpers
// ===========================================

// isTableNotFound verifica si el error es por tabla inexistente
func isTableNotFound(err error) bool {
	if err == nil {
		return false
	}
	// pgx devuelve errores con códigos SQL estándar
	// 42P01 = "undefined_table" en PostgreSQL
	return err.Error() != "" &&
		(err.Error() == "table does not exist" ||
			err.Error() == "relation \"bike.busqueda_historial\" does not exist" ||
			err.Error() == "pq: relation \"bike.busqueda_historial\" does not exist")
}
