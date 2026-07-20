package correction

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunCheckCommandReportsAggregateValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	payload := `{"reportId":"current","deleteAfter":"2026-07-20T12:00:00Z"}` + "\n"
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := RunCheckCommand(
		[]string{"-path", path, "-as-of", "2026-07-19T12:00:00Z"},
		&stdout,
		&stderr,
		func() time.Time { return time.Time{} },
	)
	if exitCode != 0 {
		t.Fatalf("RunCheckCommand() exit = %d, stderr = %s", exitCode, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, `"status":"valid"`) || !strings.Contains(got, `"reportCount":1`) {
		t.Fatalf("RunCheckCommand() output = %s", got)
	}
}

func TestRunCheckCommandDoesNotPrintCorruptContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	if err := os.WriteFile(path, []byte("{\"deleteAfter\":\"PRIVATE-CORRUPT-CONTENT\"}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	var stderr bytes.Buffer
	if exitCode := RunCheckCommand([]string{"-path", path}, io.Discard, &stderr, time.Now); exitCode != 1 {
		t.Fatalf("RunCheckCommand() exit = %d, want 1", exitCode)
	}
	if strings.Contains(stderr.String(), "PRIVATE-CORRUPT-CONTENT") {
		t.Fatalf("RunCheckCommand() exposed corrupt content: %s", stderr.String())
	}
}
