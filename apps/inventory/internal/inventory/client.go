package inventory

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rammtw/order-management-system/apps/order/internal/model"
	pb "github.com/rammtw/order-management-system/gen/inventory/v1"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.InventoryServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, client: pb.NewInventoryServiceClient(conn)}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) ReserveStock(ctx context.Context, orderID string, items []model.OrderItem) (bool, []string, error) {
	pbItems := make([]*pb.ReserveStockRequest_Item, 0, len(items))
	for _, item := range items {
		pbItems = append(pbItems, &pb.ReserveStockRequest_Item{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	resp, err := c.client.ReserveStock(ctx, &pb.ReserveStockRequest{
		OrderId: orderID,
		Items:   pbItems,
	})
	if err != nil {
		return false, nil, err
	}

	return resp.GetSuccess(), resp.GetUnavailableProductIds(), nil
}

func (c *Client) ReleaseStock(ctx context.Context, orderID string) error {
	_, err := c.client.ReleaseStock(ctx, &pb.ReleaseStockRequest{OrderId: orderID})
	return err
}

func (c *Client) CheckAvailability(ctx context.Context, items []model.OrderItem) (bool, []string, error) {
	pbItems := make([]*pb.CheckAvailabilityRequest_Item, 0, len(items))
	for _, item := range items {
		pbItems = append(pbItems, &pb.CheckAvailabilityRequest_Item{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	resp, err := c.client.CheckAvailability(ctx, &pb.CheckAvailabilityRequest{Items: pbItems})
	if err != nil {
		return false, nil, err
	}

	return resp.GetAvailable(), resp.GetUnavailableProductIds(), nil
}
