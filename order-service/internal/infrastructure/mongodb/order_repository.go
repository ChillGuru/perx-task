package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"order-service/internal/domain/entities"
	"order-service/internal/domain/repositories"
	"order-service/internal/infrastructure/logger"
)

type OrderRepositoryMongo struct {
	client     *mongo.Client
	collection *mongo.Collection
	logger     *logger.Logger
}

func NewOrderRepositoryMongo(uri, dbName string, logger *logger.Logger) (*OrderRepositoryMongo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	collection := client.Database(dbName).Collection("orders")

	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "order_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	return &OrderRepositoryMongo{
		client:     client,
		collection: collection,
		logger:     logger,
	}, nil
}

func (r *OrderRepositoryMongo) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.client.Disconnect(ctx)
}

func (r *OrderRepositoryMongo) Create(ctx context.Context, order *entities.Order) error {
	doc := toOrderDocument(order)

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return repositories.ErrOrderAlreadyExists
		}
		return fmt.Errorf("failed to insert order: %w", err)
	}

	return nil
}

func (r *OrderRepositoryMongo) GetByID(ctx context.Context, orderID string) (*entities.Order, error) {
	var doc OrderDocument
	err := r.collection.FindOne(ctx, bson.M{"order_id": orderID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, repositories.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to find order: %w", err)
	}

	return toOrderEntity(&doc), nil
}

func (r *OrderRepositoryMongo) UpdateStatus(ctx context.Context, orderID, status string) error {
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"order_id": orderID},
		bson.M{"$set": bson.M{"status": status}},
	)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	if result.MatchedCount == 0 {
		return repositories.ErrOrderNotFound
	}

	if result.ModifiedCount == 0 && result.MatchedCount > 0 {
		r.logger.Info("Order status already set to requested value",
			"order_id", orderID,
			"status", status,
			"matched_count", result.MatchedCount,
			"modified_count", result.ModifiedCount)
	} else {
		r.logger.Info("Order status updated successfully",
			"order_id", orderID,
			"new_status", status,
			"matched_count", result.MatchedCount,
			"modified_count", result.ModifiedCount)
	}

	return nil
}

func toOrderDocument(order *entities.Order) *OrderDocument {
	doc := &OrderDocument{
		OrderID:     order.OrderID,
		UserID:      order.UserID,
		TotalAmount: order.TotalAmount,
		Status:      order.Status,
		CreatedAt:   order.CreatedAt,
		Items:       make([]ItemDocument, len(order.Items)),
	}

	for i, item := range order.Items {
		doc.Items[i] = ItemDocument{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}

	return doc
}

func toOrderEntity(doc *OrderDocument) *entities.Order {
	items := make([]entities.Item, len(doc.Items))
	for i, item := range doc.Items {
		items[i] = entities.Item{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}

	return &entities.Order{
		OrderID:     doc.OrderID,
		UserID:      doc.UserID,
		Items:       items,
		TotalAmount: doc.TotalAmount,
		Status:      doc.Status,
		CreatedAt:   doc.CreatedAt,
	}
}
