package server

import (
	"context"
	"net"
	"time"

	"github.com/aliexe/ms-priceFetcher/internal/service"
	"github.com/aliexe/ms-priceFetcher/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCPriceFetcherServer struct {
	svc service.PriceService
	proto.UnimplementedPriceFetcherServer
}

type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
}

func MakeGRPCServer(listenAddr string, svc service.PriceService) (*GRPCServer, error) {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}

	server := grpc.NewServer()
	proto.RegisterPriceFetcherServer(server, NewGRPCPriceFetcherServer(svc))
	reflection.Register(server)

	return &GRPCServer{
		server:   server,
		listener: ln,
	}, nil
}

func (s *GRPCServer) Run() error {
	return s.server.Serve(s.listener)
}

func (s *GRPCServer) Stop() {
	s.server.GracefulStop()
}

func NewGRPCPriceFetcherServer(svc service.PriceService) *GRPCPriceFetcherServer {
	return &GRPCPriceFetcherServer{svc: svc}
}

func (s *GRPCPriceFetcherServer) FetchPrice(ctx context.Context, req *proto.FetchPriceRequest) (*proto.FetchPriceResponse, error) {
	reqID := uuid.New().ID()
	ctx = context.WithValue(ctx, "requestID", reqID)
	price, err := s.svc.FetchPrice(ctx, req.Ticker)
	if err != nil {
		return nil, err
	}
	resp := &proto.FetchPriceResponse{
		Ticker: req.Ticker,
		Price:  float32(price),
	}
	return resp, nil
}

func (s *GRPCPriceFetcherServer) StreamPrices(req *proto.StreamPricesRequest, stream proto.PriceFetcher_StreamPricesServer) error {
	ctx := stream.Context()
	reqID := uuid.New().ID()
	ctx = context.WithValue(ctx, "requestID", reqID)

	// Default interval to 5 seconds if not specified
	interval := time.Duration(req.IntervalSeconds)
	if interval == 0 {
		interval = 5 * time.Second
	}

	doneChan := make(chan struct{})

	// Start price update goroutine
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Send updates for all requested tickers
				for _, t := range req.Tickers {
					price, err := s.svc.FetchPrice(ctx, t)
					if err != nil {
						continue
					}

					resp := &proto.StreamPricesResponse{
						Ticker:    t,
						Price:     float32(price),
						Timestamp: time.Now().Format(time.RFC3339),
					}

					if err := stream.Send(resp); err != nil {
						return
					}
				}
			case <-doneChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Keep streaming until client disconnects
	<-ctx.Done()
	close(doneChan)

	return nil
}
