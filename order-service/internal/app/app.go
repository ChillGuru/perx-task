package app

import (
	"context"
	"log"
	"net"
	"os"

	"order-service/internal/delivery/grpc/handler"
	"order-service/internal/delivery/grpc/proto"
	"order-service/internal/infrastructure/logger"
	"order-service/internal/infrastructure/mongodb"
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

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	mongoDB := os.Getenv("MONGO_DB")
	if mongoDB == "" {
		mongoDB = "orderdb"
	}

	orderRepo, err := mongodb.NewOrderRepositoryMongo(mongoURI, mongoDB, logger)
	if err != nil {
		logger.Error("Failed to connect to MongoDB", "error", err)
		log.Fatal(err)
	}
	defer orderRepo.Close()
	logger.Info("Connected to MongoDB", "uri", mongoURI, "db", mongoDB)

	orderUseCase := usecase.NewOrderUseCase(orderRepo)

	orderHandler := handler.NewOrderHandler(orderUseCase)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(logger)),
	)

	proto.RegisterOrderServiceServer(grpcServer, orderHandler)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	logger.Info("Starting gRPC server on port 50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	return nil
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
