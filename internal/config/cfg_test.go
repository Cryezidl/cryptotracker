package config

import (
	"os"
	"testing"
	"time"
)

// clearEnv fully unsets config-related env vars for the duration of the test,
// so results don't depend on the developer's shell or a stray .env file.
func clearEnv(t *testing.T) {
	t.Helper()
	vars := []string{
		"GRPC_PORT", "CACHE_TTL",
		"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"BINANCE_URL", "BINANCE_RPS", "BINANCE_BURST",
		"COINGECKO_URL", "COINGECKO_RPS", "COINGECKO_BURST",
	}
	for _, v := range vars {
		old, existed := os.LookupEnv(v)
		os.Unsetenv(v)
		t.Cleanup(func() {
			if existed {
				os.Setenv(v, old)
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GRPCPort != "50051" {
		t.Errorf("GRPCPort = %q, want %q", cfg.GRPCPort, "50051")
	}
	if cfg.CacheTTL != 30*time.Minute {
		t.Errorf("CacheTTL = %v, want %v", cfg.CacheTTL, 30*time.Minute)
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "localhost:6379")
	}
	if cfg.Binance.Burst != 20 {
		t.Errorf("Binance.Burst = %d, want %d", cfg.Binance.Burst, 20)
	}
	if cfg.Coingecko.Burst != 5 {
		t.Errorf("Coingecko.Burst = %d, want %d", cfg.Coingecko.Burst, 5)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	clearEnv(t)
	t.Setenv("GRPC_PORT", "9999")
	t.Setenv("REDIS_ADDR", "redis:6380")
	t.Setenv("BINANCE_BURST", "42")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GRPCPort != "9999" {
		t.Errorf("GRPCPort = %q, want %q", cfg.GRPCPort, "9999")
	}
	if cfg.Redis.Addr != "redis:6380" {
		t.Errorf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "redis:6380")
	}
	if cfg.Binance.Burst != 42 {
		t.Errorf("Binance.Burst = %d, want %d", cfg.Binance.Burst, 42)
	}
}
