package correction

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// FileValidation is intentionally aggregate-only so diagnostic output never
// exposes correction text, evidence URLs, contact details, or report IDs.
type FileValidation struct {
	ReportCount  int `json:"reportCount"`
	ExpiredCount int `json:"expiredCount"`
}

// ValidateFile checks whether the JSON Lines file can be read by the runtime
// without mutating it.
func ValidateFile(path string, now time.Time) (FileValidation, error) {
	if path == "" {
		return FileValidation{}, fmt.Errorf("%w: path is required", ErrStoreUnavailable)
	}
	info, err := os.Stat(path)
	if err != nil {
		return FileValidation{}, fmt.Errorf("inspect correction store: %w", err)
	}
	if info.Size() > maxStoreBytes {
		return FileValidation{}, fmt.Errorf("correction store exceeds the %d-byte capacity", maxStoreBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return FileValidation{}, fmt.Errorf("open correction store: %w", err)
	}
	defer file.Close()

	result := FileValidation{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), storeScannerMaxBytes)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		if len(scanner.Bytes()) > maxStoredReportBytes {
			return FileValidation{}, fmt.Errorf("decode correction store line %d", lineNumber)
		}
		var report Report
		if err := json.Unmarshal(scanner.Bytes(), &report); err != nil || report.DeleteAfter.IsZero() {
			return FileValidation{}, fmt.Errorf("decode correction store line %d", lineNumber)
		}
		result.ReportCount++
		if !report.DeleteAfter.After(now.UTC()) {
			result.ExpiredCount++
		}
	}
	if err := scanner.Err(); err != nil {
		return FileValidation{}, fmt.Errorf("scan correction store near line %d: %w", lineNumber+1, err)
	}
	return result, nil
}
