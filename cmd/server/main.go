package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"cryptotracker/internal/config"
	"cryptotracker/internal/repository/cache"
	"cryptotracker/internal/repository/external"
	"cryptotracker/internal/repository/external/aggregator"
	"cryptotracker/internal/repository/external/binance"
	"cryptotracker/internal/repository/external/coingecko"
	"cryptotracker/internal/service"
	pb "cryptotracker/proto/rates"
)

type server struct {
	pb.UnimplementedRatesServiceServer
	rateService *service.Service
}

func newServer(rateService *service.Service) *server {
	return &server{rateService: rateService}
}

func (s *server) GetRate(ctx context.Context, req *pb.GetRateRequest) (*pb.RateResponse, error) {
	log.Printf("Получен запрос: %v", req)

	if req.GetPair() == nil {
		return nil, status.Error(codes.InvalidArgument, "currency pair is required")
	}

	base := req.GetPair().GetBase()
	quote := req.GetPair().GetQuote()

	rate, err := s.rateService.GetRate(ctx, base, quote)
	if err != nil {
		log.Printf("Error getting rate for %s/%s: %v", base, quote, err)
		return nil, status.Errorf(codes.Internal, "internal service error: %v", err)
	}

	return &pb.RateResponse{
		Pair:      req.GetPair(),
		Price:     rate.Price,
		UpdatedAt: timestamppb.New(rate.Timestamp),
		Source:    rate.Source,
	}, nil
}

func (s *server) ListRates(ctx context.Context, req *pb.ListRatesRequest) (*pb.ListRatesResponse, error) {
	if len(req.GetPairs()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one currency pair is required")
	}

	responses := make([]*pb.RateResponse, 0, len(req.GetPairs()))
	for _, pair := range req.GetPairs() {
		base := pair.GetBase()
		quote := pair.GetQuote()

		rate, err := s.rateService.GetRate(ctx, base, quote)
		if err != nil {
			log.Printf("Error getting rate for %s/%s: %v", base, quote, err)
			continue
		}

		responses = append(responses, &pb.RateResponse{
			Pair:      pair,
			Price:     rate.Price,
			UpdatedAt: timestamppb.New(rate.Timestamp),
			Source:    rate.Source,
		})
	}

	if len(responses) == 0 {
		return nil, status.Error(codes.Internal, "failed to fetch rates for all requested currency pairs")
	}

	return &pb.ListRatesResponse{Rates: responses}, nil
}

func main() {
	// 0. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 1. Initialize the cache and service
	cache := cache.NewReddisCache(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)

	coinGeckoClient := coingecko.New(cfg.Coingecko.URL, "CoinGecko", cfg.Coingecko.RPS, cfg.Coingecko.Burst)
	binanceClient := binance.New(cfg.Binance.URL, "Binance", cfg.Binance.RPS, cfg.Binance.Burst)

	aggregator := aggregator.New([]external.Provider{coinGeckoClient, binanceClient})
	rateService := service.New(cache, aggregator, cfg.CacheTTL)

	// 2. Create a TCP listener on port from config
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// 3. Create a new gRPC server
	grpcServer := grpc.NewServer()

	// 4. Register the service implementation with the gRPC server
	gRPCServerImpl := newServer(rateService)
	pb.RegisterRatesServiceServer(grpcServer, gRPCServerImpl)

	// 5. Start serving requests
	log.Printf("gRPC сервер запущен на порту :%s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
