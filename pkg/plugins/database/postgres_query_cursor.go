package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/ports"
)

type canonicalCursor struct {
	Schema     string `json:"v"`
	Chain      string `json:"c"`
	Generation int64  `json:"g"`
	Filter     string `json:"f"`
	Block      uint64 `json:"b"`
	Log        uint64 `json:"l"`
	ID         string `json:"id"`
}

func queryCursorChain(filter map[string]any, scope []string) string {
	raw, ok := filter["chainId"]
	if !ok {
		raw = filter["network"]
	}
	values := []any{raw}
	if operators, ok := raw.(map[string]any); ok {
		values, _ = operators["$in"].([]any)
	}
	matched := map[string]bool{}
	for _, value := range values {
		key := fmt.Sprint(value)
		for _, chain := range scope {
			if key == chain || (chainid.ResolveChainID(key) > 0 && fmt.Sprint(chainid.ResolveChainID(key)) == chain) {
				matched[chain] = true
			}
		}
	}
	if len(matched) != 1 {
		return ""
	}
	for chain := range matched {
		return chain
	}
	return ""
}

func prepareQueryCursor(ctx context.Context, tx *sql.Tx, raw, chain string, filter map[string]any, offset int64) (canonicalCursor, *canonicalCursor, error) {
	encoded, err := json.Marshal(filter)
	if err != nil {
		return canonicalCursor{}, nil, fmt.Errorf("hash query cursor filter: %w", err)
	}
	digest := sha256.Sum256(encoded)
	base := canonicalCursor{Schema: "chainpulse.query-cursor.v1", Chain: chain, Filter: hex.EncodeToString(digest[:])}
	if chain != "" {
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT generation FROM reorg_generations WHERE chain_id=$1),0)`, chain).Scan(&base.Generation); err != nil {
			return base, nil, fmt.Errorf("read cursor generation: %w", err)
		}
	}
	if raw == "" {
		return base, nil, nil
	}
	invalid := func(reason string) error { return &ports.QueryCursorError{Reason: reason} }
	if chain == "" || offset != 0 || len(raw) > 4096 {
		return base, nil, invalid("cursor requires one chain and zero offset")
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return base, nil, invalid("invalid encoding")
	}
	var cursor canonicalCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return base, nil, invalid("invalid envelope")
	}
	if cursor.Schema != base.Schema || cursor.Chain != base.Chain || cursor.Filter != base.Filter || cursor.ID == "" {
		return base, nil, invalid("chain or filter does not match cursor")
	}
	if cursor.Block > math.MaxInt64 || cursor.Log > math.MaxInt64 || len(cursor.ID) > 256 {
		return base, nil, invalid("invalid sort position")
	}
	if cursor.Generation != base.Generation {
		return base, nil, &ports.QueryCursorError{Stale: true, Reason: "canonical generation changed; restart pagination"}
	}
	return base, &cursor, nil
}

func encodeQueryCursor(base canonicalCursor, event *blockchain.BlockchainEvent) (string, error) {
	base.Block = event.BlockNumber
	base.Log = event.LogIndex
	base.ID = event.ID
	payload, err := json.Marshal(base)
	if err != nil {
		return "", fmt.Errorf("encode query cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}
