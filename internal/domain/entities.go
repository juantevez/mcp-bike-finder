// internal/domain/entities.go
package domain

import (
	"fmt"
	"time"
)

// ===========================================
// Bicicleta (registered_bicycles)
// ===========================================

type Bicicleta struct {
	// ===========================================
	// Campos Primarios (PK y FK)
	// ===========================================
	ID               string `json:"id"`
	UserID           string `json:"user_id"`
	RegistrationType string `json:"registration_type"` // CATALOG | MANUAL

	// ===========================================
	// Campos de Base de Datos (mapeo directo)
	// ===========================================
	CatalogBikeID        *int    `json:"catalog_bike_id,omitempty"`
	FrameBrandID         *int    `json:"frame_brand_id,omitempty"`
	FrameModel           *string `json:"frame_model,omitempty"`
	FrameYear            *int    `json:"frame_year,omitempty"`
	BikeTypeID           *int    `json:"bike_type_id,omitempty"`
	FrameSizeID          *int    `json:"frame_size_id,omitempty"`
	FrameSizeRaw         *string `json:"frame_size_raw,omitempty"`
	PrimaryColorID       *int    `json:"primary_color_id,omitempty"`
	PrimaryColorCustom   *string `json:"primary_color_custom,omitempty"`
	SecondaryColorID     *int    `json:"secondary_color_id,omitempty"`
	AccentColorID        *int    `json:"accent_color_id,omitempty"`
	ColorDescription     *string `json:"color_description,omitempty"`
	SerialNumber         *string `json:"serial_number,omitempty"`
	SerialNumberLocation *string `json:"serial_number_location,omitempty"`

	// ===========================================
	// Campos JSONB (datos flexibles)
	// ===========================================
	Components          map[string]interface{} `json:"components,omitempty"`
	DetailedSpecs       map[string]interface{} `json:"detailed_specs,omitempty"`
	DistinguishingMarks []string               `json:"distinguishing_marks,omitempty"`
	Photos              []string               `json:"photos,omitempty"`

	// ===========================================
	// Valor, Compra y Seguro
	// ===========================================
	EstimatedCurrentValue *float64   `json:"estimated_current_value,omitempty"`
	PurchaseDate          *time.Time `json:"purchase_date,omitempty"`
	PurchasePrice         *float64   `json:"purchase_price,omitempty"`
	PurchaseCurrency      string     `json:"purchase_currency"`
	PurchaseMethod        *string    `json:"purchase_method,omitempty"`
	PurchaseReceiptURL    *string    `json:"purchase_receipt_url,omitempty"`
	InsurancePolicyNumber *string    `json:"insurance_policy_number,omitempty"`

	// ===========================================
	// Estado y Metadatos
	// ===========================================
	Status    string    `json:"status"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`

	// ===========================================
	// ⚠️ CAMPOS ALIAS PARA LÓGICA DE NEGOCIO (MCP)
	// Estos simplifican el acceso vs los campos de DB con prefijos
	// Deben estar DENTRO del struct para ser válidos
	// ===========================================

	// Marca/Modelo/Año (alias para FrameModel/FrameYear)
	Marca  string `json:"marca,omitempty"`  // Alias simplificado para lógica de negocio
	Modelo string `json:"modelo,omitempty"` // Alias simplificado para lógica de negocio
	Anio   int    `json:"anio,omitempty"`   // Alias simplificado para lógica de negocio

	// Talle (alias para FrameSizeRaw)
	Talle string `json:"talle,omitempty"` // Alias: "M", "54cm", "L", etc.

	// URL de imagen principal (derivado de Photos o BicyclePhoto)
	ImagenS3URL string `json:"imagen_s3_url,omitempty"` // URL S3 de la foto principal

	// Color principal (alias para PrimaryColorCustom)
	Color string `json:"color,omitempty"` // Alias simplificado para lógica de negocio

	Tipo        string `json:"tipo,omitempty"`  // Alias: "mountain_bike", "road_bike", etc.
	
}

// ===========================================
// BicyclePhoto (bicycle_photos)
// ===========================================

type BicyclePhoto struct {
	ID            string  `json:"id"`
	BicycleID     string  `json:"bicycle_id"`
	UploadedBy    string  `json:"uploaded_by"`
	FileName      string  `json:"file_name"`
	ContentType   string  `json:"content_type"`
	FileSizeBytes int64   `json:"file_size_bytes"`
	StoragePath   *string `json:"storage_path,omitempty"` // S3 key
	PhotoType     string  `json:"photo_type"`             // "frame", "component", etc.
	IsPrimary     *bool   `json:"is_primary,omitempty"`
	Description   *string `json:"description,omitempty"`

	// EXIF data
	EXIFLatitude    *float64   `json:"exif_latitude,omitempty"`
	EXIFLongitude   *float64   `json:"exif_longitude,omitempty"`
	EXIFDateTime    *time.Time `json:"exif_date_time,omitempty"`
	EXIFCameraMake  *string    `json:"exif_camera_make,omitempty"`
	EXIFCameraModel *string    `json:"exif_camera_model,omitempty"`
	EXIFOrientation *int       `json:"exif_orientation,omitempty"`

	UploadedAt time.Time `json:"uploaded_at"`
}

// Helper para URL S3
func (p *BicyclePhoto) GetS3URL(bucket, region string) string {
	if p.StoragePath == nil || *p.StoragePath == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, *p.StoragePath)
}

// ===========================================
// BicicletaInfo (Resultado de OCR/Vision)
// ===========================================

type BicicletaInfo struct {
	Marca       string          `json:"marca"`
	Modelo      string          `json:"modelo"`
	Anio        int             `json:"anio,omitempty"`
	Color       string          `json:"color"`
	Talle       string          `json:"talle"`
	Tipo        string          `json:"tipo"` // mountain_bike, road_bike, etc.
	Componentes ComponentesBici `json:"componentes,omitempty"`
}

type ComponentesBici struct {
	Asiento     string `json:"asiento,omitempty"`
	Tija        string `json:"tija,omitempty"`
	Manubrio    string `json:"manubrio,omitempty"`
	Pedales     string `json:"pedales,omitempty"`
	Suspension  string `json:"suspension,omitempty"`
	Transmision string `json:"transmision,omitempty"`
	Frenos      string `json:"frenos,omitempty"`
	Ruedas      string `json:"ruedas,omitempty"`
}

// ===========================================
// ListadoMarketplace (Resultados de scraping)
// ===========================================

type ListadoMarketplace struct {
	ID          string    `json:"id"`
	Titulo      string    `json:"titulo"`
	Precio      float64   `json:"precio"`
	Moneda      string    `json:"moneda"`
	URL         string    `json:"url"`
	Marketplace string    `json:"marketplace"`
	Ubicacion   string    `json:"ubicacion,omitempty"`
	Vendedor    string    `json:"vendedor,omitempty"`
	ImagenURL   string    `json:"imagen_url,omitempty"`
	Fecha       time.Time `json:"fecha"`
}

// ===========================================
// BusquedaHistorial
// ===========================================

type BusquedaHistorial struct {
	ID          string    `json:"id"`
	UsuarioID   string    `json:"usuario_id"`
	BicicletaID string    `json:"bicicleta_id"`
	Criterios   string    `json:"criterios"`
	Resultados  int       `json:"resultados"`
	CreatedAt   time.Time `json:"created_at"`
}

// ===========================================
// Tablas de Referencia (Lookup Tables)
// ===========================================

type Brand struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Country   *string   `json:"country,omitempty"`
	Website   *string   `json:"website,omitempty"`
	LogoURL   *string   `json:"logo_url,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type BikeType struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Description  *string `json:"description,omitempty"`
	IconName     *string `json:"icon_name,omitempty"`
	DisplayOrder int     `json:"display_order"`
	SizeSystemID *int    `json:"size_system_id,omitempty"`
}

type StandardColor struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	NameES       *string `json:"name_es,omitempty"`
	HexCode      *string `json:"hex_code,omitempty"`
	ColorFamily  *string `json:"color_family,omitempty"`
	DisplayOrder int     `json:"display_order"`
}

type BikeCatalog struct {
	ID              int       `json:"id"`
	BrandID         int       `json:"brand_id"`
	Brand           *Brand    `json:"brand,omitempty"`
	ModelName       string    `json:"model_name"`
	ModelYear       int       `json:"model_year"`
	BikeTypeID      int       `json:"bike_type_id"`
	BikeType        *BikeType `json:"bike_type,omitempty"`
	FrameMaterial   *string   `json:"frame_material,omitempty"`
	GroupsetBrandID *int      `json:"groupset_brand_id,omitempty"`
	GroupsetModel   *string   `json:"groupset_model,omitempty"`
	SpeedConfigID   *int      `json:"speed_config_id,omitempty"`
	BrakeType       *string   `json:"brake_type,omitempty"`
	MSRPUSD         *float64  `json:"msrp_usd,omitempty"`
	MSRPARS         *float64  `json:"msrp_ars,omitempty"`
	WeightKG        *float64  `json:"weight_kg,omitempty"`
	ProductURL      *string   `json:"product_url,omitempty"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ===========================================
// User (auth.users - solo campos necesarios)
// ===========================================

type User struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	FullName      *string    `json:"full_name,omitempty"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	Status        string     `json:"status"`
	CountryName   *string    `json:"country_name,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
}

// ===========================================
// VisionResult (Output de análisis de imagen)
// ===========================================

type VisionResult struct {
	ColorDominante    string   `json:"color_dominante,omitempty"`
	TipoBicicleta     string   `json:"tipo_bicicleta,omitempty"` // mountain, road, hybrid
	ObjetosDetectados []string `json:"objetos_detectados,omitempty"`
	Confianza         float64  `json:"confianza,omitempty"`
}

type ImageValidation struct {
	EsBicicleta bool     `json:"es_bicicicleta"`
	Confianza   float64  `json:"confianza"`
	Mensaje     string   `json:"mensaje,omitempty"`
	Objetos     []string `json:"objetos,omitempty"`
}

// ===========================================
// Alerta (posible coincidencia de bicicleta robada)
// ===========================================

const (
	AlertaStatusNueva      = "NUEVA"
	AlertaStatusRevisada   = "REVISADA"
	AlertaStatusDescartada = "DESCARTADA"
	AlertaStatusConfirmada = "CONFIRMADA"
)

type Alerta struct {
	ID             string    `json:"id"`
	BicicletaID    string    `json:"bicicleta_id"`
	UsuarioID      string    `json:"usuario_id"`
	Titulo         string    `json:"titulo"`          // título del listado encontrado
	URL            string    `json:"url"`             // URL del listado en el marketplace
	Marketplace    string    `json:"marketplace"`
	Precio         float64   `json:"precio"`
	ScoreSimilitud float64   `json:"score_similitud"` // 0-100
	Status         string    `json:"status"`          // NUEVA | REVISADA | DESCARTADA | CONFIRMADA
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SyncToDB actualiza los campos de DB desde los alias (antes de guardar)
func (b *Bicicleta) SyncToDB() {
	if b.Marca != "" && b.FrameModel == nil {
		b.FrameModel = &b.Marca
	}
	if b.Modelo != "" && b.FrameModel == nil {
		b.FrameModel = &b.Modelo
	}
	if b.Anio > 0 && b.FrameYear == nil {
		b.FrameYear = &b.Anio
	}
	if b.Talle != "" && b.FrameSizeRaw == nil {
		b.FrameSizeRaw = &b.Talle
	}
	if b.Color != "" && b.PrimaryColorCustom == nil {
		b.PrimaryColorCustom = &b.Color
	}
}

// SyncFromDB actualiza los alias desde los campos de DB (después de leer)
func (b *Bicicleta) SyncFromDB() {
	if b.FrameModel != nil {
		b.Marca = *b.FrameModel
		b.Modelo = *b.FrameModel
	}
	if b.FrameYear != nil {
		b.Anio = *b.FrameYear
	}
	if b.FrameSizeRaw != nil {
		b.Talle = *b.FrameSizeRaw
	}
	if b.PrimaryColorCustom != nil {
		b.Color = *b.PrimaryColorCustom
	}
}
