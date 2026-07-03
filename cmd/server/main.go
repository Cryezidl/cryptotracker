package main

import (
	"context"
	"log"
	"net"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

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

	val, err := s.rateService.GetRate(ctx, base, quote)
	if err != nil {
		log.Printf("Error getting rate for %s/%s: %v", base, quote, err)
		return nil, status.Errorf(codes.Internal, "internal service error: %v", err)
	}

	return &pb.RateResponse{
		Pair:      req.GetPair(),
		Price:     val,
		UpdatedAt: timestamppb.Now(), //ЗАГЛУШКА
		Source:    "in process",      //ЗАГЛУШКА
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

		val, err := s.rateService.GetRate(ctx, base, quote)
		if err != nil {
			log.Printf("Error getting rate for %s/%s: %v", base, quote, err)
			continue
		}

		responses = append(responses, &pb.RateResponse{
			Pair:      pair,
			Price:     val,
			UpdatedAt: timestamppb.Now(), //ЗАГЛУШКА
			Source:    "in process",      //ЗАГЛУШКА
		})
	}

	if len(responses) == 0 {
		return nil, status.Error(codes.Internal, "failed to fetch rates for all requested currency pairs")
	}

	return &pb.ListRatesResponse{Rates: responses}, nil
}

func main() {
	//ЗАГЛУШКИ В АРГУМЕНТАХ, ДОБАВИТЬ ПОЗЖЕ КОНФИГ
	cache := cache.NewReddisCache("localhost:6379", "", 0)
	coinGeckoClient := coingecko.New("https://api.coingecko.com/api/v3/simple/price", "CoinGecko", rate.Limit(10.0/60.0), 5)
	binanceClient := binance.New("https://data-api.binance.vision/api/v3/ticker/price", "Binance", rate.Limit(10), 20)
	aggregator := aggregator.New([]external.Provider{coinGeckoClient, binanceClient})
	rateService := service.New(cache, aggregator)

	// 2. Create a TCP listener on port 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// 3. Create a new gRPC server
	grpcServer := grpc.NewServer()

	// 4. Register the service implementation with the gRPC server
	gRPCServerImpl := newServer(rateService)
	pb.RegisterRatesServiceServer(grpcServer, gRPCServerImpl)

	// // 4. Включаем рефлексию для отладки (через evans или postman), опционально
	// reflection.Register(grpcServer)

	// 5. Start serving requests
	log.Printf("gRPC сервер запущен на порту :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
