package shared

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// CompressionLevel defines compression levels
type CompressionLevel int

const (
	CompressionNone CompressionLevel = iota
	CompressionFast
	CompressionDefault
	CompressionBest
)

// ResponseCompressor compresses responses for efficient transmission
type ResponseCompressor struct {
	level   CompressionLevel
	metrics *CompressionMetrics
}

// CompressionMetrics tracks compression metrics
type CompressionMetrics struct {
	totalResponses    int64
	compressedCount   int64
	originalSize      int64
	compressedSize    int64
	totalDuration     time.Duration
	mu                sync.RWMutex
}

// NewResponseCompressor creates a new response compressor
func NewResponseCompressor(level CompressionLevel) *ResponseCompressor {
	return &ResponseCompressor{
		level:   level,
		metrics: &CompressionMetrics{},
	}
}

// Compress compresses a response
func (c *ResponseCompressor) Compress(data interface{}) ([]byte, error) {
	start := time.Now()
	defer func() {
		c.recordMetric(time.Since(start))
	}()

	if c.level == CompressionNone {
		// No compression, just marshal to JSON
		return json.Marshal(data)
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Skip compression for small responses
	if len(jsonData) < 1024 {
		c.recordResponse(len(jsonData), len(jsonData), false)
		return jsonData, nil
	}

	// Compress with gzip
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, c.getGzipLevel())
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := writer.Write(jsonData); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	compressedData := buf.Bytes()
	c.recordResponse(len(jsonData), len(compressedData), true)

	return compressedData, nil
}

// Decompress decompresses a response
func (c *ResponseCompressor) Decompress(data []byte) ([]byte, error) {
	if c.level == CompressionNone {
		return data, nil
	}

	// Try to decompress with gzip
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		// Not compressed, return as-is
		return data, nil
	}
	defer func() {
		if err := reader.Close(); err != nil {
			// Ignore close errors in defer
			_ = err
		}
	}()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	return decompressed, nil
}

// GetMetrics returns compression metrics
func (c *ResponseCompressor) GetMetrics() map[string]interface{} {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	compressionRatio := 0.0
	if c.metrics.originalSize > 0 {
		compressionRatio = float64(c.metrics.compressedSize) / float64(c.metrics.originalSize) * 100.0
	}

	avgDuration := time.Duration(0)
	if c.metrics.totalResponses > 0 {
		avgDuration = c.metrics.totalDuration / time.Duration(c.metrics.totalResponses)
	}

	return map[string]interface{}{
		"total_responses":    c.metrics.totalResponses,
		"compressed_count":   c.metrics.compressedCount,
		"original_size":      c.metrics.originalSize,
		"compressed_size":    c.metrics.compressedSize,
		"compression_ratio":  compressionRatio,
		"avg_duration_ms":    avgDuration.Milliseconds(),
		"total_duration":     c.metrics.totalDuration.String(),
	}
}

// Helper methods

func (c *ResponseCompressor) getGzipLevel() int {
	switch c.level {
	case CompressionFast:
		return gzip.BestSpeed
	case CompressionDefault:
		return gzip.DefaultCompression
	case CompressionBest:
		return gzip.BestCompression
	default:
		return gzip.DefaultCompression
	}
}

func (c *ResponseCompressor) recordMetric(duration time.Duration) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()
	c.metrics.totalDuration += duration
}

func (c *ResponseCompressor) recordResponse(originalSize, compressedSize int, compressed bool) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	c.metrics.totalResponses++
	c.metrics.originalSize += int64(originalSize)
	c.metrics.compressedSize += int64(compressedSize)

	if compressed {
		c.metrics.compressedCount++
	}
}
