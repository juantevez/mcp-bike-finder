// internal/domain/ports.go
package domain

import "context"

// ===========================================
// Ports de servicios externos (driven ports)
// ===========================================

// ImageStorage define el contrato para descargar imágenes desde almacenamiento remoto.
type ImageStorage interface {
	Download(ctx context.Context, url string) ([]byte, error)
}
