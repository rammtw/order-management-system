package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/rammtw/order-management-system/apps/inventory/internal/config"
	"github.com/rammtw/order-management-system/apps/inventory/internal/handler"
	invkafka "github.com/rammtw/order-management-system/apps/inventory/internal/kafka"
	"github.com/rammtw/order-management-system/apps/inventory/internal/repository"
	"github.com/rammtw/order-management-system/apps/inventory/internal/service"
	pb "github.com/rammtw/order-management-system/gen/inventory/v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	repo := repository.NewProductRepository(pool)
	svc := service.NewInventoryService(repo)
	h := handler.NewInventoryHandler(svc)

	consumer := invkafka.NewConsumer(cfg.KafkaBroker, cfg.KafkaGroupID, svc)
	consumer.Start(ctx)
	defer consumer.Close()

	grpcServer := grpc.NewServer()
	pb.RegisterInventoryServiceServer(grpcServer, h)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("inventory service started", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	grpcServer.GracefulStop()
}
