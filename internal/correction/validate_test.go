package correction

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateFileReturnsAggregateCounts(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	payload := strings.Join([]string{
		`{"reportId":"current","deleteAfter":"2026-07-20T12:00:00Z"}`,
		`{"reportId":"expired","deleteAfter":"2026-07-18T12:00:00Z"}`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := ValidateFile(path, now)
	if err != nil {
		t.Fatalf("ValidateFile() error = %v", err)
	}
	if result.ReportCount != 2 || result.ExpiredCount != 1 {
		t.Fatalf("ValidateFile() = %#v", result)
	}
}

func TestValidateFileReportsCorruptLineWithoutItsContents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	payload := "{\"reportId\":\"valid\",\"deleteAfter\":\"2026-07-20T12:00:00Z\"}\n{\"deleteAfter\":\"PRIVATE-CORRUPT-CONTENT\"}\n"
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := ValidateFile(path, time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("ValidateFile() error = %v, want line 2", err)
	}
	if strings.Contains(err.Error(), "PRIVATE-CORRUPT-CONTENT") {
		t.Fatalf("ValidateFile() exposed corrupt content: %v", err)
	}
}

func TestValidateFileRejectsOversizedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxStoredReportBytes+1)), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := ValidateFile(path, time.Now()); err == nil {
		t.Fatal("ValidateFile() error = nil, want oversized-line error")
	}
}

func TestValidateFileRejectsReportWithoutRetentionDeadline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	if err := os.WriteFile(path, []byte("{\"reportId\":\"missing-deadline\"}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := ValidateFile(path, time.Now()); err == nil {
		t.Fatal("ValidateFile() error = nil, want missing-deadline error")
	}
}
