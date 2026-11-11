package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"nagomi-receipts/internal/config"
	pb "nagomi-receipts/internal/gen/nagomi/v1"
	"nagomi-receipts/internal/server"
	"nagomi-receipts/internal/vision"
)

func main() {
	cfg := config.Load()

	formatter := log.TextFormatter
	if cfg.LogFormat == "json" {
		formatter = log.JSONFormatter
	}
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Level:           cfg.LogLevel,
		Formatter:       formatter,
	})

	model := vision.New(cfg.BaseURL, cfg.APIKey, cfg.Model)

	lis, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		logger.Fatal("failed to listen", "err", err)
	}

	srv := grpc.NewServer()
	pb.RegisterReceiptOCRServiceServer(srv, server.New(model, logger))

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	reflection.Register(srv)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		logger.Info("shutting down")
		healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		srv.GracefulStop()
	}()

	logger.Info("server started", "addr", cfg.ListenAddress, "api", cfg.BaseURL, "model", cfg.Model)
	if err := srv.Serve(lis); err != nil {
		logger.Fatal("serve failed", "err", err)
	}
}
