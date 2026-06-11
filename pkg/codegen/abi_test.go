package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSolToGo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		solType string
		goType  string
	}{
		{"address", "address", "common.Address"},
		{"bytes32", "bytes32", "[32]byte"},
		{"string", "string", "string"},
		{"bool", "bool", "bool"},
		{"uint256", "uint256", "*big.Int"},
		{"int256", "int256", "*big.Int"},
		{"uint128", "uint128", "*big.Int"},
		{"int128", "int128", "*big.Int"},
		{"uint64", "uint64", "uint64"},
		{"uint32", "uint32", "uint32"},
		{"int64", "int64", "int64"},
		{"bytes", "bytes", "[]byte"},
		{"bytes4", "bytes4", "[]byte"},
		{"bytes_dynamic", "bytes32[]", "[]byte"},
		{"unknown", "mystery", "interface{}"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := solToGo(tc.solType); got != tc.goType {
				t.Errorf("solToGo(%q) = %q, want %q", tc.solType, got, tc.goType)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		output string
	}{
		{"simple", "transfer", "Transfer"},
		{"snake_case", "transfer_event", "TransferEvent"},
		{"multi_word", "erc20_transfer_event", "Erc20TransferEvent"},
		{"already_pascal", "TransferEvent", "TransferEvent"},
		{"lowercase_start", "transfer", "Transfer"},
		{"single_letter", "t", "T"},
		{"empty", "", ""},
		{"with_numbers", "event_v2", "EventV2"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := toPascalCase(tc.input); got != tc.output {
				t.Errorf("toPascalCase(%q) = %q, want %q", tc.input, got, tc.output)
			}
		})
	}
}

func TestGenerateABIFromFile(t *testing.T) {
	dir := t.TempDir()
	abiPath := filepath.Join(dir, "test.abi.json")
	outPath := filepath.Join(dir, "events.go")

	abiContent := `{"abi":[{"type":"event","name":"Transfer","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}]}]}`
	if err := os.WriteFile(abiPath, []byte(abiContent), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := GenerateABIFromFile(abiPath, outPath, "testpkg"); err != nil {
		t.Fatalf("GenerateABIFromFile failed: %v", err)
	}

	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, "type Transfer struct") {
		t.Error("output should contain Transfer struct")
	}
	if !strings.Contains(content, "package testpkg") {
		t.Error("output should contain package declaration")
	}
}

func TestGenerateABIFromFile_FileNotFound(t *testing.T) {
	err := GenerateABIFromFile("/nonexistent/path.abi.json", "/tmp/out.go", "pkg")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestGenerateABIFromFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	abiPath := filepath.Join(dir, "bad.abi.json")
	outPath := filepath.Join(dir, "events.go")

	if err := os.WriteFile(abiPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := GenerateABIFromFile(abiPath, outPath, "pkg")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestGenerateABIFromFile_NoEvents(t *testing.T) {
	dir := t.TempDir()
	abiPath := filepath.Join(dir, "noevents.abi.json")
	outPath := filepath.Join(dir, "events.go")

	abiContent := `{"abi":[{"type":"function","name":"transfer","inputs":[]}]}`
	if err := os.WriteFile(abiPath, []byte(abiContent), 0o644); err != nil {
		t.Fatal(err)
	}

	err := GenerateABIFromFile(abiPath, outPath, "pkg")
	if err == nil {
		t.Fatal("expected error when no events found")
	}
	if !strings.Contains(err.Error(), "no events found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenderGoFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "events.go")

	events := []GenEvent{
		{
			Name: "Transfer",
			Fields: []GenField{
				{Name: "From", GoType: "common.Address", JSONTag: "from"},
				{Name: "To", GoType: "common.Address", JSONTag: "to"},
				{Name: "Value", GoType: "*big.Int", JSONTag: "value"},
			},
		},
	}

	if err := renderGoFile(outPath, "testpkg", events); err != nil {
		t.Fatalf("renderGoFile failed: %v", err)
	}

	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, "package testpkg") {
		t.Error("should contain package")
	}
	if !strings.Contains(content, "type Transfer struct") {
		t.Error("should contain struct")
	}
	if !strings.Contains(content, "DO NOT EDIT") {
		t.Error("should contain codegen header")
	}
}
