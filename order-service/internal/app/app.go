package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-service/internal/config"
	"order-service/internal/delivery/grpc/handler"
	"order-service/internal/delivery/grpc/proto"
	"order-service/internal/domain/entities"
	"order-service/internal/infrastructure/logger"
	"order-service/internal/infrastructure/mongodb"
	"order-service/internal/infrastructure/nats"
	"order-service/internal/usecase"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type App struct {
	cfg    *config.Config
	logger *logger.Logger
}

func New(cfg *config.Config) *App {
	return &App{
		cfg:    cfg,
		logger: logger.NewLogger(),
	}
}

func (a *App) Run() error {
	a.logger.Info("Starting order-service")

	orderRepo, err := a.initMongoDB()
	if err != nil {
		return err
	}
	defer orderRepo.Close()

	natsPublisher := a.initNATS()
	if closer, ok := natsPublisher.(interface{ Close() }); ok {
		defer closer.Close()
	}

	orderUseCase := usecase.NewOrderUseCase(orderRepo, natsPublisher)

	grpcServer, lis, err := a.initGRPCServer(orderUseCase)
	if err != nil {
		return err
	}

	return a.runServerWithGracefulShutdown(grpcServer, lis)
}

func (a *App) initMongoDB() (*mongodb.OrderRepositoryMongo, error) {
	a.logger.Info("Connecting to MongoDB", "uri", a.cfg.Mongo.URI, "db", a.cfg.Mongo.DB)

	orderRepo, err := mongodb.NewOrderRepositoryMongo(a.cfg.Mongo.URI, a.cfg.Mongo.DB, a.logger)
	if err != nil {
		a.logger.Error("Failed to connect to MongoDB", "error", err)
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	a.logger.Info("Connected to MongoDB successfully")
	return orderRepo, nil
}

func (a *App) initNATS() usecase.NatsPublisher {
	if a.cfg.NATS.URL == "" {
		a.logger.Info("NATS URL not set, event publishing disabled")
		return &noopNatsPublisher{}
	}

	publisher, err := connectToNATSWithRetry(a.cfg.NATS.URL, a.logger, 3, 2*time.Second)
	if err != nil {
		a.logger.Warn("Failed to connect to NATS, continuing without event publishing",
			"error", err,
			"url", a.cfg.NATS.URL)
		return &noopNatsPublisher{}
	}

	a.logger.Info("Connected to NATS successfully")
	return publisher
}

func (a *App) initGRPCServer(orderUseCase *usecase.OrderUseCase) (*grpc.Server, net.Listener, error) {
	orderHandler := handler.NewOrderHandler(orderUseCase)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(a.loggingInterceptor()),
	)

	proto.RegisterOrderServiceServer(grpcServer, orderHandler)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+a.cfg.GRPC.Port)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen on port %s: %w", a.cfg.GRPC.Port, err)
	}

	return grpcServer, lis, nil
}

func (a *App) runServerWithGracefulShutdown(grpcServer *grpc.Server, lis net.Listener) error {
	serverErrors := make(chan error, 1)

	go func() {
		a.logger.Info("Starting gRPC server", "port", a.cfg.GRPC.Port)
		serverErrors <- grpcServer.Serve(lis)
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		a.logger.Info("Received shutdown signal, starting graceful shutdown", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		shutdownComplete := make(chan struct{})

		go func() {
			a.logger.Info("Stopping gRPC server gracefully")
			grpcServer.GracefulStop()
			close(shutdownComplete)
		}()

		select {
		case <-shutdownComplete:
			a.logger.Info("Graceful shutdown completed")
		case <-ctx.Done():
			a.logger.Warn("Graceful shutdown timeout, forcing stop")
			grpcServer.Stop()
		}

		return nil
	}
}

func connectToNATSWithRetry(url string, logger *logger.Logger, maxRetries int, delay time.Duration) (usecase.NatsPublisher, error) {
	for i := 0; i < maxRetries; i++ {
		publisher, err := nats.NewNatsPublisher(url, logger)
		if err == nil {
			return publisher, nil
		}

		logger.Warn("Failed to connect to NATS, retrying...",
			"attempt", i+1,
			"max_retries", maxRetries,
			"error", err)

		if i < maxRetries-1 {
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("failed to connect to NATS after %d attempts", maxRetries)
}

type noopNatsPublisher struct{}

func (n *noopNatsPublisher) PublishOrderCreated(ctx context.Context, order *entities.Order) error {
	return nil
}

func (n *noopNatsPublisher) Close() {
}

func (a *App) loggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		a.logger.Info("gRPC method called", "method", info.FullMethod)
		resp, err := handler(ctx, req)
		if err != nil {
			a.logger.Error("gRPC method failed", "method", info.FullMethod, "error", err)
		} else {
			a.logger.Info("gRPC method completed", "method", info.FullMethod)
		}
		return resp, err
	}
}
