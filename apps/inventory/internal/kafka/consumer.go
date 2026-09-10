package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"

	"github.com/rammtw/order-management-system/apps/inventory/internal/model"
	"github.com/rammtw/order-management-system/apps/inventory/internal/service"
)

const (
	topicOrderCreated   = "order.created"
	topicOrderCancelled = "order.cancelled"
	topicOrderStatus    = "order.status"
)

type OrderEvent struct {
	OrderID    string      `json:"order_id"`
	CustomerID string      `json:"customer_id"`
	Status     int32       `json:"status"`
	TotalCents int64       `json:"total_cents"`
	Items      []OrderItem `json:"items"`
}

type OrderItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int32  `json:"quantity"`
	PriceCents  int64  `json:"price_cents"`
}

type Consumer struct {
	readers []*kafka.Reader
	svc     *service.InventoryService
}

func NewConsumer(broker, groupID string, svc *service.InventoryService) *Consumer {
	topics := []string{topicOrderCreated, topicOrderCancelled, topicOrderStatus}
	readers := make([]*kafka.Reader, 0, len(topics))

	for _, topic := range topics {
		readers = append(readers, kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{broker},
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 1,
			MaxBytes: 10 << 20,
		}))
	}

	return &Consumer{readers: readers, svc: svc}
}

func (c *Consumer) Start(ctx context.Context) {
	for _, r := range c.readers {
		go c.consume(ctx, r)
	}
}

func (c *Consumer) Close() {
	for _, r := range c.readers {
		r.Close()
	}
}

func (c *Consumer) consume(ctx context.Context, r *kafka.Reader) {
	for {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("kafka fetch error", "topic", r.Config().Topic, "error", err)
			continue
		}

		if err := c.handle(ctx, msg); err != nil {
			slog.Error("kafka handle error", "topic", msg.Topic, "error", err)
			continue
		}

		if err := r.CommitMessages(ctx, msg); err != nil {
			slog.Error("kafka commit error", "topic", msg.Topic, "error", err)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, msg kafka.Message) error {
	var event OrderEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	switch msg.Topic {
	case topicOrderCreated:
		return c.handleOrderCreated(ctx, event)
	case topicOrderCancelled:
		return c.handleOrderCancelled(ctx, event)
	case topicOrderStatus:
		return c.handleOrderStatus(ctx, event)
	}

	return nil
}

func (c *Consumer) handleOrderCreated(ctx context.Context, event OrderEvent) error {
	items := make([]model.ReservationItem, 0, len(event.Items))
	for _, item := range event.Items {
		items = append(items, model.ReservationItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	success, unavailable, err := c.svc.ReserveStock(ctx, event.OrderID, items)
	if err != nil {
		return err
	}

	if !success {
		slog.Warn("insufficient stock for order", "order_id", event.OrderID, "unavailable", unavailable)
	}

	return nil
}

func (c *Consumer) handleOrderCancelled(ctx context.Context, event OrderEvent) error {
	return c.svc.ReleaseStock(ctx, event.OrderID)
}

func (c *Consumer) handleOrderStatus(ctx context.Context, event OrderEvent) error {
	const statusDelivered = 5

	if event.Status == statusDelivered {
		return c.svc.ConfirmStock(ctx, event.OrderID)
	}

	return nil
}
