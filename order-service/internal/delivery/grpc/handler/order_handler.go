package handler

import (
	"context"

	"order-service/internal/delivery/grpc/proto"
	"order-service/internal/domain/entities"
	"order-service/internal/domain/repositories"
	"order-service/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderHandler struct {
	proto.UnimplementedOrderServiceServer
	orderUseCase *usecase.OrderUseCase
}

func NewOrderHandler(orderUseCase *usecase.OrderUseCase) *OrderHandler {
	return &OrderHandler{
		orderUseCase: orderUseCase,
	}
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *proto.CreateOrderRequest) (*proto.CreateOrderResponse, error) {
	// Конвертация protobuf -> domain entities
	items := make([]entities.Item, len(req.Items))
	for i, item := range req.Items {
		items[i] = entities.Item{
			ProductID: item.ProductId,
			Quantity:  int(item.Quantity),
			Price:     item.Price,
		}
	}

	order, err := h.orderUseCase.CreateOrder(ctx, req.UserId, items)
	if err != nil {
		return nil, h.mapErrorToStatus(err)
	}

	protoOrder := h.domainToProto(order)
	return &proto.CreateOrderResponse{Order: protoOrder}, nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *proto.GetOrderRequest) (*proto.GetOrderResponse, error) {
	order, err := h.orderUseCase.GetOrder(ctx, req.OrderId)
	if err != nil {
		return nil, h.mapErrorToStatus(err)
	}

	protoOrder := h.domainToProto(order)
	return &proto.GetOrderResponse{Order: protoOrder}, nil
}

func (h *OrderHandler) UpdateOrderStatus(ctx context.Context, req *proto.UpdateOrderStatusRequest) (*proto.UpdateOrderStatusResponse, error) {
	order, err := h.orderUseCase.UpdateOrderStatus(ctx, req.OrderId, req.Status)
	if err != nil {
		return nil, h.mapErrorToStatus(err)
	}

	protoOrder := h.domainToProto(order)
	return &proto.UpdateOrderStatusResponse{Order: protoOrder}, nil
}

func (h *OrderHandler) domainToProto(order *entities.Order) *proto.Order {
	protoItems := make([]*proto.Item, len(order.Items))
	for i, item := range order.Items {
		protoItems[i] = &proto.Item{
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
			Price:     item.Price,
		}
	}

	return &proto.Order{
		OrderId:     order.OrderID,
		UserId:      order.UserID,
		Items:       protoItems,
		TotalAmount: order.TotalAmount,
		Status:      order.Status,
		CreatedAt:   timestamppb.New(order.CreatedAt),
	}
}

func (h *OrderHandler) mapErrorToStatus(err error) error {
	switch err {
	case usecase.ErrInvalidUserID, usecase.ErrInvalidOrderID, usecase.ErrEmptyItems,
		usecase.ErrInvalidItem, usecase.ErrInvalidStatus:
		return status.Error(codes.InvalidArgument, err.Error())
	case repositories.ErrOrderNotFound:
		return status.Error(codes.NotFound, err.Error())
	case repositories.ErrOrderAlreadyExists:
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
