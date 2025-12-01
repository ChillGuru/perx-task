package repositories

import "order-service/internal/domain/entities"

type OrderRepository interface {
	Create(order *entities.Order) error
	GetByID(orderID string) (*entities.Order, error)
	UpdateStatus(orderID, status string) error
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
