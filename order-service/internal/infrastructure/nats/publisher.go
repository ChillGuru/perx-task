package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"order-service/internal/domain/entities"

	"github.com/nats-io/nats.go"
)

type NatsPublisher struct {
	nc *nats.Conn
}

type OrderCreatedEvent struct {
	OrderID     string  `json:"order_id"`
	UserID      string  `json:"user_id"`
	TotalAmount float64 `json:"total_amount"`
	CreatedAt   string  `json:"created_at"`
}

func NewNatsPublisher(url string) (*NatsPublisher, error) {
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
				log.Printf("NATS disconnected: %v", err)
			}),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
			}),
		)

		if err == nil {
			log.Printf("Connected to NATS at %s", url)
			return &NatsPublisher{nc: nc}, nil
		}

		log.Printf("Attempt %d failed to connect to NATS: %v", i+1, err)

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

	go func() {
		publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		subject := "order.created"

		for i := 0; i < 3; i++ {
			select {
			case <-publishCtx.Done():
				log.Printf("Context cancelled while publishing to NATS")
				return
			default:
				err := p.nc.Publish(subject, data)
				if err != nil {
					log.Printf("Attempt %d failed to publish to NATS: %v", i+1, err)

					time.Sleep(1 * time.Second)
					continue
				}

				if err := p.nc.FlushTimeout(2 * time.Second); err != nil {
					log.Printf("Failed to flush NATS connection: %v", err)
					continue
				}

				log.Printf("Successfully published order.created event for order %s", order.OrderID)
				return
			}
		}

		log.Printf("Failed to publish event to NATS after retries for order %s", order.OrderID)
	}()

	return nil
}

func (p *NatsPublisher) Close() {
	if p.nc != nil && p.nc.IsConnected() {
		p.nc.Close()
		log.Print("NATS connection closed")
	}
}
