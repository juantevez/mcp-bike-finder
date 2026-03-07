// internal/infrastructure/s3/client.go
package s3

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/juantevez/mcp-bike-finder/internal/config"
)

// ===========================================
// Client - Cliente para AWS S3
// ===========================================

type Client struct {
	config Config
	mock   bool // true = modo mock para desarrollo local
}

type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string // Para S3 compatible (MinIO, LocalStack)
	UsePathStyle    bool   // Para S3 compatible
}

// NewClient crea una instancia del cliente S3
func NewClient(ctx context.Context, cfg config.AWSConfig) (*Client, error) {

	// Para desarrollo local: usar modo mock si no hay credenciales reales
	mockMode := cfg.AccessKeyID == "" || cfg.AccessKeyID == "tu_access_key"

	if mockMode {
		log.Println("🪣 [S3] Inicializando cliente en modo MOCK (sin credenciales AWS reales)")
	} else {
		log.Printf("🪣 [S3] Inicializando cliente real para región: %s", cfg.Region)
	}

	return &Client{
		config: Config{
			Region:          cfg.Region,
			AccessKeyID:     cfg.AccessKeyID,
			SecretAccessKey: cfg.SecretAccessKey,
			Bucket:          cfg.S3Bucket,
		},
		mock: mockMode,
	}, nil
}

// ===========================================
// Método Principal: Download
// ===========================================

// Download descarga una imagen desde S3 dado una URL o S3 path
// Soporta formatos:
//   - "s3://bucket/key/path.jpg"
//   - "https://bucket.s3.region.amazonaws.com/key/path.jpg"
//   - "key/path.jpg" (asume bucket configurado)
func (c *Client) Download(ctx context.Context, s3URL string) ([]byte, error) {

	if c.mock {
		return c.mockDownload(s3URL)
	}

	return c.downloadFromAWS(ctx, s3URL)
}

// ===========================================
// Implementación Mock (para desarrollo)
// ===========================================

func (c *Client) mockDownload(s3URL string) ([]byte, error) {
	log.Printf("🪣 [S3-MOCK] Simulando descarga de: %s", s3URL)

	// Opción 1: Si la URL apunta a un archivo local para testing
	localPath := extractLocalPath(s3URL)
	if localPath != "" && fileExists(localPath) {
		log.Printf("📁 [S3-MOCK] Leyendo archivo local: %s", localPath)
		return os.ReadFile(localPath)
	}

	// Opción 2: Retornar una imagen PNG mínima válida (1x1 pixel transparente)
	// Esto permite que el flujo de OCR/Vision continúe sin errores
	log.Println("🖼️ [S3-MOCK] Retornando imagen de prueba mínima")

	// PNG 1x1 pixel transparente (en base64 decodificado)
	// Este es un PNG válido mínimo para que las librerías de imagen no fallen
	minimalPNG := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 pixels
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, // IEND chunk
		0x42, 0x60, 0x82,
	}

	return minimalPNG, nil
}

// extractLocalPath intenta extraer una ruta local de una URL S3 para testing
func extractLocalPath(s3URL string) string {
	// Si es una ruta absoluta local, usarla directamente
	if strings.HasPrefix(s3URL, "file://") {
		return strings.TrimPrefix(s3URL, "file://")
	}

	// Si parece una URL de S3 pero tenemos un archivo local con el mismo nombre para testing
	// Ej: "s3://my-bucket/test-bike.jpg" → "./testdata/test-bike.jpg"
	if strings.Contains(s3URL, ".jpg") || strings.Contains(s3URL, ".jpeg") || strings.Contains(s3URL, ".png") {
		parts := strings.Split(s3URL, "/")
		filename := parts[len(parts)-1]
		testPath := filepath.Join("testdata", filename)
		if fileExists(testPath) {
			return testPath
		}
	}

	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// ===========================================
// Implementación Real con AWS SDK v2
// ===========================================

func (c *Client) downloadFromAWS(ctx context.Context, s3URL string) ([]byte, error) {

	// 🚧 En producción, descomentar y configurar AWS SDK v2

	/*
		import (
			"github.com/aws/aws-sdk-go-v2/aws"
			"github.com/aws/aws-sdk-go-v2/config"
			"github.com/aws/aws-sdk-go-v2/service/s3"
		)

		// 1. Cargar configuración AWS
		awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(c.config.Region))
		if err != nil {
			return nil, fmt.Errorf("error cargando config AWS: %w", err)
		}

		// 2. Crear cliente S3
		s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if c.config.Endpoint != "" {
				o.BaseEndpoint = aws.String(c.config.Endpoint)
				o.UsePathStyle = c.config.UsePathStyle
			}
		})

		// 3. Parsear S3 URL para obtener bucket y key
		bucket, key, err := parseS3URL(s3URL, c.config.Bucket)
		if err != nil {
			return nil, fmt.Errorf("error parseando URL S3: %w", err)
		}

		// 4. Descargar objeto
		output, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return nil, fmt.Errorf("error descargando de S3: %w", err)
		}
		defer output.Body.Close()

		// 5. Leer todo el contenido
		return io.ReadAll(output.Body)
	*/

	// Por ahora, fallback a mock
	log.Println("⚠️ [S3] Implementación AWS no disponible, usando mock")
	return c.mockDownload(s3URL)
}

// parseS3URL extrae bucket y key de una URL S3
func parseS3URL(s3URL, defaultBucket string) (bucket, key string, err error) {

	// Formato: s3://bucket/key
	if strings.HasPrefix(s3URL, "s3://") {
		parts := strings.SplitN(strings.TrimPrefix(s3URL, "s3://"), "/", 2)
		if len(parts) < 2 {
			return "", "", fmt.Errorf("URL S3 inválida: %s", s3URL)
		}
		return parts[0], parts[1], nil
	}

	// Formato: https://bucket.s3.region.amazonaws.com/key
	if strings.Contains(s3URL, "s3.") && strings.Contains(s3URL, "amazonaws.com") {
		// Parseo simplificado
		// En producción usar url.Parse() y regex más robusto
		return defaultBucket, extractKeyFromHTTPS(s3URL), nil
	}

	// Formato: solo key (asumir bucket configurado)
	if defaultBucket != "" {
		return defaultBucket, strings.TrimPrefix(s3URL, "/"), nil
	}

	return "", "", fmt.Errorf("no se pudo parsear URL S3: %s", s3URL)
}

func extractKeyFromHTTPS(url string) string {
	// Simplificado: tomar todo después del dominio
	parts := strings.SplitN(url, ".amazonaws.com/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return url
}

// ===========================================
// Métodos Adicionales (Stubs)
// ===========================================

// Upload sube un archivo a S3 (stub para desarrollo)
func (c *Client) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {

	if c.mock {
		log.Printf("🪣 [S3-MOCK] Simulando upload: %s (%d bytes)", key, len(data))
		return fmt.Sprintf("s3://%s/%s", c.config.Bucket, key), nil
	}

	// 🚧 Implementación real con AWS SDK aquí
	return "", fmt.Errorf("upload no implementado en modo real")
}

// Exists verifica si un objeto existe en S3
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {

	if c.mock {
		// En mock, asumir que existe si no es una ruta imposible
		return !strings.Contains(key, "no-existe"), nil
	}

	// 🚧 Implementación real con HeadObject
	return false, fmt.Errorf("exists no implementado en modo real")
}

// GetURL genera una URL pública o pre-signed para un objeto
func (c *Client) GetURL(ctx context.Context, key string, expiresMinutes int) (string, error) {

	if c.mock {
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
			c.config.Bucket, c.config.Region, key), nil
	}

	// 🚧 Implementación real con presign
	return "", fmt.Errorf("GetURL no implementado en modo real")
}

// ===========================================
// Helpers
// ===========================================

// ParseS3Path helper para extraer bucket/key de varias formas
func ParseS3Path(input string) (bucket, key string) {
	input = strings.TrimSpace(input)

	// s3://bucket/key
	if strings.HasPrefix(input, "s3://") {
		input = strings.TrimPrefix(input, "s3://")
		parts := strings.SplitN(input, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return input, ""
	}

	// https://bucket.s3.region.amazonaws.com/key
	if strings.Contains(input, ".s3.") && strings.Contains(input, "amazonaws.com") {
		// Parseo básico
		return "", input // Simplificado
	}

	// Asumir que es solo el key
	return "", input
}
