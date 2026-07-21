package correction

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServiceStoresValidatedReportWithRetention(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	service, err := NewService(store, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	receipt, err := service.Submit(context.Background(), Submission{
		FacilityID:     "OSK-F001",
		Category:       CategoryHours,
		Details:        "公式ページの営業時間と表示内容が異なります。",
		EvidenceURL:    "https://example.com/official",
		Contact:        "skater@example.com",
		ContactConsent: true,
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if receipt.ReportID == "" || len(store.Reports()) != 1 {
		t.Fatalf("receipt = %#v, reports = %#v", receipt, store.Reports())
	}
	wantDeleteAfter := now.AddDate(0, 0, RetentionDays)
	if !store.Reports()[0].DeleteAfter.Equal(wantDeleteAfter) {
		t.Fatalf("DeleteAfter = %s, want %s", store.Reports()[0].DeleteAfter, wantDeleteAfter)
	}
}

func TestServiceRejectsInvalidReports(t *testing.T) {
	service, err := NewService(NewMemoryStore(), time.Now)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	tests := []Submission{
		{FacilityID: "OSK-F001", Category: CategoryHours, Details: "短い"},
		{FacilityID: "OSK-F001", Category: "unknown", Details: "十分な長さの訂正内容です。"},
		{FacilityID: "OSK-F001", Category: CategoryRules, Details: "十分な長さの訂正内容です。", EvidenceURL: "http://example.com"},
		{FacilityID: "OSK-F001", Category: CategoryAccess, Details: "十分な長さの訂正内容です。", Contact: "user@example.com"},
	}
	for _, submission := range tests {
		if _, err := service.Submit(context.Background(), submission); err == nil {
			t.Fatalf("Submit(%#v) error = nil", submission)
		}
	}
}

func TestFileStorePurgesExpiredReports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	now := time.Date(2026, time.July, 19, 0, 0, 0, 0, time.UTC)
	payload := `{"reportId":"old","facilityId":"OSK-F001","category":"hours","details":"expired report","receivedAt":"2026-01-01T00:00:00Z","deleteAfter":"2026-04-01T00:00:00Z"}` + "\n"
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	store, err := NewFileStore(path, now)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if err := store.Save(context.Background(), Report{ReportID: "new", DeleteAfter: now.Add(time.Hour)}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) == payload || !contains(string(content), `"reportId":"new"`) {
		t.Fatalf("store content = %s", content)
	}
}

func TestFileStorePurgesExpiredReportsAfterStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	now := time.Date(2026, time.July, 19, 0, 0, 0, 0, time.UTC)
	store, err := NewFileStore(path, now)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if err := store.Save(context.Background(), Report{ReportID: "expires", DeleteAfter: now.Add(time.Hour)}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := store.PurgeExpired(now.Add(2 * time.Hour)); err != nil {
		t.Fatalf("PurgeExpired() error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(content) != 0 {
		t.Fatalf("store content = %s, want empty", content)
	}
}

func TestFileStoreRestrictsExistingFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := NewFileStore(path, time.Now())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("permissions = %o, want 600", got)
	}
}

func TestNewFileStoreCreatesWritableFileAtStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	if _, err := NewFileStore(path, time.Now()); err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("permissions = %o, want 600", got)
	}
}

func TestFileStoreRejectsWritesBeyondCapacity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	store, err := NewFileStore(path, time.Now())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if err := os.Truncate(path, maxStoreBytes); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}

	err = store.Save(context.Background(), Report{ReportID: "over-capacity", DeleteAfter: time.Now().Add(time.Hour)})
	if err == nil {
		t.Fatal("Save() error = nil, want capacity error")
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatalf("Stat() error = %v", statErr)
	}
	if info.Size() != maxStoreBytes {
		t.Fatalf("store size = %d, want %d", info.Size(), maxStoreBytes)
	}
}

func TestNewFileStoreRejectsExistingFileBeyondCapacity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	if err := file.Truncate(maxStoreBytes + 1); err != nil {
		file.Close()
		t.Fatalf("Truncate() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := NewFileStore(path, time.Now()); err == nil {
		t.Fatal("NewFileStore() error = nil, want capacity error")
	}
}

func TestNewFileStoreFailsWhenStorePathIsNotWritableFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if _, err := NewFileStore(path, time.Now()); err == nil {
		t.Fatal("NewFileStore() error = nil, want startup failure")
	}
}

func TestNewFileStoreDoesNotExposeCorruptReportContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrections.jsonl")
	privateValue := "PRIVATE-CORRUPT-CONTENT"
	payload := `{"deleteAfter":"` + privateValue + `"}` + "\n"
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := NewFileStore(path, time.Now())
	if err == nil {
		t.Fatal("NewFileStore() error = nil, want corrupt-report error")
	}
	if contains(err.Error(), privateValue) {
		t.Fatalf("NewFileStore() exposed corrupt report content: %v", err)
	}
}

func contains(value string, target string) bool {
	for index := 0; index+len(target) <= len(value); index++ {
		if value[index:index+len(target)] == target {
			return true
		}
	}
	return false
}
