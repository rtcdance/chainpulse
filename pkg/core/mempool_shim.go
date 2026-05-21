package core

import (
	"time"

	"github.com/rtcdance/chainpulse/pkg/mempool"
)

type AAMempoolEntry = mempool.AAMempoolEntry
type PreValidationResult = mempool.PreValidationResult
type AAMempool = mempool.AAMempool

func NewAAMempool(maxSize int, entryTTL time.Duration) *AAMempool {
	return mempool.NewAAMempool(maxSize, entryTTL)
}