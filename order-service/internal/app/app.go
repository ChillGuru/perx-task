package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

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

func Run() error {
	logger := logger.NewLogger()

	logger.Info("Environment variables:")
	logger.Info("GRPC_PORT=" + os.Getenv("GRPC_PORT"))
	logger.Info("MONGO_URI=" + os.Getenv("MONGO_URI"))
	logger.Info("MONGO_DB=" + os.Getenv("MONGO_DB"))
	logger.Info("NATS_URL=" + os.Getenv("NATS_URL"))

	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	mongoDB := getEnv("MONGO_DB", "orderdb")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	grpcPort := getEnv("GRPC_PORT", "50051")

	orderRepo, err := mongodb.NewOrderRepositoryMongo(mongoURI, mongoDB, logger)
	if err != nil {
		logger.Error("Failed to connect to MongoDB", "error", err)
		log.Fatal(err)
	}
	defer orderRepo.Close()
	logger.Info("Connected to MongoDB", "uri", mongoURI, "db", mongoDB)

	var natsPublisher usecase.NatsPublisher
	if natsURL != "" {
		publisher, err := connectToNATSWithRetry(natsURL, logger, 3, 2*time.Second)
		if err != nil {
			logger.Warn("Failed to connect to NATS, continuing without event publishing",
				"error", err,
				"url", natsURL)
			natsPublisher = &noopNatsPublisher{}
		} else {
			defer publisher.Close()
			natsPublisher = publisher
			logger.Info("Connected to NATS and event publishing is enabled")
		}
	} else {
		logger.Info("NATS_URL not set, event publishing disabled")
		natsPublisher = &noopNatsPublisher{}
	}

	orderUseCase := usecase.NewOrderUseCase(orderRepo, natsPublisher)

	orderHandler := handler.NewOrderHandler(orderUseCase)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(logger)),
	)

	proto.RegisterOrderServiceServer(grpcServer, orderHandler)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	logger.Info("Starting gRPC server on port 50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	return nil
}

func connectToNATSWithRetry(url string, logger *logger.Logger, maxRetries int, delay time.Duration) (usecase.NatsPublisher, error) {
	for i := 0; i < maxRetries; i++ {
		publisher, err := nats.NewNatsPublisher(url)
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

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func loggingInterceptor(logger *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Info("gRPC method called", "method", info.FullMethod)
		resp, err := handler(ctx, req)
		if err != nil {
			logger.Error("gRPC method failed", "method", info.FullMethod, "error", err)
		} else {
			logger.Info("gRPC method completed", "method", info.FullMethod)
		}
		return resp, err
	}
}
