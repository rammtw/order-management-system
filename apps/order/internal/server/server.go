package server

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/rammtw/order-management-system/apps/order/internal/handler"
	pb "github.com/rammtw/order-management-system/gen/order/v1"
)

type Server struct {
	grpc *grpc.Server
	lis  net.Listener
}

func New(h *handler.OrderHandler, opts ...grpc.ServerOption) *Server {
	srv := grpc.NewServer(opts...)
	pb.RegisterOrderServiceServer(srv, h)
	reflection.Register(srv)

	return &Server{grpc: srv}
}

func (s *Server) Start(port string) error {
	var err error
	s.lis, err = net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	return s.grpc.Serve(s.lis)
}

func (s *Server) Stop(_ context.Context) {
	s.grpc.GracefulStop()
}
