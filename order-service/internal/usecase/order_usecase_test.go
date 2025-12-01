package usecase

import (
	"context"
	"testing"

	"order-service/internal/domain/entities"
	"order-service/internal/domain/repositories"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) Create(order *entities.Order) error {
	args := m.Called(order)
	return args.Error(0)
}

func (m *MockOrderRepository) GetByID(orderID string) (*entities.Order, error) {
	args := m.Called(orderID)
	return args.Get(0).(*entities.Order), args.Error(1)
}

func (m *MockOrderRepository) UpdateStatus(orderID, status string) error {
	args := m.Called(orderID, status)
	return args.Error(0)
}

func TestOrderUseCase_CreateOrder(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	useCase := NewOrderUseCase(mockRepo)
	ctx := context.Background()

	items := []entities.Item{
		{ProductID: "prod1", Quantity: 2, Price: 10.0},
		{ProductID: "prod2", Quantity: 1, Price: 5.0},
	}

	mockRepo.On("Create", mock.AnythingOfType("*entities.Order")).
		Return(nil).
		Run(func(args mock.Arguments) {
			order := args.Get(0).(*entities.Order)
			assert.Equal(t, "PENDING", order.Status)
			assert.Equal(t, 25.0, order.TotalAmount)
		})

	order, err := useCase.CreateOrder(ctx, "user123", items)

	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, "user123", order.UserID)
	assert.Equal(t, "PENDING", order.Status)
	assert.Equal(t, 25.0, order.TotalAmount)
	assert.Len(t, order.Items, 2)

	mockRepo.AssertExpectations(t)
}

func TestOrderUseCase_CreateOrder_InvalidInput(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	useCase := NewOrderUseCase(mockRepo)
	ctx := context.Background()

	tests := []struct {
		name    string
		userID  string
		items   []entities.Item
		wantErr string
	}{
		{
			name:    "empty user id",
			userID:  "",
			items:   []entities.Item{{ProductID: "prod1", Quantity: 1, Price: 10.0}},
			wantErr: "invalid user ID",
		},
		{
			name:    "empty items",
			userID:  "user123",
			items:   []entities.Item{},
			wantErr: "items list cannot be empty",
		},
		{
			name:    "invalid quantity",
			userID:  "user123",
			items:   []entities.Item{{ProductID: "prod1", Quantity: 0, Price: 10.0}},
			wantErr: "invalid item: item 0 has invalid quantity",
		},
		{
			name:    "invalid price",
			userID:  "user123",
			items:   []entities.Item{{ProductID: "prod1", Quantity: 1, Price: -10.0}},
			wantErr: "invalid item: item 0 has invalid price",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := useCase.CreateOrder(ctx, tt.userID, tt.items)
			assert.Error(t, err)
			assert.Nil(t, order)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestOrderUseCase_GetOrder(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	useCase := NewOrderUseCase(mockRepo)
	ctx := context.Background()

	expectedOrder := &entities.Order{
		OrderID: "test-order",
		UserID:  "user123",
		Status:  "PENDING",
	}

	mockRepo.On("GetByID", "test-order").Return(expectedOrder, nil)

	order, err := useCase.GetOrder(ctx, "test-order")

	assert.NoError(t, err)
	assert.Equal(t, expectedOrder, order)
	mockRepo.AssertExpectations(t)
}

func TestOrderUseCase_UpdateOrderStatus(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	useCase := NewOrderUseCase(mockRepo)
	ctx := context.Background()

	existingOrder := &entities.Order{
		OrderID: "test-order",
		UserID:  "user123",
		Status:  "PENDING",
	}

	mockRepo.On("GetByID", "test-order").Return(existingOrder, nil)
	mockRepo.On("UpdateStatus", "test-order", "PAID").Return(nil)

	order, err := useCase.UpdateOrderStatus(ctx, "test-order", "PAID")

	assert.NoError(t, err)
	assert.Equal(t, "PAID", order.Status)
	mockRepo.AssertExpectations(t)
}

func TestOrderUseCase_UpdateOrderStatus_Invalid(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	useCase := NewOrderUseCase(mockRepo)
	ctx := context.Background()

	_, err := useCase.UpdateOrderStatus(ctx, "test-order", "INVALID_STATUS")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid order status")

	mockRepo.On("GetByID", "non-existent").Return((*entities.Order)(nil), repositories.ErrOrderNotFound)
	_, err = useCase.UpdateOrderStatus(ctx, "non-existent", "PAID")
	assert.Error(t, err)
}
