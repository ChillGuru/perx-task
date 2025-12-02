package memory

import (
	"sync"

	"order-service/internal/domain/entities"
	"order-service/internal/domain/repositories"
)

type OrderRepositoryMemory struct {
	mu     sync.RWMutex
	orders map[string]*entities.Order
}

func NewOrderRepositoryMemory() *OrderRepositoryMemory {
	return &OrderRepositoryMemory{
		orders: make(map[string]*entities.Order),
	}
}

func (r *OrderRepositoryMemory) Create(order *entities.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orders[order.OrderID]; exists {
		return repositories.ErrOrderAlreadyExists
	}

	orderCopy := *order
	r.orders[order.OrderID] = &orderCopy
	return nil
}

func (r *OrderRepositoryMemory) GetByID(orderID string) (*entities.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, exists := r.orders[orderID]
	if !exists {
		return nil, repositories.ErrOrderNotFound
	}

	orderCopy := *order
	return &orderCopy, nil
}

func (r *OrderRepositoryMemory) UpdateStatus(orderID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, exists := r.orders[orderID]
	if !exists {
		return repositories.ErrOrderNotFound
	}

	order.Status = status
	return nil
}
