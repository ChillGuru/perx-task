package mongodb

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderDocument struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	OrderID     string             `bson:"order_id"`
	UserID      string             `bson:"user_id"`
	Items       []ItemDocument     `bson:"items"`
	TotalAmount float64            `bson:"total_amount"`
	Status      string             `bson:"status"`
	CreatedAt   time.Time          `bson:"created_at"`
}

type ItemDocument struct {
	ProductID string  `bson:"product_id"`
	Quantity  int     `bson:"quantity"`
	Price     float64 `bson:"price"`
}
