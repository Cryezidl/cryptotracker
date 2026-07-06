package main

import (
	"context"
	"log"
	"net"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"cryptotracker/internal/broadcast"
	"cryptotracker/internal/config"
	"cryptotracker/internal/model"
	"cryptotracker/internal/repository/cache"
	"cryptotracker/internal/repository/external"
	"cryptotracker/internal/repository/external/aggregator"
	"cryptotracker/internal/repository/external/binance"
	"cryptotracker/internal/repository/external/coingecko"
	"cryptotracker/internal/service"
	"cryptotracker/internal/worker"
	pb "cryptotracker/proto/rates"
)

type server struct {
	pb.UnimplementedRatesServiceServer
	rateService *service.Service
	broadcaster *broadcast.Broadcaster
}

func newServer(rateService *service.Service, broadcaster *broadcast.Broadcaster) *server {
	return &server{rateService: rateService, broadcaster: broadcaster}
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

func (s *server) SubscribeRates(req *pb.SubscribeRatesRequest, stream grpc.ServerStreamingServer[pb.RateUpdate]) error {
	id := uuid.New().String()
	ch := s.broadcaster.Subscribe(id)
	defer s.broadcaster.Unsubscribe(id)

	for {
		select {
		case update := <-ch:
			err := stream.Send((&pb.RateUpdate{
				Pair:          &pb.CurrencyPair{Base: update.Pair.Base, Quote: update.Pair.Quote},
				OldPrice:      update.OldPrice,
				NewPrice:      update.NewPrice,
				ChangePercent: update.ChangePercent,
				UpdatedAt:     timestamppb.Now(),
			}))
			if err != nil {
				return err
			}

		case <-stream.Context().Done():
			return nil
		}
	}
}

// parseTrackedPairs turns the config's "BASE/QUOTE" strings into PairKeys,
// skipping malformed entries so one typo doesn't kill the whole worker.
func parseTrackedPairs(raw []string) []model.PairKey {
	pairs := make([]model.PairKey, 0, len(raw))
	for _, p := range raw {
		parts := strings.Split(strings.TrimSpace(p), "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			log.Printf("skipping invalid tracked pair %q (want BASE/QUOTE)", p)
			continue
		}
		pairs = append(pairs, model.PairKey{Base: parts[0], Quote: parts[1]})
	}
	return pairs
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

	broadcaster := broadcast.New()
	subsWorker := worker.New(rateService, broadcaster, cfg.WorkerInterval, parseTrackedPairs(cfg.TrackedPairs))

	// 2. Create a TCP listener on port from config
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// 3. Create a new gRPC server
	grpcServer := grpc.NewServer()

	// 4. Register the service implementation with the gRPC server
	gRPCServerImpl := newServer(rateService, broadcaster)
	pb.RegisterRatesServiceServer(grpcServer, gRPCServerImpl)

	go subsWorker.Start(context.Background())
	// 5. Start serving requests
	log.Printf("gRPC сервер запущен на порту :%s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
