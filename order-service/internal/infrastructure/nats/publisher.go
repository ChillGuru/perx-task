package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"order-service/internal/domain/entities"
	"order-service/internal/infrastructure/logger"

	"github.com/nats-io/nats.go"
)

type NatsPublisher struct {
	nc     *nats.Conn
	logger *logger.Logger
}

type OrderCreatedEvent struct {
	OrderID     string  `json:"order_id"`
	UserID      string  `json:"user_id"`
	TotalAmount float64 `json:"total_amount"`
	CreatedAt   string  `json:"created_at"`
}

func NewNatsPublisher(url string, logger *logger.Logger) (*NatsPublisher, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var nc *nats.Conn
	var err error

	for i := 0; i < 3; i++ {
		nc, err = nats.Connect(url,
			nats.Name("Order Service"),
			nats.MaxReconnects(5),
			nats.ReconnectWait(2*time.Second),
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				logger.Warn("NATS disconnected", "error", err)
			}),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				logger.Info("NATS reconnected", "url", nc.ConnectedUrl())
			}),
		)

		if err == nil {
			logger.Info("Connected to NATS", "url", url)
			return &NatsPublisher{nc: nc, logger: logger}, nil
		}

		logger.Warn("Failed to connect to NATS", "attempt", i+1, "error", err)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("failed to connect to NATS: %w", err)
		case <-time.After(2 * time.Second):
			continue
		}
	}

	return nil, fmt.Errorf("failed to connect to NATS after retries: %w", err)
}

func (p *NatsPublisher) PublishOrderCreated(ctx context.Context, order *entities.Order) error {
	event := OrderCreatedEvent{
		OrderID:     order.OrderID,
		UserID:      order.UserID,
		TotalAmount: order.TotalAmount,
		CreatedAt:   order.CreatedAt.Format(time.RFC3339),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	subject := "order.created"

	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			p.logger.Warn("Context cancelled while publishing to NATS")
			return ctx.Err()
		default:
			err := p.nc.Publish(subject, data)
			if err != nil {
				p.logger.Warn("Failed to publish to NATS", "attempt", i+1, "error", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if err := p.nc.FlushTimeout(2 * time.Second); err != nil {
				p.logger.Warn("Failed to flush NATS connection", "error", err)
				continue
			}

			p.logger.Info("Successfully published order.created event", "order_id", order.OrderID)
			return nil
		}
	}

	p.logger.Error("Failed to publish event to NATS after retries", "order_id", order.OrderID)
	return fmt.Errorf("failed to publish event after retries")
}

func (p *NatsPublisher) Close() {
	if p.nc != nil && p.nc.IsConnected() {
		p.nc.Close()
		p.logger.Info("NATS connection closed")
	}
}
