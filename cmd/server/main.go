package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/juantevez/mcp-bike-finder/internal/config"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/database"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/ocr"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/s3"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/scraper"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/vision"
	"github.com/juantevez/mcp-bike-finder/internal/mcp"
	"github.com/juantevez/mcp-bike-finder/internal/service"
)

func main() {
	// ===========================================
	// 1. Parsear flags de línea de comandos
	// ===========================================
	healthCheck := flag.Bool("health", false, "Ejecutar health check")
	flag.Parse()

	if *healthCheck {
		fmt.Println("OK")
		os.Exit(0)
	}

	// ===========================================
	// 2. Cargar configuración
	// ===========================================
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error cargando configuración: %v", err)
	}

	// ===========================================
	// 3. Inicializar contexto con cancelación
	// ===========================================
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Manejar señales de interrupción
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Recibida señal de interrupción, cerrando...")
		cancel()
	}()

	// ===========================================
	// 4. Inicializar Infraestructura
	// ===========================================

	// Conexión a PostgreSQL
	db, err := database.NewPostgresDB(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("Error conectando a PostgreSQL: %v", err)
	}
	defer db.Close()

	// Repositorios
	biciRepo := database.NewBicycleRepository(db.Pool(), db.BikeSchema())
	refRepo := database.NewReferenceRepository(db.Pool(), db.BikeSchema())
	// Usar mock en lugar de implementación real
	busquedaRepo := database.NewBusquedaRepositoryMock()
	//busquedaRepo := database.NewBusquedaRepository(db.Pool(), db.BikeSchema())
	//fotoRepo := database.NewPhotoRepository(db.Pool(), db.BikeSchema())
	//userRepo := database.NewUserRepository(db.Pool(), db.AuthSchema())

	// Cliente S3
	s3Client, err := s3.NewClient(ctx, cfg.AWS)
	if err != nil {
		log.Fatalf("Error inicializando cliente S3: %v", err)
	}

	// Cliente OCR (AWS Textract)
	var ocrClient *ocr.Client
	if cfg.OCR.TextractEnabled {
		// ✅ Pasar cfg.AWS (de tipo config.AWSConfig), NO cfg.OCR
		ocrClient, err = ocr.NewClient(ctx, cfg.AWS)
		if err != nil {
			log.Printf("⚠️ OCR Textract no disponible: %v (usando mock)", err)
		}
	}

	// Cliente Vision (AWS Rekognition)
	visionConfig := config.AWSConfig{ // ✅ Usar directamente config.AWSConfig
		Region:          cfg.AWS.Region,
		AccessKeyID:     cfg.AWS.AccessKeyID,
		SecretAccessKey: cfg.AWS.SecretAccessKey,
	}

	visionClient, err := vision.NewClient(ctx, visionConfig)
	if err != nil {
		log.Printf("⚠️ Vision no disponible: %v (usando mock)", err)
	}

	// ===========================================
	// 5. Inicializar Servicios (Capa de Aplicación)
	// ===========================================
	extractorSvc := service.NewExtractorService(s3Client, ocrClient, visionClient)
	busquedaSvc := service.NewBusquedaService(scraper.ScraperConfig(cfg.Scraper))
	//bicicletaSvc := service.NewBicicletaService(biciRepo, refRepo, extractorSvc, busquedaSvc)

	// ✅ Llamar con los parámetros en el orden correcto:
	bicicletaSvc := service.NewBicicletaService(
		biciRepo,     // 1: domain.BicicletaRepository // FALLA ACA
		busquedaRepo, // 2: domain.BusquedaRepository
		refRepo,      // 3: domain.ReferenceRepository ✅  // FALLA ACA
		extractorSvc, // 4: *ExtractorService
		busquedaSvc,  // 5: *BusquedaService
	)

	// ===========================================
	// 6. Inicializar Servidor MCP
	// ===========================================
	mcpServer := mcp.NewServer(
		cfg.MCP,
		bicicletaSvc,
		extractorSvc,
		busquedaSvc,
		refRepo, // ← domain.ReferenceRepository para handlers de marcas/tipos
	)

	// ===========================================
	// 7. Iniciar Servidor
	// ===========================================
	log.Printf("🚀 Iniciando servidor MCP: %s v%s", cfg.MCP.Name, cfg.MCP.Version)
	log.Printf("📡 Transporte: %s", cfg.MCP.Transport)

	if err := mcpServer.Run(ctx); err != nil {
		log.Fatalf("❌ Error del servidor MCP: %v", err)
	}
}
