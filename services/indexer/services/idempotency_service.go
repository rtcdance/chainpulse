package service

import (
	"context"
	"time"

	"chainpulse/shared/cache"
	"chainpulse/shared/database"

	"gorm.io/gorm"
)

// IdempotencyService 幂等性服务
type IdempotencyService struct {
	cache *cache.Cache
	db    *database.Database
	ttl   time.Duration
}

// NewIdempotencyService 创建幂等性服务
func NewIdempotencyService(cache *cache.Cache, db *database.Database, ttl time.Duration) *IdempotencyService {
	return &IdempotencyService{
		cache: cache,
		db:    db,
		ttl:   ttl,
	}
}

// IsProcessed 检查事件是否已经被处理过
func (is *IdempotencyService) IsProcessed(ctx context.Context, eventKey string) (bool, error) {
	var exists bool
	err := is.cache.Get(ctx, "processed:"+eventKey, &exists)
	if err == nil {
		return exists, nil
	}
	
	// 如果缓存中不存在，检查数据库
	exists, err = is.db.EventExists(eventKey)
	if err != nil {
		return false, err
	}
	
	// 如果数据库中存在，也认为已处理
	if exists {
		// 设置缓存以提高后续检查的性能
		is.cache.Set(ctx, "processed:"+eventKey, true, is.ttl)
	}
	
	return exists, nil
}

// MarkProcessed 标记事件为已处理
func (is *IdempotencyService) MarkProcessed(ctx context.Context, eventKey string) error {
	// 在数据库中标记事件
	if err := is.db.MarkEventAsProcessed(eventKey); err != nil {
		return err
	}
	
	// 在缓存中标记事件
	return is.cache.Set(ctx, "processed:"+eventKey, true, is.ttl)
}

// MarkProcessedWithTx 标记事件为已处理（带事务）
func (is *IdempotencyService) MarkProcessedWithTx(tx *gorm.DB, eventKey string) error {
	// 在数据库中标记事件
	return is.db.MarkEventAsProcessedWithTx(tx, eventKey)
}