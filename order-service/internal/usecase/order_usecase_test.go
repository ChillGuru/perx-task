package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"

	"order-service/internal/domain/entities"
	"order-service/internal/domain/repositories"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) Create(ctx context.Context, order *entities.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) GetByID(ctx context.Context, orderID string) (*entities.Order, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Order), args.Error(1)
}

func (m *MockOrderRepository) UpdateStatus(ctx context.Context, orderID, status string) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

type MockNatsPublisher struct {
	mock.Mock
}

func (m *MockNatsPublisher) PublishOrderCreated(ctx context.Context, order *entities.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockNatsPublisher) Close() {
	m.Called()
}

func TestOrderUseCase_CreateOrder(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	mockNats := new(MockNatsPublisher)

	useCase := NewOrderUseCase(mockRepo, mockNats)
	ctx := context.Background()

	items := []entities.Item{
		{ProductID: "prod1", Quantity: 2, Price: 10.0},
		{ProductID: "prod2", Quantity: 1, Price: 5.0},
	}

	var wg sync.WaitGroup
	wg.Add(1)

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Order")).
		Return(nil).
		Run(func(args mock.Arguments) {
			order := args.Get(1).(*entities.Order)
			assert.Equal(t, "PENDING", order.Status)
			assert.Equal(t, 25.0, order.TotalAmount)
			assert.Equal(t, "user123", order.UserID)
			assert.Len(t, order.Items, 2)
		})

	mockNats.On("PublishOrderCreated", mock.Anything, mock.AnythingOfType("*entities.Order")).
		Return(nil).
		Run(func(args mock.Arguments) {
			wg.Done()
		})

	order, err := useCase.CreateOrder(ctx, "user123", items)

	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, "user123", order.UserID)
	assert.Equal(t, "PENDING", order.Status)
	assert.Equal(t, 25.0, order.TotalAmount)
	assert.Len(t, order.Items, 2)

	wg.Wait()

	mockRepo.AssertExpectations(t)
	mockNats.AssertExpectations(t)
}

func TestOrderUseCase_CreateOrder_NATSErrorNotFatal(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	mockNats := new(MockNatsPublisher)

	useCase := NewOrderUseCase(mockRepo, mockNats)
	ctx := context.Background()

	items := []entities.Item{
		{ProductID: "prod1", Quantity: 1, Price: 10.0},
	}

	var wg sync.WaitGroup
	wg.Add(1)

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Order")).
		Return(nil)

	mockNats.On("PublishOrderCreated", mock.Anything, mock.AnythingOfType("*entities.Order")).
		Return(errors.New("nats connection failed")).
		Run(func(args mock.Arguments) {
			wg.Done()
		})

	order, err := useCase.CreateOrder(ctx, "user123", items)

	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, "user123", order.UserID)
	assert.Equal(t, "PENDING", order.Status)

	wg.Wait()

	mockRepo.AssertExpectations(t)
	mockNats.AssertExpectations(t)
}

func TestOrderUseCase_CreateOrder_WithoutNATSPublisher(t *testing.T) {
	mockRepo := new(MockOrderRepository)

	useCase := NewOrderUseCase(mockRepo, nil)
	ctx := context.Background()

	items := []entities.Item{
		{ProductID: "prod1", Quantity: 1, Price: 10.0},
	}

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Order")).
		Return(nil)

	order, err := useCase.CreateOrder(ctx, "user123", items)

	assert.NoError(t, err)
	assert.NotNil(t, order)

	mockRepo.AssertExpectations(t)
}

func TestOrderUseCase_CreateOrder_InvalidInput(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	mockNats := new(MockNatsPublisher)

	useCase := NewOrderUseCase(mockRepo, mockNats)
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

			mockRepo.AssertNotCalled(t, "Create", mock.Anything)

			mockNats.AssertNotCalled(t, "PublishOrderCreated", mock.Anything, mock.Anything)
		})
	}
}

func TestOrderUseCase_GetOrder(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	mockNats := new(MockNatsPublisher)

	useCase := NewOrderUseCase(mockRepo, mockNats)
	ctx := context.Background()

	expectedOrder := &entities.Order{
		OrderID: "test-order",
		UserID:  "user123",
		Status:  "PENDING",
	}

	mockRepo.On("GetByID", mock.Anything, "test-order").Return(expectedOrder, nil)

	order, err := useCase.GetOrder(ctx, "test-order")

	assert.NoError(t, err)
	assert.Equal(t, expectedOrder, order)

	mockRepo.AssertExpectations(t)
	mockNats.AssertNotCalled(t, "PublishOrderCreated", mock.Anything, mock.Anything)
}

func TestOrderUseCase_GetOrder_NotFound(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	mockNats := new(MockNatsPublisher)

	useCase := NewOrderUseCase(mockRepo, mockNats)
	ctx := context.Background()

	mockRepo.On("GetByID", mock.Anything, "non-existent").Return((*entities.Order)(nil), repositories.ErrOrderNotFound)

	order, err := useCase.GetOrder(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "order not found")

	mockRepo.AssertExpectations(t)
	mockNats.AssertNotCalled(t, "PublishOrderCreated", mock.Anything, mock.Anything)
}

func TestOrderUseCase_UpdateOrderStatus(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	mockNats := new(MockNatsPublisher)

	useCase := NewOrderUseCase(mockRepo, mockNats)
	ctx := context.Background()

	existingOrder := &entities.Order{
		OrderID: "test-order",
		UserID:  "user123",
		Status:  "PENDING",
	}

	mockRepo.On("GetByID", mock.Anything, "test-order").Return(existingOrder, nil)
	mockRepo.On("UpdateStatus", mock.Anything, "test-order", "PAID").Return(nil)

	order, err := useCase.UpdateOrderStatus(ctx, "test-order", "PAID")

	assert.NoError(t, err)
	assert.Equal(t, "PAID", order.Status)

	mockRepo.AssertExpectations(t)
	mockNats.AssertNotCalled(t, "PublishOrderCreated", mock.Anything, mock.Anything)
}

func TestOrderUseCase_UpdateOrderStatus_InvalidStatus(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	mockNats := new(MockNatsPublisher)

	useCase := NewOrderUseCase(mockRepo, mockNats)
	ctx := context.Background()

	_, err := useCase.UpdateOrderStatus(ctx, "test-order", "INVALID_STATUS")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid order status")

	mockRepo.AssertNotCalled(t, "GetByID", mock.Anything)
	mockRepo.AssertNotCalled(t, "UpdateStatus", mock.Anything, mock.Anything)
	mockNats.AssertNotCalled(t, "PublishOrderCreated", mock.Anything, mock.Anything)
}

func TestOrderUseCase_UpdateOrderStatus_NotFound(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	mockNats := new(MockNatsPublisher)

	useCase := NewOrderUseCase(mockRepo, mockNats)
	ctx := context.Background()

	mockRepo.On("GetByID", mock.Anything, "non-existent").Return((*entities.Order)(nil), repositories.ErrOrderNotFound)

	_, err := useCase.UpdateOrderStatus(ctx, "non-existent", "PAID")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order not found")

	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "UpdateStatus", mock.Anything, mock.Anything)
	mockNats.AssertNotCalled(t, "PublishOrderCreated", mock.Anything, mock.Anything)
}

func TestOrderUseCase_UpdateOrderStatus_AlreadyInStatus(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	mockNats := new(MockNatsPublisher)

	useCase := NewOrderUseCase(mockRepo, mockNats)
	ctx := context.Background()

	existingOrder := &entities.Order{
		OrderID: "test-order",
		UserID:  "user123",
		Status:  "PAID",
	}

	mockRepo.On("GetByID", mock.Anything, "test-order").Return(existingOrder, nil)
	mockRepo.On("UpdateStatus", mock.Anything, "test-order", "PAID").Return(nil)

	order, err := useCase.UpdateOrderStatus(ctx, "test-order", "PAID")

	assert.NoError(t, err)
	assert.Equal(t, "PAID", order.Status)

	mockRepo.AssertExpectations(t)
	mockNats.AssertNotCalled(t, "PublishOrderCreated", mock.Anything, mock.Anything)
}
