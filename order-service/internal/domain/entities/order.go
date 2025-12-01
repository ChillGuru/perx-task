package entities

import "time"

type Order struct {
	OrderID     string    `json:"order_id"`
	UserID      string    `json:"user_id"`
	Items       []Item    `json:"items"`
	TotalAmount float64   `json:"total_amount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Item struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

func ValidStatus(status string) bool {
	validStatuses := map[string]bool{
		"PENDING":   true,
		"PAID":      true,
		"CANCELLED": true,
		"FAILED":    true,
	}
	return validStatuses[status]
}
