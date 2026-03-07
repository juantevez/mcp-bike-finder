package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

// ===========================================
// ReferenceRepository - Tablas de lookup
// ===========================================

type ReferenceRepository struct {
	pool       *pgxpool.Pool
	bikeSchema string
}

func NewReferenceRepository(pool *pgxpool.Pool, bikeSchema string) *ReferenceRepository {
	return &ReferenceRepository{
		pool:       pool,
		bikeSchema: bikeSchema,
	}
}

// ===========================================
// Brands
// ===========================================

// ObtenerTodasLasMarcas - Para autocompletado en MCP
func (r *ReferenceRepository) ObtenerTodasLasMarcas(ctx context.Context, soloActivas bool) ([]*domain.Brand, error) {

	query := fmt.Sprintf(`
		SELECT id, name, slug, country, website, logo_url, is_active, created_at
		FROM %s.brands
	`, r.bikeSchema)

	if soloActivas {
		query += " WHERE is_active = true"
	}
	query += " ORDER BY name ASC"

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var marcas []*domain.Brand
	for rows.Next() {
		var m domain.Brand
		err := rows.Scan(&m.ID, &m.Name, &m.Slug, &m.Country, &m.Website,
			&m.LogoURL, &m.IsActive, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		marcas = append(marcas, &m)
	}

	return marcas, rows.Err()
}

// BuscarMarcaPorNombre - Búsqueda flexible (case-insensitive, partial match)
func (r *ReferenceRepository) BuscarMarcaPorNombre(ctx context.Context, nombre string) (*domain.Brand, error) {

	query := fmt.Sprintf(`
		SELECT id, name, slug, country, website, logo_url, is_active, created_at
		FROM %s.brands
		WHERE LOWER(name) = LOWER($1) OR LOWER(slug) = LOWER($1)
		AND is_active = true
		LIMIT 1
	`, r.bikeSchema)

	var m domain.Brand
	err := r.pool.QueryRow(ctx, query, nombre).Scan(
		&m.ID, &m.Name, &m.Slug, &m.Country, &m.Website,
		&m.LogoURL, &m.IsActive, &m.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // No encontrada, no es error
	}
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// ===========================================
// BikeTypes
// ===========================================

// ObtenerTodosLosTipos - Para filtrado en MCP
func (r *ReferenceRepository) ObtenerTodosLosTipos(ctx context.Context) ([]*domain.BikeType, error) {

	query := fmt.Sprintf(`
		SELECT id, name, slug, description, icon_name, display_order, size_system_id
		FROM %s.bike_types
		ORDER BY display_order ASC, name ASC
	`, r.bikeSchema)

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tipos []*domain.BikeType
	for rows.Next() {
		var t domain.BikeType
		err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Description,
			&t.IconName, &t.DisplayOrder, &t.SizeSystemID)
		if err != nil {
			return nil, err
		}
		tipos = append(tipos, &t)
	}

	return tipos, rows.Err()
}

// ===========================================
// StandardColors
// ===========================================

// ObtenerColoresPorFamilia - Para sugerencias de color
func (r *ReferenceRepository) ObtenerColoresPorFamilia(ctx context.Context, familia *string) ([]*domain.StandardColor, error) {

	query := fmt.Sprintf(`
		SELECT id, name, name_es, hex_code, color_family, display_order
		FROM %s.standard_colors
	`, r.bikeSchema)

	args := []interface{}{}
	argIndex := 1

	if familia != nil && *familia != "" {
		query += fmt.Sprintf(" WHERE LOWER(color_family) = LOWER($%d)", argIndex)
		args = append(args, *familia)
		argIndex++
	}

	query += " ORDER BY display_order ASC, name ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var colores []*domain.StandardColor
	for rows.Next() {
		var c domain.StandardColor
		err := rows.Scan(&c.ID, &c.Name, &c.NameES, &c.HexCode,
			&c.ColorFamily, &c.DisplayOrder)
		if err != nil {
			return nil, err
		}
		colores = append(colores, &c)
	}

	return colores, rows.Err()
}

// ===========================================
// BikeCatalog - Matching inteligente
// ===========================================

// BuscarEnCatalogo - Búsqueda flexible para matching con OCR
func (r *ReferenceRepository) BuscarEnCatalogo(
	ctx context.Context,
	marca, modelo *string,
	anio *int, // ← anio como *int ✅
	limite int,
) ([]*domain.BikeCatalog, error) {

	// Query con JOINs para traer marca y tipo en una sola consulta
	query := fmt.Sprintf(`
		SELECT 
			bc.id, bc.brand_id, bc.model_name, bc.model_year,
			bc.bike_type_id, bc.frame_material, bc.groupset_model,
			bc.brake_type, bc.msrp_usd, bc.msrp_ars, bc.weight_kg,
			bc.product_url, bc.is_active, bc.created_at, bc.updated_at,
			-- Marca (JOIN)
			b.id as brand_id, b.name as brand_name, b.slug as brand_slug,
			-- Tipo de bici (JOIN)
			bt.id as type_id, bt.name as type_name, bt.slug as type_slug
		FROM %s.bike_catalog bc
		LEFT JOIN %s.brands b ON bc.brand_id = b.id
		LEFT JOIN %s.bike_types bt ON bc.bike_type_id = bt.id
		WHERE bc.is_active = true
	`, r.bikeSchema, r.bikeSchema, r.bikeSchema)

	args := []interface{}{}
	argIndex := 1

	// Filtro por marca (nombre o slug)
	/*
		if marcaNombre != nil && *marcaNombre != "" {
			query += fmt.Sprintf(` AND (
				LOWER(b.name) = LOWER($%d) OR
				LOWER(b.slug) = LOWER($%d)
			)`, argIndex, argIndex)
			args = append(args, *marcaNombre, *marcaNombre)
			argIndex++
		}
	*/

	// Filtro por modelo (partial match, case-insensitive)
	if modelo != nil && *modelo != "" {
		query += fmt.Sprintf(` AND LOWER(bc.model_name) LIKE LOWER('%%' || $%d || '%%')`, argIndex)
		args = append(args, *modelo)
		argIndex++
	}

	// Filtro por año (rango ±1 año para tolerancia)
	if anio != nil && *anio > 0 { // ← Ahora anio es *int
		query += fmt.Sprintf(` AND bc.model_year BETWEEN ($%d - 1) AND ($%d + 1)`, argIndex, argIndex)
		args = append(args, *anio, *anio) // ← Pasar el valor int, no string
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY bc.model_year DESC, bc.model_name ASC LIMIT $%d", argIndex)
	args = append(args, limite)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultados []*domain.BikeCatalog
	for rows.Next() {
		var cat domain.BikeCatalog
		var brand domain.Brand
		var bikeType domain.BikeType

		err := rows.Scan(
			// BikeCatalog fields
			&cat.ID, &cat.BrandID, &cat.ModelName, &cat.ModelYear,
			&cat.BikeTypeID, &cat.FrameMaterial, &cat.GroupsetModel,
			&cat.BrakeType, &cat.MSRPUSD, &cat.MSRPARS, &cat.WeightKG,
			&cat.ProductURL, &cat.IsActive, &cat.CreatedAt, &cat.UpdatedAt,
			// Brand fields (from JOIN)
			&brand.ID, &brand.Name, &brand.Slug,
			// BikeType fields (from JOIN)
			&bikeType.ID, &bikeType.Name, &bikeType.Slug,
		)
		if err != nil {
			return nil, err
		}

		// Asignar JOINs a structs anidados
		cat.Brand = &brand
		cat.BikeType = &bikeType

		resultados = append(resultados, &cat)
	}

	return resultados, rows.Err()
}

// ObtenerCatalogoPorID - Para detalles completos de un modelo
func (r *ReferenceRepository) ObtenerCatalogoPorID(ctx context.Context, id int) (*domain.BikeCatalog, error) {

	query := fmt.Sprintf(`
		SELECT bc.id, bc.brand_id, bc.model_name, bc.model_year,
		       bc.bike_type_id, bc.frame_material, bc.groupset_model,
		       bc.brake_type, bc.msrp_usd, bc.msrp_ars, bc.weight_kg,
		       bc.product_url, bc.is_active, bc.created_at, bc.updated_at
		FROM %s.bike_catalog bc
		WHERE bc.id = $1 AND bc.is_active = true
	`, r.bikeSchema)

	var cat domain.BikeCatalog
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&cat.ID, &cat.BrandID, &cat.ModelName, &cat.ModelYear,
		&cat.BikeTypeID, &cat.FrameMaterial, &cat.GroupsetModel,
		&cat.BrakeType, &cat.MSRPUSD, &cat.MSRPARS, &cat.WeightKG,
		&cat.ProductURL, &cat.IsActive, &cat.CreatedAt, &cat.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &cat, nil
}

// ===========================================
// Métodos Stub para cumplir domain.ReferenceRepository
// ===========================================

// ObtenerMarcaPorID recupera una marca por su ID
func (r *ReferenceRepository) ObtenerMarcaPorID(ctx context.Context, id int) (*domain.Brand, error) {

	query := fmt.Sprintf(`
		SELECT id, name, slug, country, website, logo_url, is_active, created_at
		FROM %s.brands
		WHERE id = $1 AND is_active = true
	`, r.bikeSchema)

	var m domain.Brand
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.Name, &m.Slug, &m.Country, &m.Website,
		&m.LogoURL, &m.IsActive, &m.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// ObtenerTipoPorID recupera un tipo de bicicleta por su ID
func (r *ReferenceRepository) ObtenerTipoPorID(ctx context.Context, id int) (*domain.BikeType, error) {

	query := fmt.Sprintf(`
		SELECT id, name, slug, description, icon_name, display_order, size_system_id
		FROM %s.bike_types
		WHERE id = $1
	`, r.bikeSchema)

	var t domain.BikeType
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Description,
		&t.IconName, &t.DisplayOrder, &t.SizeSystemID,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// ObtenerColorPorID recupera un color estándar por su ID ⭐ (este era el que faltaba)
func (r *ReferenceRepository) ObtenerColorPorID(ctx context.Context, id int) (*domain.StandardColor, error) {

	query := fmt.Sprintf(`
		SELECT id, name, name_es, hex_code, color_family, display_order
		FROM %s.standard_colors
		WHERE id = $1
	`, r.bikeSchema)

	var c domain.StandardColor
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.NameES, &c.HexCode,
		&c.ColorFamily, &c.DisplayOrder,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

