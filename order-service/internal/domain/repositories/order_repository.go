package repositories

import (
	"context"
	"order-service/internal/domain/entities"
)

type OrderRepository interface {
	Create(ctx context.Context, order *entities.Order) error
	GetByID(ctx context.Context, orderID string) (*entities.Order, error)
	UpdateStatus(ctx context.Context, orderID, status string) error
}

var (
	ErrOrderNotFound      = &RepositoryError{"order not found"}
	ErrOrderAlreadyExists = &RepositoryError{"order already exists"}
)

type RepositoryError struct {
	message string
}

func (e *RepositoryError) Error() string {
	return e.message
}
