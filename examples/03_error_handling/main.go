package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

// 错误分类: Transient(可重试), Permanent(不可重试), Critical(需立即关注)
// 对应生产代码: pkg/core/errors.go

type ErrorType int

const (
	ErrorTypeTransient ErrorType = iota
	ErrorTypePermanent
	ErrorTypeCritical
)

// RPCCaller 模拟区块链 RPC 调用
type RPCCaller struct {
	callCount int
}

// Call 模拟可能失败的网络请求
func (r *RPCCaller) Call(ctx context.Context, blockNum uint64) (string, error) {
	r.callCount++

	if r.callCount%3 == 0 {
		return "", ErrRPCTimeout
	}
	if blockNum == 0 {
		return "", ErrBlockNotFound
	}

	rand.Seed(time.Now().UnixNano())
	if rand.Intn(100) < 5 {
		return "", ErrRPCCritical
	}

	return fmt.Sprintf("block_%d_data", blockNum), nil
}

// 预定义 sentinel errors (对应 pkg/core/errors.go)
var (
	ErrRPCTimeout    = &RPCError{Type: ErrorTypeTransient, msg: "rpc timeout"}
	ErrBlockNotFound = &RPCError{Type: ErrorTypePermanent, msg: "block not found"}
	ErrRPCCritical   = &RPCError{Type: ErrorTypeCritical, msg: "rpc critical failure"}
)

type RPCError struct {
	Type ErrorType
	msg  string
}

func (e *RPCError) Error() string { return e.msg }

// ClassifyError 错误分类 (简化版，对应 ClassifyError in pkg/core/errors.go)
func ClassifyError(err error) ErrorType {
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		return rpcErr.Type
	}
	return ErrorTypePermanent
}

// CallWithRetry 带指数退避的重试逻辑
// 对应生产代码: pkg/services/resilience/retry_logic.go
func CallWithRetry(ctx context.Context, caller *RPCCaller, blockNum uint64, maxRetries int) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := caller.Call(ctx, blockNum)
		if err == nil {
			if attempt > 0 {
				fmt.Printf("  Succeeded after %d retries\n", attempt)
			}
			return result, nil
		}

		lastErr = err
		errType := ClassifyError(err)

		switch errType {
		case ErrorTypeCritical:
			fmt.Printf("  ✗ Critical error (attempt %d): %v\n", attempt+1, err)
			return "", err
		case ErrorTypePermanent:
			fmt.Printf("  ✗ Permanent error (attempt %d): %v - not retrying\n", attempt+1, err)
			return "", err
		case ErrorTypeTransient:
			if attempt < maxRetries {
				backoff := time.Duration(1<<uint(attempt)) * 10 * time.Millisecond
				fmt.Printf("  ⏳ Transient error (attempt %d): %v - retrying in %v\n", attempt+1, err, backoff)

				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
		}
	}

	return "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func main() {
	ctx := context.Background()
	caller := &RPCCaller{}

	fmt.Println("=== Test 1: Successful call with retries ===")
	result, err := CallWithRetry(ctx, caller, 100, 3)
	if err == nil {
		fmt.Printf("  Result: %s\n\n", result)
	}

	fmt.Println("=== Test 2: Permanent error (no retry) ===")
	_, err = CallWithRetry(ctx, caller, 0, 3)
	fmt.Printf("  Final error: %v\n\n", err)

	fmt.Println("=== Test 3: Transient errors then success ===")
	result, err = CallWithRetry(ctx, caller, 200, 3)
	if err == nil {
		fmt.Printf("  Result: %s\n", result)
	}
}
