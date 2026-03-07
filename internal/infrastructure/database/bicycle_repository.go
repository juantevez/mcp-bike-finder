// internal/infrastructure/database/bicycle_repository.go

package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juantevez/mcp-bike-finder/internal/domain"
)

type BicycleRepository struct {
	pool   *pgxpool.Pool
	schema string // "bike"
}

func NewBicycleRepository(pool *pgxpool.Pool, schema string) *BicycleRepository {
	return &BicycleRepository{
		pool:   pool,
		schema: schema,
	}
}

// ===========================================
// Queries para registered_bicycles
// ===========================================

// GuardarBicicleta inserta o actualiza (UPSERT)
func (r *BicycleRepository) Guardar(ctx context.Context, bici *domain.Bicicleta) error {

	// ✅ Sincronización INLINE: alias → campos de DB antes de guardar
	// (En lugar de llamar a bici.SyncToDB())
	if bici.Marca != "" && bici.FrameModel == nil {
		bici.FrameModel = &bici.Marca
	}
	if bici.Modelo != "" && bici.FrameModel == nil {
		bici.FrameModel = &bici.Modelo
	}
	if bici.Anio > 0 && bici.FrameYear == nil {
		bici.FrameYear = &bici.Anio
	}
	if bici.Talle != "" && bici.FrameSizeRaw == nil {
		bici.FrameSizeRaw = &bici.Talle
	}
	if bici.Color != "" && bici.PrimaryColorCustom == nil {
		bici.PrimaryColorCustom = &bici.Color
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.registered_bicycles 
		(id, user_id, registration_type, frame_model, frame_year, frame_size_raw,
		 primary_color_custom, color_description, components, detailed_specs,
		 estimated_current_value, purchase_price, purchase_currency, status, notes,
		 created_at, updated_at, "version")
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (id) DO UPDATE SET
			frame_model = EXCLUDED.frame_model,
			frame_year = EXCLUDED.frame_year,
			frame_size_raw = EXCLUDED.frame_size_raw,
			primary_color_custom = EXCLUDED.primary_color_custom,
			color_description = EXCLUDED.color_description,
			components = EXCLUDED.components,
			detailed_specs = EXCLUDED.detailed_specs,
			estimated_current_value = EXCLUDED.estimated_current_value,
			purchase_price = EXCLUDED.purchase_price,
			status = EXCLUDED.status,
			notes = EXCLUDED.notes,
			updated_at = EXCLUDED.updated_at,
			"version" = %s.registered_bicycles."version" + 1
	`, r.schema, r.schema)

	// Convertir maps a JSONB
	componentsJSON, _ := json.Marshal(bici.Components)
	specsJSON, _ := json.Marshal(bici.DetailedSpecs)

	_, err := r.pool.Exec(ctx, query,
		bici.ID, bici.UserID, bici.RegistrationType, bici.FrameModel, bici.FrameYear,
		bici.FrameSizeRaw, bici.PrimaryColorCustom, bici.ColorDescription,
		string(componentsJSON), string(specsJSON),
		bici.EstimatedCurrentValue, bici.PurchasePrice, bici.PurchaseCurrency,
		bici.Status, bici.Notes, bici.CreatedAt, bici.UpdatedAt, bici.Version,
	)

	return err
}

// ObtenerPorID con JOIN para traer fotos primarias
func (r *BicycleRepository) ObtenerPorID(ctx context.Context, id string) (*domain.Bicicleta, error) {

	query := fmt.Sprintf(`
		SELECT 
			rb.id, rb.user_id, rb.registration_type, rb.frame_model, rb.frame_year,
			rb.frame_size_raw, rb.primary_color_custom, rb.color_description,
			rb.components, rb.detailed_specs, rb.estimated_current_value,
			rb.purchase_price, rb.purchase_currency, rb.status, rb.notes,
			rb.created_at, rb.updated_at, rb."version"
		FROM %s.registered_bicycles rb
		WHERE rb.id = $1 AND rb.status = 'ACTIVE'
	`, r.schema)

	row := r.pool.QueryRow(ctx, query, id)

	var bici domain.Bicicleta
	var componentsJSON, specsJSON []byte

	err := row.Scan(
		&bici.ID, &bici.UserID, &bici.RegistrationType, &bici.FrameModel, &bici.FrameYear,
		&bici.FrameSizeRaw, &bici.PrimaryColorCustom, &bici.ColorDescription,
		&componentsJSON, &specsJSON, &bici.EstimatedCurrentValue,
		&bici.PurchasePrice, &bici.PurchaseCurrency, &bici.Status, &bici.Notes,
		&bici.CreatedAt, &bici.UpdatedAt, &bici.Version,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("bicicleta no encontrada: %s", id)
		}
		return nil, fmt.Errorf("error consultando bicicleta: %w", err)
	}

	// Parsear JSONB a maps
	if len(componentsJSON) > 0 {
		json.Unmarshal(componentsJSON, &bici.Components)
	}
	if len(specsJSON) > 0 {
		json.Unmarshal(specsJSON, &bici.DetailedSpecs)
	}

	// ✅ Sincronización INLINE: DB → alias para uso conveniente en lógica de negocio
	// (En lugar de llamar a bici.SyncFromDB())
	if bici.FrameModel != nil {
		bici.Marca = *bici.FrameModel
		bici.Modelo = *bici.FrameModel
	}
	if bici.FrameYear != nil {
		bici.Anio = *bici.FrameYear
	}
	if bici.FrameSizeRaw != nil {
		bici.Talle = *bici.FrameSizeRaw
	}
	if bici.PrimaryColorCustom != nil {
		bici.Color = *bici.PrimaryColorCustom
	}

	return &bici, nil
}

// BuscarPorMarcaModeloTalle - Query optimizada para búsqueda
func (r *BicycleRepository) BuscarPorMarcaModeloTalle(
	ctx context.Context,
	marca, modelo, talle string,
	limite int,
) ([]*domain.Bicicleta, error) {

	query := fmt.Sprintf(`
		SELECT id, user_id, frame_model, frame_year, frame_size_raw,
		       primary_color_custom, components, estimated_current_value, status
		FROM %s.registered_bicycles
		WHERE status = 'ACTIVE'
		  AND ($1::varchar = '' OR frame_brand_id IN (
		      SELECT id FROM bike.brands WHERE LOWER(name) = LOWER($1)
		  ))
		  AND ($2::varchar = '' OR LOWER(frame_model) LIKE LOWER('%%' || $2 || '%%'))
		  AND ($3::varchar = '' OR frame_size_raw = $3 OR frame_size_id IN (
		      SELECT id FROM bike.frame_sizes WHERE LOWER(size_label) = LOWER($3)
		  ))
		ORDER BY updated_at DESC
		LIMIT $4
	`, r.schema)

	rows, err := r.pool.Query(ctx, query, marca, modelo, talle, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bicicletas []*domain.Bicicleta
	for rows.Next() {
		var bici domain.Bicicleta
		var componentsJSON []byte

		err := rows.Scan(
			&bici.ID, &bici.UserID, &bici.FrameModel, &bici.FrameYear,
			&bici.FrameSizeRaw, &bici.PrimaryColorCustom, &componentsJSON,
			&bici.EstimatedCurrentValue, &bici.Status,
		)
		if err != nil {
			return nil, err
		}

		if len(componentsJSON) > 0 {
			json.Unmarshal(componentsJSON, &bici.Components)
		}

		bicicletas = append(bicicletas, &bici)
	}

	return bicicletas, rows.Err()
}

// ===========================================
// Queries para bicycle_photos
// ===========================================

// ObtenerFotosPorBicicleta - Incluye filtro por tipo y primary
func (r *BicycleRepository) ObtenerFotosPorBicicleta(
	ctx context.Context,
	bicycleID string,
	photoType *string,
	onlyPrimary bool,
) ([]*domain.BicyclePhoto, error) {

	query := fmt.Sprintf(`
		SELECT id, bicycle_id, uploaded_by, file_name, content_type,
		       file_size_bytes, storage_path, photo_type, is_primary,
		       description, exif_latitude, exif_longitude, exif_date_time,
		       exif_camera_make, exif_camera_model, uploaded_at
		FROM %s.bicycle_photos
		WHERE bicycle_id = $1
		  AND ($2::varchar IS NULL OR photo_type = $2)
		  AND ($3 = false OR is_primary = true)
		ORDER BY is_primary DESC, uploaded_at DESC
	`, r.schema)

	rows, err := r.pool.Query(ctx, query, bicycleID, photoType, onlyPrimary)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fotos []*domain.BicyclePhoto
	for rows.Next() {
		var foto domain.BicyclePhoto
		err := rows.Scan(
			&foto.ID, &foto.BicycleID, &foto.UploadedBy, &foto.FileName,
			&foto.ContentType, &foto.FileSizeBytes, &foto.StoragePath,
			&foto.PhotoType, &foto.IsPrimary, &foto.Description,
			&foto.EXIFLatitude, &foto.EXIFLongitude, &foto.EXIFDateTime,
			&foto.EXIFCameraMake, &foto.EXIFCameraModel, &foto.UploadedAt,
		)
		if err != nil {
			return nil, err
		}
		fotos = append(fotos, &foto)
	}

	return fotos, rows.Err()
}

// GuardarFoto - Inserta nueva foto con validación de FK
func (r *BicycleRepository) GuardarFoto(ctx context.Context, foto *domain.BicyclePhoto) error {

	query := fmt.Sprintf(`
		INSERT INTO %s.bicycle_photos 
		(id, bicycle_id, uploaded_by, file_name, content_type, file_size_bytes,
		 storage_path, photo_type, is_primary, description,
		 exif_latitude, exif_longitude, exif_date_time,
		 exif_camera_make, exif_camera_model, uploaded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, r.schema)

	_, err := r.pool.Exec(ctx, query,
		foto.ID, foto.BicycleID, foto.UploadedBy, foto.FileName,
		foto.ContentType, foto.FileSizeBytes, foto.StoragePath,
		foto.PhotoType, foto.IsPrimary, foto.Description,
		foto.EXIFLatitude, foto.EXIFLongitude, foto.EXIFDateTime,
		foto.EXIFCameraMake, foto.EXIFCameraModel, foto.UploadedAt,
	)

	return err
}

// ===========================================
// Helpers y Queries Avanzadas
// ===========================================

// ObtenerBicicletaConFotos - Query con JOIN para traer bicicleta + sus fotos
func (r *BicycleRepository) ObtenerBicicletaConFotos(ctx context.Context, bicycleID string) (*domain.Bicicleta, []*domain.BicyclePhoto, error) {

	// 1. Obtener bicicleta
	bici, err := r.ObtenerPorID(ctx, bicycleID)
	if err != nil {
		return nil, nil, err
	}

	// 2. Obtener fotos (solo las primeras 5 para no sobrecargar)
	fotos, err := r.ObtenerFotosPorBicicleta(ctx, bicycleID, nil, false)
	if err != nil {
		return nil, nil, err
	}

	// Limitar a 5 fotos para respuesta MCP
	if len(fotos) > 5 {
		fotos = fotos[:5]
	}

	return bici, fotos, nil
}

// ActualizarComponentes - Update parcial de JSONB (sin tocar otros campos)
func (r *BicycleRepository) ActualizarComponentes(ctx context.Context, bicycleID string, nuevosComponentes map[string]interface{}) error {

	componentsJSON, err := json.Marshal(nuevosComponentes)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		UPDATE %s.registered_bicycles
		SET components = $1,
		    updated_at = CURRENT_TIMESTAMP,
		    "version" = "version" + 1
		WHERE id = $2
	`, r.schema)

	_, err = r.pool.Exec(ctx, query, string(componentsJSON), bicycleID)
	return err
}

// ContarBicicletasPorUsuario - Para estadísticas
func (r *BicycleRepository) ContarBicicletasPorUsuario(ctx context.Context, userID string, status *string) (int, error) {

	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.registered_bicycles
		WHERE user_id = $1
		  AND ($2::varchar IS NULL OR status = $2)
	`, r.schema)

	var count int
	err := r.pool.QueryRow(ctx, query, userID, status).Scan(&count)
	return count, err
}

// internal/infrastructure/database/bicycle_repository.go

// ===========================================
// Nuevo Método: ObtenerPorImagenURL
// ===========================================

// ObtenerPorImagenURL busca una bicicleta por su URL de imagen en S3
// Útil para evitar duplicados al procesar imágenes
func (r *BicycleRepository) ObtenerPorImagenURL(ctx context.Context, imagenS3URL string) (*domain.Bicicleta, error) {

	// Query simple: buscar por la URL de imagen
	// Nota: En producción, considera agregar un índice en la columna que almacena la URL
	// o usar una tabla intermedia si las imágenes están en bicycle_photos
	query := fmt.Sprintf(`
		SELECT id, user_id, registration_type, frame_model, frame_year,
		       frame_size_raw, primary_color_custom, color_description,
		       components, detailed_specs, estimated_current_value,
		       purchase_price, purchase_currency, status, notes,
		       created_at, updated_at, "version"
		FROM %s.registered_bicycles
		WHERE status = 'ACTIVE'
		  AND (
		    -- Opción A: Si la URL está en un campo directo (ej: primary_image_url)
		    -- primary_image_url = $1
		    -- Opción B: Si la URL está en el JSONB 'photos'
		    photos::text LIKE $1
		    -- Opción C: Si usas la tabla bicycle_photos (JOIN)
		    -- EXISTS (
		    --   SELECT 1 FROM %s.bicycle_photos bp 
		    --   WHERE bp.bicycle_id = registered_bicycles.id 
		    --   AND bp.storage_path = $1
		    -- )
		  )
		LIMIT 1
	`, r.schema)

	row := r.pool.QueryRow(ctx, query, "%"+imagenS3URL+"%")

	var bici domain.Bicicleta
	var componentsJSON, specsJSON []byte

	err := row.Scan(
		&bici.ID, &bici.UserID, &bici.RegistrationType, &bici.FrameModel, &bici.FrameYear,
		&bici.FrameSizeRaw, &bici.PrimaryColorCustom, &bici.ColorDescription,
		&componentsJSON, &specsJSON, &bici.EstimatedCurrentValue,
		&bici.PurchasePrice, &bici.PurchaseCurrency, &bici.Status, &bici.Notes,
		&bici.CreatedAt, &bici.UpdatedAt, &bici.Version,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No encontrada, no es error
		}
		return nil, fmt.Errorf("error buscando por imagen URL: %w", err)
	}

	// Parsear JSONB a maps
	if len(componentsJSON) > 0 {
		json.Unmarshal(componentsJSON, &bici.Components)
	}
	if len(specsJSON) > 0 {
		json.Unmarshal(specsJSON, &bici.DetailedSpecs)
	}

	// Sincronización inline: DB → alias
	if bici.FrameModel != nil {
		bici.Marca = *bici.FrameModel
		bici.Modelo = *bici.FrameModel
	}
	if bici.FrameYear != nil {
		bici.Anio = *bici.FrameYear
	}
	if bici.FrameSizeRaw != nil {
		bici.Talle = *bici.FrameSizeRaw
	}
	if bici.PrimaryColorCustom != nil {
		bici.Color = *bici.PrimaryColorCustom
	}

	return &bici, nil
}

// Actualizar actualiza una bicicleta existente en la base de datos
func (r *BicycleRepository) Actualizar(ctx context.Context, bici *domain.Bicicleta) error {

	query := fmt.Sprintf(`
		UPDATE %s.registered_bicycles
		SET frame_model = $1,
		    frame_year = $2,
		    frame_size_raw = $3,
		    primary_color_custom = $4,
		    color_description = $5,
		    components = $6,
		    detailed_specs = $7,
		    estimated_current_value = $8,
		    purchase_price = $9,
		    purchase_currency = $10,
		    status = $11,
		    notes = $12,
		    updated_at = CURRENT_TIMESTAMP,
		    "version" = "version" + 1
		WHERE id = $13
	`, r.schema)

	// Convertir maps a JSONB
	componentsJSON, _ := json.Marshal(bici.Components)
	specsJSON, _ := json.Marshal(bici.DetailedSpecs)

	_, err := r.pool.Exec(ctx, query,
		bici.FrameModel, bici.FrameYear, bici.FrameSizeRaw,
		bici.PrimaryColorCustom, bici.ColorDescription,
		string(componentsJSON), string(specsJSON),
		bici.EstimatedCurrentValue, bici.PurchasePrice, bici.PurchaseCurrency,
		bici.Status, bici.Notes, bici.ID,
	)

	return err
}

// ===========================================
// Métodos Stub (para cumplir la interfaz)
// ===========================================

// Eliminar marca una bicicleta como inactiva (soft delete)
func (r *BicycleRepository) Eliminar(ctx context.Context, id string) error {

	// Soft delete: actualizar status en lugar de borrar físicamente
	query := fmt.Sprintf(`
		UPDATE %s.registered_bicycles
		SET status = 'INACTIVE',
		    updated_at = CURRENT_TIMESTAMP,
		    "version" = "version" + 1
		WHERE id = $1
	`, r.schema)

	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// ===========================================
// Métodos de la Interfaz domain.BicicletaRepository
// ===========================================

// ObtenerPorMarcaModelo busca bicicletas por marca y modelo
func (r *BicycleRepository) ObtenerPorMarcaModelo(ctx context.Context, marca, modelo string) ([]*domain.Bicicleta, error) {

	query := fmt.Sprintf(`
		SELECT id, user_id, registration_type, frame_model, frame_year,
		       frame_size_raw, primary_color_custom, color_description,
		       components, detailed_specs, estimated_current_value,
		       purchase_price, purchase_currency, status, notes,
		       created_at, updated_at, "version"
		FROM %s.registered_bicycles
		WHERE status = 'ACTIVE'
		  AND ($1::varchar = '' OR LOWER(frame_model) LIKE LOWER('%%' || $1 || '%%'))
		  AND ($2::varchar = '' OR LOWER(frame_model) LIKE LOWER('%%' || $2 || '%%'))
		ORDER BY created_at DESC
		LIMIT 50
	`, r.schema)

	rows, err := r.pool.Query(ctx, query, marca, modelo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bicicletas []*domain.Bicicleta
	for rows.Next() {
		var bici domain.Bicicleta
		var componentsJSON, specsJSON []byte

		err := rows.Scan(
			&bici.ID, &bici.UserID, &bici.RegistrationType, &bici.FrameModel, &bici.FrameYear,
			&bici.FrameSizeRaw, &bici.PrimaryColorCustom, &bici.ColorDescription,
			&componentsJSON, &specsJSON, &bici.EstimatedCurrentValue,
			&bici.PurchasePrice, &bici.PurchaseCurrency, &bici.Status, &bici.Notes,
			&bici.CreatedAt, &bici.UpdatedAt, &bici.Version,
		)
		if err != nil {
			return nil, err
		}

		// Parsear JSONB a maps
		if len(componentsJSON) > 0 {
			json.Unmarshal(componentsJSON, &bici.Components)
		}
		if len(specsJSON) > 0 {
			json.Unmarshal(specsJSON, &bici.DetailedSpecs)
		}

		// Sincronización inline: DB → alias
		if bici.FrameModel != nil {
			bici.Marca = *bici.FrameModel
			bici.Modelo = *bici.FrameModel
		}
		if bici.FrameYear != nil {
			bici.Anio = *bici.FrameYear
		}
		if bici.FrameSizeRaw != nil {
			bici.Talle = *bici.FrameSizeRaw
		}
		if bici.PrimaryColorCustom != nil {
			bici.Color = *bici.PrimaryColorCustom
		}

		bicicletas = append(bicicletas, &bici)
	}

	return bicicletas, rows.Err()
}
