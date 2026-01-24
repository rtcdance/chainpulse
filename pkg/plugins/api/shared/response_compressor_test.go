package shared

import (
	"testing"
)

func TestResponseCompressorNoCompression(t *testing.T) {
	compressor := NewResponseCompressor(CompressionNone)

	data := map[string]interface{}{
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
	compressor := NewResponseCompressor(CompressionDefault)

	// Create a large payload to ensure compression
	data := make(map[string]interface{})
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
	compressor := NewResponseCompressor(CompressionDefault)

	data := map[string]interface{}{
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
	compressor := NewResponseCompressor(CompressionDefault)

	data := map[string]interface{}{
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
	compressor := NewResponseCompressor(CompressionDefault)

	data := make(map[string]interface{})
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
	compressor := NewResponseCompressor(CompressionDefault)

	// Create a large payload with repetitive data
	data := make(map[string]interface{})
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
	levels := []CompressionLevel{
		CompressionNone,
		CompressionFast,
		CompressionDefault,
		CompressionBest,
	}

	data := make(map[string]interface{})
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
