package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/juantevez/mcp-bike-finder/internal/config"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/database"
	"github.com/juantevez/mcp-bike-finder/internal/infrastructure/scraper"
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
	busquedaRepo := database.NewBusquedaRepositoryMock()
	alertaRepo := database.NewAlertaRepositoryMock()

	// ===========================================
	// 5. Inicializar Servicios (Capa de Aplicación)
	// ===========================================
	extractorSvc := service.NewExtractorService(biciRepo)
	busquedaSvc := service.NewBusquedaService(scraper.ScraperConfig(cfg.Scraper))
	alertaSvc := service.NewAlertaService(alertaRepo)
	schedulerSvc := service.NewSchedulerService(
		biciRepo,
		busquedaSvc,
		alertaSvc,
		time.Duration(cfg.Scheduler.IntervaloHoras)*time.Hour,
		cfg.Scheduler.LimiteBicis,
	)
	bicicletaSvc := service.NewBicicletaService(
		biciRepo,
		busquedaRepo,
		refRepo,
		extractorSvc,
		busquedaSvc,
	)

	go schedulerSvc.Start(ctx)

	// ===========================================
	// 6. Inicializar Servidor MCP
	// ===========================================
	mcpServer := mcp.NewServer(
		cfg.MCP,
		bicicletaSvc,
		extractorSvc,
		busquedaSvc,
		alertaSvc,
		refRepo,
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
