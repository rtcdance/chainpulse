package shared

import (
	"testing"
)

func TestResponseCompressorNoCompression(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionNone)

	data := map[string]any{
		"id":   "123",
		"name": "test",
	}

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("failed to compress: %v", err)
	}

	if compressed == nil {
		t.Fatal("compressed data is nil")
	}
}

func TestResponseCompressorGzipCompression(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionDefault)

	// Create a large payload to ensure compression
	data := make(map[string]any)
	for i := 0; i < 100; i++ {
		data[string(rune(i))] = "this is a test value that should be compressed"
	}

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("failed to compress: %v", err)
	}

	if compressed == nil {
		t.Fatal("compressed data is nil")
	}
}

func TestResponseCompressorDecompress(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionDefault)

	data := map[string]any{
		"id":   "123",
		"name": "test",
	}

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("failed to compress: %v", err)
	}

	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}

	if decompressed == nil {
		t.Fatal("decompressed data is nil")
	}
}

func TestResponseCompressorSmallPayload(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionDefault)

	data := map[string]any{
		"id": "123",
	}

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("failed to compress: %v", err)
	}

	if compressed == nil {
		t.Fatal("compressed data is nil")
	}
}

func TestResponseCompressorMetrics(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionDefault)

	data := make(map[string]any)
	for i := 0; i < 50; i++ {
		data[string(rune(i))] = "test value"
	}

	_, _ = compressor.Compress(data)
	_, _ = compressor.Compress(data)

	metrics := compressor.GetMetrics()
	if metrics["total_responses"].(int64) != 2 {
		t.Errorf("expected 2 total responses, got %v", metrics["total_responses"])
	}

	if metrics["original_size"].(int64) <= 0 {
		t.Errorf("expected positive original_size, got %v", metrics["original_size"])
	}

	if metrics["compressed_size"].(int64) <= 0 {
		t.Errorf("expected positive compressed_size, got %v", metrics["compressed_size"])
	}
}

func TestResponseCompressorCompressionRatio(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionDefault)

	// Create a large payload with repetitive data
	data := make(map[string]any)
	for i := 0; i < 100; i++ {
		data[string(rune(i))] = "this is a test value that should compress well"
	}

	_, _ = compressor.Compress(data)

	metrics := compressor.GetMetrics()
	ratio := metrics["compression_ratio"].(float64)

	// Compression ratio should be less than 100% for repetitive data
	if ratio >= 100 {
		t.Errorf("expected compression ratio < 100%%, got %v%%", ratio)
	}
}

func TestResponseCompressorCompressionLevels(t *testing.T) {
	t.Parallel()
	levels := []CompressionLevel{
		CompressionNone,
		CompressionFast,
		CompressionDefault,
		CompressionBest,
	}

	data := make(map[string]any)
	for i := 0; i < 50; i++ {
		data[string(rune(i))] = "test value"
	}

	for _, level := range levels {
		compressor := NewResponseCompressor(level)
		_, err := compressor.Compress(data)
		if err != nil {
			t.Errorf("failed to compress with level %v: %v", level, err)
		}
	}
}

func TestResponseCompressorDecompressUncompressed(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionDefault)

	// Try to decompress uncompressed data
	uncompressed := []byte(`{"id":"123","name":"test"}`)
	decompressed, err := compressor.Decompress(uncompressed)
	if err != nil {
		t.Fatalf("failed to decompress uncompressed data: %v", err)
	}

	if decompressed == nil {
		t.Fatal("decompressed data is nil")
	}
}

func TestResponseCompressorRuntimeMetricsUnobserved(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionDefault)

	metrics := compressor.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "compression-unobserved" {
		t.Fatalf("expected compression-unobserved, got %v", metrics["coverage_posture"])
	}
	if metrics["efficiency_posture"] != "compression-unobserved" {
		t.Fatalf("expected compression-unobserved efficiency, got %v", metrics["efficiency_posture"])
	}
}

func TestResponseCompressorMetricsIncludesPostureFields(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionDefault)

	data := make(map[string]any)
	for i := 0; i < 120; i++ {
		data[string(rune(i))] = "this is a repeated test value that should compress well"
	}

	_, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("failed to compress payload: %v", err)
	}

	metrics := compressor.GetMetrics()
	if metrics["coverage_posture"] != "compression-active" {
		t.Fatalf("expected compression-active, got %v", metrics["coverage_posture"])
	}
	if metrics["efficiency_posture"] != "compression-efficient" {
		t.Fatalf("expected compression-efficient, got %v", metrics["efficiency_posture"])
	}
	if metrics["reliability_hint"] != "compression runtime is active and operating within expected efficiency" {
		t.Fatalf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestResponseCompressorRuntimeMetricsBypassed(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionDefault)

	_, err := compressor.Compress(map[string]any{"id": "123"})
	if err != nil {
		t.Fatalf("failed to compress small payload: %v", err)
	}

	metrics := compressor.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "compression-bypassed" {
		t.Fatalf("expected compression-bypassed, got %v", metrics["coverage_posture"])
	}
	if metrics["efficiency_posture"] != "compression-idle" {
		t.Fatalf("expected compression-idle, got %v", metrics["efficiency_posture"])
	}
}

func TestResponseCompressorRuntimeMetricsEfficient(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionDefault)

	data := make(map[string]any)
	for i := 0; i < 120; i++ {
		data[string(rune(i))] = "this is a repeated test value that should compress well"
	}

	_, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("failed to compress payload: %v", err)
	}

	metrics := compressor.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "compression-active" {
		t.Fatalf("expected compression-active, got %v", metrics["coverage_posture"])
	}
	if metrics["efficiency_posture"] != "compression-efficient" {
		t.Fatalf("expected compression-efficient, got %v", metrics["efficiency_posture"])
	}
}

func TestResponseCompressorDecompressNoCompression(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionNone)

	data := []byte(`{"id":"123"}`)
	decompressed, err := compressor.Decompress(data)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}
	if string(decompressed) != `{"id":"123"}` {
		t.Errorf("unexpected decompressed data: %s", string(decompressed))
	}
}

func TestResponseCompressorCompressFast(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionFast)

	data := make(map[string]any)
	for i := 0; i < 50; i++ {
		data[string(rune(i))] = "test value"
	}

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("failed to compress with fast level: %v", err)
	}
	if compressed == nil {
		t.Fatal("compressed data is nil")
	}
}

func TestResponseCompressorCompressBest(t *testing.T) {
	t.Parallel()
	compressor := NewResponseCompressor(CompressionBest)

	data := make(map[string]any)
	for i := 0; i < 50; i++ {
		data[string(rune(i))] = "test value"
	}

	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("failed to compress with best level: %v", err)
	}
	if compressed == nil {
		t.Fatal("compressed data is nil")
	}
}
