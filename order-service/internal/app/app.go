package app

import (
	"context"
	"log"
	"net"

	"order-service/internal/delivery/grpc/handler"
	"order-service/internal/delivery/grpc/proto"
	"order-service/internal/infrastructure/logger"
	"order-service/internal/infrastructure/memory"
	"order-service/internal/usecase"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func Run() error {
	logger := logger.NewLogger()

	orderRepo := memory.NewOrderRepositoryMemory()

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
