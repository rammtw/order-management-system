package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rammtw/order-management-system/apps/order/internal/model"
	"github.com/rammtw/order-management-system/apps/order/internal/service"
	pb "github.com/rammtw/order-management-system/gen/order/v1"
)

type OrderHandler struct {
	pb.UnimplementedOrderServiceServer
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	if req.GetCustomerId() == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}

	items := make([]model.OrderItem, 0, len(req.GetItems()))
	for _, i := range req.GetItems() {
		items = append(items, model.OrderItem{
			ProductID:   i.GetProductId(),
			ProductName: i.GetProductName(),
			Quantity:    i.GetQuantity(),
			PriceCents:  i.GetPriceCents(),
		})
	}

	order, err := h.svc.CreateOrder(ctx, req.GetCustomerId(), items)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.CreateOrderResponse{Order: toProto(order)}, nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	order, err := h.svc.GetOrder(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetOrderResponse{Order: toProto(order)}, nil
}

func (h *OrderHandler) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	if req.GetCustomerId() == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}

	orders, total, err := h.svc.ListOrders(ctx, req.GetCustomerId(), req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, toGRPCError(err)
	}

	pbOrders := make([]*pb.Order, 0, len(orders))
	for _, o := range orders {
		pbOrders = append(pbOrders, toProto(o))
	}

	return &pb.ListOrdersResponse{Orders: pbOrders, Total: total}, nil
}

func (h *OrderHandler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	order, err := h.svc.CancelOrder(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.CancelOrderResponse{Order: toProto(order)}, nil
}

func (h *OrderHandler) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.UpdateOrderStatusResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	order, err := h.svc.UpdateOrderStatus(ctx, req.GetId(), model.OrderStatus(req.GetStatus()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.UpdateOrderStatusResponse{Order: toProto(order)}, nil
}

func toProto(o *model.Order) *pb.Order {
	items := make([]*pb.OrderItem, 0, len(o.Items))
	for _, i := range o.Items {
		items = append(items, &pb.OrderItem{
			ProductId:   i.ProductID,
			ProductName: i.ProductName,
			Quantity:    i.Quantity,
			PriceCents:  i.PriceCents,
		})
	}

	return &pb.Order{
		Id:         o.ID,
		CustomerId: o.CustomerID,
		Items:      items,
		Status:     pb.OrderStatus(o.Status),
		TotalCents: o.TotalCents,
		CreatedAt:  timestamppb.New(o.CreatedAt),
		UpdatedAt:  timestamppb.New(o.UpdatedAt),
	}
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, service.ErrOrderNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrEmptyItems):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, service.ErrInvalidStatus):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, service.ErrAlreadyCancelled):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
