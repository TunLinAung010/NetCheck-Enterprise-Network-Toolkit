package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/netcheck/netcheck/pkg/logger"
)

type Exportable interface {
	ToMap() map[string]interface{}
	Headers() []string
	Row() []string
}

type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
	FormatHTML ExportFormat = "html"
)

type Exporter struct {
	format   ExportFormat
	filePath string
	data     []Exportable
}

func NewExporter(format ExportFormat, filePath string) *Exporter {
	return &Exporter{
		format:   format,
		filePath: filePath,
	}
}

func (e *Exporter) Add(data Exportable) {
	e.data = append(e.data, data)
}

func (e *Exporter) Export() error {
	switch e.format {
	case FormatJSON:
		return e.exportJSON()
	case FormatCSV:
		return e.exportCSV()
	case FormatHTML:
		return e.exportHTML()
	default:
		return fmt.Errorf("unsupported format: %s", e.format)
	}
}

func (e *Exporter) exportJSON() error {
	entries := make([]map[string]interface{}, len(e.data))
	for i, d := range e.data {
		entries[i] = d.ToMap()
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if e.filePath != "" {
		if err := os.WriteFile(e.filePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		logger.Info("exported JSON to %s", e.filePath)
	} else {
		fmt.Println(string(data))
	}

	return nil
}

func (e *Exporter) exportCSV() error {
	if len(e.data) == 0 {
		return fmt.Errorf("no data to export")
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if err := writer.Write(e.data[0].Headers()); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, d := range e.data {
		if err := writer.Write(d.Row()); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("CSV writer error: %w", err)
	}

	if e.filePath != "" {
		if err := os.WriteFile(e.filePath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		logger.Info("exported CSV to %s", e.filePath)
	} else {
		fmt.Print(buf.String())
	}

	return nil
}

func (e *Exporter) exportHTML() error {
	if len(e.data) == 0 {
		return fmt.Errorf("no data to export")
	}

	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>NetCheck Report - {{.Timestamp}}</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 20px; background: #f5f5f5; }
  h1 { color: #333; }
  table { border-collapse: collapse; width: 100%; background: #fff; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
  th, td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #ddd; }
  th { background: #4a90d9; color: white; }
  tr:hover { background: #f0f4ff; }
  .timestamp { color: #666; margin-bottom: 20px; }
  .open { color: green; font-weight: bold; }
  .closed { color: red; }
  .filtered { color: orange; }
</style>
</head>
<body>
<h1>NetCheck Report</h1>
<div class="timestamp">Generated: {{.Timestamp}}</div>
<table>
<thead><tr>
{{range .Headers}}<th>{{.}}</th>{{end}}
</tr></thead>
<tbody>
{{range .Rows}}<tr>
{{range .}}<td>{{.}}</td>{{end}}
</tr>{{end}}
</tbody>
</table>
</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	headers := e.data[0].Headers()
	var rows [][]string
	for _, d := range e.data {
		rows = append(rows, d.Row())
	}

	data := struct {
		Timestamp string
		Headers   []string
		Rows      [][]string
	}{
		Timestamp: time.Now().Format(time.RFC3339),
		Headers:   headers,
		Rows:      rows,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	if e.filePath != "" {
		if err := os.WriteFile(e.filePath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		logger.Info("exported HTML to %s", e.filePath)
	} else {
		fmt.Print(buf.String())
	}

	return nil
}

func DetectFormat(path string) ExportFormat {
	switch {
	case hasSuffix(path, ".json"):
		return FormatJSON
	case hasSuffix(path, ".csv"):
		return FormatCSV
	case hasSuffix(path, ".html"), hasSuffix(path, ".htm"):
		return FormatHTML
	default:
		return ""
	}
}

func hasSuffix(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
