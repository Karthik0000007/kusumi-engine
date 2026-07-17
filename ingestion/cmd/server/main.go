// Package main is the entry point for the Kasumi Engine ingestion service.
//
// The ingestion service handles real-time clickstream and dwell-time event
// intake via gRPC, applies APPI-compliant edge anonymization (salted hashing,
// differential privacy noise), aggregates events into windowed features, and
// writes results to the Redis feature store.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/kasumi-engine/ingestion/api/kasumi/v1"
	"github.com/kasumi-engine/ingestion/internal/config"
	"github.com/kasumi-engine/ingestion/internal/logger"
	"github.com/kasumi-engine/ingestion/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load configuration
	configPath := "config/default.yaml"
	if envPath := os.Getenv("KASUMI_CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New(cfg.Ingestion.LogLevel, "ingestion")
	log.Info().Str("config_path", configPath).Msg("configuration loaded successfully")

	// Set up gRPC listener
	addr := fmt.Sprintf("%s:%d", cfg.Ingestion.Host, cfg.Ingestion.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen")
	}

	// Set up server options with interceptors
	interceptors := server.NewInterceptors(&cfg.Ingestion, log)

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptors.UnaryInterceptor()),
		grpc.StreamInterceptor(interceptors.StreamInterceptor()),
		grpc.MaxConcurrentStreams(uint32(cfg.Ingestion.MaxConcurrentStreams)),
	}

	if cfg.Ingestion.TLSEnabled {
		creds, err := credentials.NewServerTLSFromFile(cfg.Ingestion.TLSCertPath, cfg.Ingestion.TLSKeyPath)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to setup TLS")
		}
		opts = append(opts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(opts...)

	// Register services
	ingestionServer := server.NewIngestionServer(&cfg.Ingestion, log)
	
	// Start background event processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ingestionServer.Start(ctx)

	pb.RegisterIngestionServiceServer(grpcServer, ingestionServer)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Register reflection service on gRPC server for debugging/grpcurl
	reflection.Register(grpcServer)

	// Start server
	go func() {
		log.Info().Msgf("Starting gRPC server on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("failed to serve gRPC server")
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	select {
	case sig := <-sigCh:
		log.Info().Str("signal", sig.String()).Msg("received shutdown signal")
		cancel() // Stop the background event processor
	}

	log.Info().Msg("shutting down gRPC server gracefully...")
	healthServer.Shutdown()
	grpcServer.GracefulStop()
	log.Info().Msg("ingestion service stopped")
}
