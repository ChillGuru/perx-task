package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"order-service/internal/domain/entities"
	"order-service/internal/domain/repositories"

	"github.com/google/uuid"
)

type OrderUseCase struct {
	orderRepo repositories.OrderRepository
}

func NewOrderUseCase(orderRepo repositories.OrderRepository) *OrderUseCase {
	return &OrderUseCase{
		orderRepo: orderRepo,
	}
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, userID string, items []entities.Item) (*entities.Order, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}

	totalAmount := 0.0
	for i, item := range items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("%w: item %d has invalid quantity", ErrInvalidItem, i)
		}
		if item.Price < 0 {
			return nil, fmt.Errorf("%w: item %d has invalid price", ErrInvalidItem, i)
		}
		totalAmount += float64(item.Quantity) * item.Price
	}

	order := &entities.Order{
		OrderID:     uuid.New().String(),
		UserID:      userID,
		Items:       items,
		TotalAmount: totalAmount,
		Status:      "PENDING",
		CreatedAt:   time.Now(),
	}

	if err := uc.orderRepo.Create(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, orderID string) (*entities.Order, error) {
	if orderID == "" {
		return nil, ErrInvalidOrderID
	}

	order, err := uc.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

func (uc *OrderUseCase) UpdateOrderStatus(ctx context.Context, orderID, status string) (*entities.Order, error) {
	if orderID == "" {
		return nil, ErrInvalidOrderID
	}
	if !entities.ValidStatus(status) {
		return nil, ErrInvalidStatus
	}

	order, err := uc.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order for update: %w", err)
	}

	if err := uc.orderRepo.UpdateStatus(orderID, status); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	order.Status = status
	return order, nil
}

var (
	ErrInvalidUserID  = errors.New("invalid user ID")
	ErrInvalidOrderID = errors.New("invalid order ID")
	ErrEmptyItems     = errors.New("items list cannot be empty")
	ErrInvalidItem    = errors.New("invalid item")
	ErrInvalidStatus  = errors.New("invalid order status")
)
