package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"golang.org/x/time/rate"
)

type Config struct {
	GRPCPort string        `env:"GRPC_PORT" env-default:"50051"`
	CacheTTL time.Duration `env:"CACHE_TTL" env-default:"30m"`

	Redis struct {
		Addr     string `env:"REDIS_ADDR" env-default:"localhost:6379"`
		Password string `env:"REDIS_PASSWORD" env-default:""`
		DB       int    `env:"REDIS_DB" env-default:"0"`
	}

	Binance struct {
		URL   string     `env:"BINANCE_URL" env-default:"https://data-api.binance.vision/api/v3/ticker/price"`
		RPS   rate.Limit `env:"BINANCE_RPS" env-default:"10"`
		Burst int        `env:"BINANCE_BURST" env-default:"20"`
	}

	Coingecko struct {
		URL   string     `env:"COINGECKO_URL" env-default:"https://api.coingecko.com/api/v3/simple/price"`
		RPS   rate.Limit `env:"COINGECKO_RPS" env-default:"0.16666667"`
		Burst int        `env:"COINGECKO_BURST" env-default:"5"`
	}
}

func Load() (*Config, error) {
	var cfg Config
	_ = cleanenv.ReadConfig(".env", &cfg)

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
