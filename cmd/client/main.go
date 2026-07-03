package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"cryptotracker/internal/config"
	pb "cryptotracker/proto/rates"
)

func parsePairs(pairs string) []string {
	pairs = strings.TrimSpace(pairs)
	if pairs == "" {
		return []string{}
	}

	return strings.Split(pairs, ",")
}

func parsePair(pair string) []string {
	pair = strings.TrimSpace(pair)
	if pair == "" {
		return []string{}
	}

	splittedPair := strings.Split(pair, "/")
	if len(splittedPair) != 2 {
		return []string{}
	}
	return splittedPair
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	modeFlag := flag.String("mode", "get", "режим: get | list | subscribe")
	pairsFlag := flag.String("pairs", "BTC/USD", "pairs of currencies to be converted. If you want convert list of currencies, follow this example: \"BTC/USD,BTC/EUR\"")
	flag.Parse()
	// 1. Устанавливаем соединение с сервером.
	conn, err := grpc.NewClient("localhost:"+cfg.GRPCPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Не удалось подключиться к gRPC-серверу: %v", err)
	}
	defer conn.Close()

	// 2. Создаем клиентскую заглушку (stub) из сгенерированного кода
	client := pb.NewRatesServiceClient(conn)

	switch *modeFlag {
	case "get":
		{
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			pairs := parsePairs(*pairsFlag)
			if len(pairs) == 0 {
				fmt.Print("pairs not provided")
				return
			}
			pair := pairs[0]
			splittedPair := parsePair(pair)
			if len(splittedPair) == 0 {
				fmt.Print("invalid pair format")
				return
			}
			req := &pb.GetRateRequest{Pair: &pb.CurrencyPair{Base: splittedPair[0], Quote: splittedPair[1]}}
			resp, err := client.GetRate(ctx, req)
			if err != nil {
				fmt.Printf("something went wrong: %v", err)
				return
			}
			fmt.Printf(`
Response
Base: %v
Quote: %v
Price: %v
Updated at: %v
			`, splittedPair[0], splittedPair[1], resp.GetPrice(), resp.GetUpdatedAt().AsTime())

		}
	case "list":
		{
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			pairs := parsePairs(*pairsFlag)
			if len(pairs) == 0 {
				fmt.Print("pairs not provided")
				return
			}

			reqPairs := make([]*pb.CurrencyPair, 0, len(pairs))

			for _, pair := range pairs {
				splittedPair := parsePair(pair)
				if len(splittedPair) != 0 {
					curPair := &pb.CurrencyPair{Base: splittedPair[0], Quote: splittedPair[1]}
					reqPairs = append(reqPairs, curPair)
				}
			}

			if len(reqPairs) == 0 {
				fmt.Print("invalid pairs format")
				return
			}

			resp, err := client.ListRates(ctx, &pb.ListRatesRequest{Pairs: reqPairs})
			if err != nil {
				fmt.Printf("something went wrong: %v", err)
				return
			}

			for i, rate := range resp.GetRates() {
				fmt.Printf(`
Response №%d
Base: %v
Quote: %v
Price: %v
Updated at: %v
			`, i+1, rate.GetPair().GetBase(), rate.GetPair().GetQuote(), rate.GetPrice(), rate.GetUpdatedAt().AsTime())
			}
		}
	}
}
