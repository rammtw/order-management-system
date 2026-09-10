package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rammtw/order-management-system/apps/inventory/internal/model"
	"github.com/rammtw/order-management-system/apps/inventory/internal/service"
	pb "github.com/rammtw/order-management-system/gen/inventory/v1"
)

type InventoryHandler struct {
	pb.UnimplementedInventoryServiceServer
	svc *service.InventoryService
}

func NewInventoryHandler(svc *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{svc: svc}
}

func (h *InventoryHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	if req.GetSku() == "" {
		return nil, status.Error(codes.InvalidArgument, "sku is required")
	}
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	p, err := h.svc.CreateProduct(ctx, req.GetSku(), req.GetName(), req.GetDescription(), req.GetPriceCents(), req.GetQuantity())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.CreateProductResponse{Product: toProto(p)}, nil
}

func (h *InventoryHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	p, err := h.svc.GetProduct(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetProductResponse{Product: toProto(p)}, nil
}

func (h *InventoryHandler) GetProductBySKU(ctx context.Context, req *pb.GetProductBySKURequest) (*pb.GetProductBySKUResponse, error) {
	if req.GetSku() == "" {
		return nil, status.Error(codes.InvalidArgument, "sku is required")
	}

	p, err := h.svc.GetProductBySKU(ctx, req.GetSku())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetProductBySKUResponse{Product: toProto(p)}, nil
}

func (h *InventoryHandler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	products, total, err := h.svc.ListProducts(ctx, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, toGRPCError(err)
	}

	pbProducts := make([]*pb.Product, 0, len(products))
	for _, p := range products {
		pbProducts = append(pbProducts, toProto(p))
	}

	return &pb.ListProductsResponse{Products: pbProducts, Total: total}, nil
}

func (h *InventoryHandler) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.UpdateStockResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	p, err := h.svc.UpdateStock(ctx, req.GetId(), req.GetDelta())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.UpdateStockResponse{Product: toProto(p)}, nil
}

func (h *InventoryHandler) CheckAvailability(ctx context.Context, req *pb.CheckAvailabilityRequest) (*pb.CheckAvailabilityResponse, error) {
	items := make([]model.ReservationItem, 0, len(req.GetItems()))
	for _, i := range req.GetItems() {
		items = append(items, model.ReservationItem{
			ProductID: i.GetProductId(),
			Quantity:  i.GetQuantity(),
		})
	}

	available, unavailable := h.svc.CheckAvailability(ctx, items)

	return &pb.CheckAvailabilityResponse{
		Available:             available,
		UnavailableProductIds: unavailable,
	}, nil
}

func (h *InventoryHandler) ReserveStock(ctx context.Context, req *pb.ReserveStockRequest) (*pb.ReserveStockResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	items := make([]model.ReservationItem, 0, len(req.GetItems()))
	for _, i := range req.GetItems() {
		items = append(items, model.ReservationItem{
			ProductID: i.GetProductId(),
			Quantity:  i.GetQuantity(),
		})
	}

	success, unavailable, err := h.svc.ReserveStock(ctx, req.GetOrderId(), items)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.ReserveStockResponse{
		Success:               success,
		UnavailableProductIds: unavailable,
	}, nil
}

func (h *InventoryHandler) ReleaseStock(ctx context.Context, req *pb.ReleaseStockRequest) (*pb.ReleaseStockResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	if err := h.svc.ReleaseStock(ctx, req.GetOrderId()); err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.ReleaseStockResponse{}, nil
}

func toProto(p *model.Product) *pb.Product {
	return &pb.Product{
		Id:               p.ID,
		Sku:              p.SKU,
		Name:             p.Name,
		Description:      p.Description,
		PriceCents:       p.PriceCents,
		Quantity:         p.Quantity,
		ReservedQuantity: p.ReservedQuantity,
		CreatedAt:        timestamppb.New(p.CreatedAt),
		UpdatedAt:        timestamppb.New(p.UpdatedAt),
	}
}

func toGRPCError(err error) error {
	switch err {
	case service.ErrProductNotFound:
		return status.Error(codes.NotFound, err.Error())
	case service.ErrInsufficientStock:
		return status.Error(codes.FailedPrecondition, err.Error())
	case service.ErrInvalidQuantity:
		return status.Error(codes.InvalidArgument, err.Error())
	case service.ErrDuplicateSKU:
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
