package export

import (
	"os"
	"testing"
)

type testExportable struct{}

func (t testExportable) ToMap() map[string]interface{} {
	return map[string]interface{}{"key": "value"}
}

func (t testExportable) Headers() []string {
	return []string{"Key"}
}

func (t testExportable) Row() []string {
	return []string{"value"}
}

func TestNewExporter(t *testing.T) {
	e := NewExporter(FormatJSON, "")
	if e == nil {
		t.Fatal("expected non-nil exporter")
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		path string
		fmt  ExportFormat
	}{
		{"result.json", FormatJSON},
		{"result.csv", FormatCSV},
		{"result.html", FormatHTML},
		{"result.htm", FormatHTML},
		{"result.xml", ""},
	}
	for _, tt := range tests {
		f := DetectFormat(tt.path)
		if f != tt.fmt {
			t.Errorf("DetectFormat(%s) = %s, want %s", tt.path, f, tt.fmt)
		}
	}
}

func TestJSONExport(t *testing.T) {
	tmpFile := "test_export.json"
	defer os.Remove(tmpFile)

	e := NewExporter(FormatJSON, tmpFile)
	e.Add(testExportable{})
	if err := e.Export(); err != nil {
		t.Fatalf("JSON export failed: %v", err)
	}
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read export: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty export")
	}
}

func TestCSVExport(t *testing.T) {
	tmpFile := "test_export.csv"
	defer os.Remove(tmpFile)

	e := NewExporter(FormatCSV, tmpFile)
	e.Add(testExportable{})
	if err := e.Export(); err != nil {
		t.Fatalf("CSV export failed: %v", err)
	}
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read export: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty export")
	}
}
