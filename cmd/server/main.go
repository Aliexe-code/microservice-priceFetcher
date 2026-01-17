package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aliexe/ms-priceFetcher/internal/config"
	"github.com/aliexe/ms-priceFetcher/internal/server"
	"github.com/aliexe/ms-priceFetcher/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or error loading: %v", err)
	}

	// Load configuration
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	svc := service.NewLoggingService(service.NewPriceService())
	alertSvc := service.NewAlertService(service.NewPriceService())

	log.Printf("Starting Price Fetcher Service...")
	log.Printf("JSON API: http://localhost%s", cfg.JSONAddr)
	log.Printf("gRPC API: localhost%s", cfg.GRPCAddr)

	// Create servers
	httpServer := server.NewJSONAPIServer(cfg.JSONAddr, svc, alertSvc)
	grpcServer, err := server.MakeGRPCServer(cfg.GRPCAddr, svc)
	if err != nil {
		log.Fatalf("Failed to create gRPC server: %v", err)
	}

	// Start alert checker in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go alertSvc.StartAlertChecker(ctx, 30*time.Second)

	// Channel to listen for shutdown signals
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	// Start servers in goroutines
	httpErrChan := make(chan error, 1)
	go func() {
		if err := httpServer.Run(); err != nil && err != http.ErrServerClosed {
			httpErrChan <- err
		}
	}()

	grpcErrChan := make(chan error, 1)
	go func() {
		if err := grpcServer.Run(); err != nil {
			grpcErrChan <- err
		}
	}()

	// Wait for shutdown signal or server errors
	select {
	case err := <-httpErrChan:
		log.Fatalf("HTTP server error: %v", err)
	case err := <-grpcErrChan:
		log.Fatalf("gRPC server error: %v", err)
	case sig := <-shutdownChan:
		log.Printf("Received signal %v, shutting down gracefully...", sig)

		// Cancel alert checker
		cancel()

		// Create context for shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		// Shutdown HTTP server
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}

		// Shutdown gRPC server
		grpcServer.Stop()

		log.Println("Servers stopped gracefully")
	}
}
