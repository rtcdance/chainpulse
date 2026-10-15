package bootstrap

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/siwe"
)

// ResolveSIWEChainID prevents an unconstrained or malformed challenge chain.
func ResolveSIWEChainID(raw string) (*big.Int, error) {
	if id, ok := new(big.Int).SetString(strings.TrimSpace(raw), 10); ok && id.Sign() > 0 {
		return id, nil
	}
	if id := chainid.ResolveChainID(raw); id > 0 {
		return big.NewInt(int64(id)), nil
	}
	return nil, fmt.Errorf("SIWE chain must be a positive numeric or known EVM chain identifier")
}

// BuildSIWENonceStore makes deployment nonce ownership explicit. Redis requests
// never silently degrade to a process-local store; the returned closer belongs
// to the composition root.
func BuildSIWENonceStore(ctx context.Context, cfg core.Config, profile string) (siwe.NonceStore, func() error, error) {
	if cfg.CacheType != "redis" {
		if profile == "production" {
			return nil, nil, fmt.Errorf("production SIWE requires a shared Redis nonce store")
		}
		return siwe.NewMemoryNonceStore(), func() error { return nil }, nil
	}
	address := strings.TrimSpace(cfg.CacheConnectionURL)
	if address == "" {
		return nil, nil, fmt.Errorf("redis nonce store endpoint is required")
	}
	opts := &redis.Options{Addr: address, DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second}
	if strings.Contains(address, "://") {
		var err error
		opts, err = redis.ParseURL(address)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid Redis nonce store URL")
		}
		opts.DialTimeout = 3 * time.Second
		opts.ReadTimeout = 3 * time.Second
		opts.WriteTimeout = 3 * time.Second
	}
	client := redis.NewClient(opts)
	probe, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(probe).Err(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("connect shared nonce store: %w", err)
	}
	store, err := siwe.NewRedisNonceStore(client)
	if err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("build shared nonce store: %w", err)
	}
	return store, client.Close, nil
}
