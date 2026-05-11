package shared

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	totalResponses  int64
	compressedCount int64
	originalSize    int64
	compressedSize  int64
	totalDuration   time.Duration
	mu              sync.RWMutex
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
			slog.Debug("gzip reader close error", "error", err)
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
	totalResponses := c.metrics.totalResponses
	compressedCount := c.metrics.compressedCount
	originalSize := c.metrics.originalSize
	compressedSize := c.metrics.compressedSize
	totalDuration := c.metrics.totalDuration
	c.metrics.mu.RUnlock()

	compressionRatio := 0.0
	if originalSize > 0 {
		compressionRatio = float64(compressedSize) / float64(originalSize) * 100.0
	}

	avgDuration := time.Duration(0)
	if totalResponses > 0 {
		avgDuration = totalDuration / time.Duration(totalResponses)
	}

	coveragePosture := classifyCompressionCoveragePosture(totalResponses, compressedCount)
	efficiencyPosture := classifyCompressionEfficiencyPosture(totalResponses, compressedCount, compressionRatio, avgDuration.Milliseconds())

	return map[string]interface{}{
		"total_responses":    totalResponses,
		"compressed_count":   compressedCount,
		"original_size":      originalSize,
		"compressed_size":    compressedSize,
		"compression_ratio":  compressionRatio,
		"avg_duration_ms":    avgDuration.Milliseconds(),
		"total_duration":     totalDuration.String(),
		"coverage_posture":   coveragePosture,
		"efficiency_posture": efficiencyPosture,
		"reliability_hint":   buildCompressionReliabilityHint(coveragePosture, efficiencyPosture),
	}
}

// GetRuntimeMetrics returns a compact runtime surface for compression coverage
// and delivery efficiency on top of the raw compression metrics.
func (c *ResponseCompressor) GetRuntimeMetrics() map[string]interface{} {
	metrics := c.GetMetrics()

	totalResponses, _ := metrics["total_responses"].(int64)
	compressedCount, _ := metrics["compressed_count"].(int64)
	compressionRatio, _ := metrics["compression_ratio"].(float64)
	avgDurationMS, _ := metrics["avg_duration_ms"].(int64)

	return map[string]interface{}{
		"total_responses":    totalResponses,
		"compressed_count":   compressedCount,
		"original_size":      metrics["original_size"],
		"compressed_size":    metrics["compressed_size"],
		"compression_ratio":  compressionRatio,
		"avg_duration_ms":    avgDurationMS,
		"coverage_posture":   metrics["coverage_posture"],
		"efficiency_posture": metrics["efficiency_posture"],
		"reliability_hint":   metrics["reliability_hint"],
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

func classifyCompressionCoveragePosture(totalResponses int64, compressedCount int64) string {
	if totalResponses == 0 {
		return "compression-unobserved"
	}
	if compressedCount == 0 {
		return "compression-bypassed"
	}
	if compressedCount < totalResponses {
		return "compression-partial"
	}
	return "compression-active"
}

func classifyCompressionEfficiencyPosture(totalResponses int64, compressedCount int64, compressionRatio float64, avgDurationMS int64) string {
	if totalResponses == 0 {
		return "compression-unobserved"
	}
	if compressedCount == 0 {
		return "compression-idle"
	}
	if compressionRatio >= 100 {
		return "compression-inefficient"
	}
	if avgDurationMS > 250 {
		return "compression-slow"
	}
	return "compression-efficient"
}

func buildCompressionReliabilityHint(coveragePosture string, efficiencyPosture string) string {
	switch {
	case efficiencyPosture == "compression-inefficient":
		return "compression is active but not reducing payload size; verify thresholds and payload shape"
	case efficiencyPosture == "compression-slow":
		return "compression is active but latency is elevated; verify compression level and payload cost"
	case coveragePosture == "compression-bypassed":
		return "responses are currently bypassing compression; verify payload size and compression policy"
	case coveragePosture == "compression-partial":
		return "compression is applied selectively; verify that partial compression matches payload expectations"
	case coveragePosture == "compression-active":
		return "compression runtime is active and operating within expected efficiency"
	default:
		return "compression runtime has not been observed yet"
	}
}
