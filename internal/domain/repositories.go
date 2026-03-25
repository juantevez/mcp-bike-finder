// internal/domain/repositories.go
package domain

import (
	"context"
)

// ===========================================
// BicicletaRepository
// ===========================================

type BicicletaRepository interface {
	Guardar(ctx context.Context, bici *Bicicleta) error
	ObtenerPorID(ctx context.Context, id string) (*Bicicleta, error)
	ObtenerPorMarcaModelo(ctx context.Context, marca, modelo string) ([]*Bicicleta, error)
	ObtenerPorImagenURL(ctx context.Context, imagenS3URL string) (*Bicicleta, error)
	ListarActivas(ctx context.Context, limite int) ([]*Bicicleta, error)
	Actualizar(ctx context.Context, bici *Bicicleta) error
	Eliminar(ctx context.Context, id string) error
}

// ===========================================
// ReferenceRepository ⭐ (¡ESTE FALTABA!)
// ===========================================

type ReferenceRepository interface {
	// Brands
	ObtenerTodasLasMarcas(ctx context.Context, soloActivas bool) ([]*Brand, error)
	BuscarMarcaPorNombre(ctx context.Context, nombre string) (*Brand, error)
	ObtenerMarcaPorID(ctx context.Context, id int) (*Brand, error)

	// BikeTypes
	ObtenerTodosLosTipos(ctx context.Context) ([]*BikeType, error)
	ObtenerTipoPorID(ctx context.Context, id int) (*BikeType, error)

	// StandardColors
	ObtenerColoresPorFamilia(ctx context.Context, familia *string) ([]*StandardColor, error)
	ObtenerColorPorID(ctx context.Context, id int) (*StandardColor, error)

	// BikeCatalog
	BuscarEnCatalogo(ctx context.Context, marca, modelo *string, anio *int, limite int) ([]*BikeCatalog, error)
	ObtenerCatalogoPorID(ctx context.Context, id int) (*BikeCatalog, error)
}

// ===========================================
// BusquedaRepository
// ===========================================

type BusquedaRepository interface {
	GuardarHistorial(ctx context.Context, historial *BusquedaHistorial) error
	ObtenerHistorial(ctx context.Context, usuarioID string, limite int) ([]*BusquedaHistorial, error)
}

// ===========================================
// AlertaRepository
// ===========================================

type AlertaRepository interface {
	Guardar(ctx context.Context, alerta *Alerta) error
	ObtenerPorBicicleta(ctx context.Context, bicicletaID string) ([]*Alerta, error)
	ObtenerPorUsuario(ctx context.Context, usuarioID string, status *string) ([]*Alerta, error)
	ActualizarStatus(ctx context.Context, id string, status string) error
	ExisteParaBicicletaYURL(ctx context.Context, bicicletaID string, url string) (bool, error)
}

// ===========================================
// PhotoRepository (para bicycle_photos)
// ===========================================

type PhotoRepository interface {
	ObtenerFotosPorBicicleta(ctx context.Context, bicycleID string, photoType *string, onlyPrimary bool) ([]*BicyclePhoto, error)
	GuardarFoto(ctx context.Context, foto *BicyclePhoto) error
	ObtenerFotoPorID(ctx context.Context, id string) (*BicyclePhoto, error)
}

// ===========================================
// UserRepository (para auth.users)
// ===========================================

type UserRepository interface {
	ObtenerPorID(ctx context.Context, id string) (*User, error)
	ObtenerPorEmail(ctx context.Context, email string) (*User, error)
	ContarBicicletasPorUsuario(ctx context.Context, userID string, status *string) (int, error)
}
