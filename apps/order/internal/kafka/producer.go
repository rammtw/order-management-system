package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"

	"github.com/rammtw/order-management-system/apps/order/internal/model"
)

const (
	TopicOrderCreated   = "order.created"
	TopicOrderCancelled = "order.cancelled"
	TopicOrderStatus    = "order.status"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(broker string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(broker),
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
		},
	}
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

type OrderEvent struct {
	OrderID    string            `json:"order_id"`
	CustomerID string            `json:"customer_id"`
	Status     model.OrderStatus `json:"status"`
	TotalCents int64             `json:"total_cents"`
	Items      []model.OrderItem `json:"items"`
}

func (p *Producer) PublishOrderEvent(ctx context.Context, topic string, order *model.Order) error {
	event := OrderEvent{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		Status:     order.Status,
		TotalCents: order.TotalCents,
		Items:      order.Items,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(order.ID),
		Value: data,
	})
}
