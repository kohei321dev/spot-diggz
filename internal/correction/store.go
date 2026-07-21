package correction

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	maxStoredReportBytes = 16 * 1024
	storeScannerMaxBytes = maxStoredReportBytes + 1024
	maxStoreBytes        = 32 * 1024 * 1024
)

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string, now time.Time) (*FileStore, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: path is required", ErrStoreUnavailable)
	}
	store := &FileStore{path: filepath.Clean(path)}
	if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
		return nil, fmt.Errorf("create correction store directory: %w", err)
	}
	if err := store.PurgeExpired(now.UTC()); err != nil {
		return nil, err
	}
	if err := store.ensureWritable(); err != nil {
		return nil, err
	}
	if err := store.ensureWithinCapacity(); err != nil {
		return nil, err
	}
	return store, nil
}

func (store *FileStore) ensureWritable() error {
	store.mu.Lock()
	defer store.mu.Unlock()

	file, err := os.OpenFile(store.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open correction store for writing: %w", err)
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("restrict correction store permissions: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync correction store: %w", err)
	}
	return nil
}

func (store *FileStore) ensureWithinCapacity() error {
	store.mu.Lock()
	defer store.mu.Unlock()
	info, err := os.Stat(store.path)
	if err != nil {
		return fmt.Errorf("inspect correction store capacity: %w", err)
	}
	if info.Size() > maxStoreBytes {
		return fmt.Errorf("correction store exceeds the %d-byte capacity", maxStoreBytes)
	}
	return nil
}

func (store *FileStore) Save(ctx context.Context, report Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if report.DeleteAfter.IsZero() {
		return fmt.Errorf("correction report requires a retention deadline")
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode correction report: %w", err)
	}
	if len(payload) > maxStoredReportBytes {
		return fmt.Errorf("correction report exceeds storage limit")
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	info, err := os.Stat(store.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect correction store capacity: %w", err)
	}
	currentSize := int64(0)
	if err == nil {
		currentSize = info.Size()
	}
	if currentSize+int64(len(payload))+1 > maxStoreBytes {
		return fmt.Errorf("correction store reached the %d-byte capacity", maxStoreBytes)
	}
	file, err := os.OpenFile(store.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("restrict correction store permissions: %w", err)
	}
	if _, err := file.Write(append(payload, '\n')); err != nil {
		return err
	}
	return file.Sync()
}

// PurgeExpired removes reports whose retention deadline has passed.
func (store *FileStore) PurgeExpired(now time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.purgeExpiredLocked(now.UTC())
}

func (store *FileStore) purgeExpiredLocked(now time.Time) error {
	file, err := os.Open(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open correction store: %w", err)
	}
	defer file.Close()

	kept := make([]Report, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), storeScannerMaxBytes)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		if len(scanner.Bytes()) > maxStoredReportBytes {
			return fmt.Errorf("decode correction store line %d", lineNumber)
		}
		var report Report
		if err := json.Unmarshal(scanner.Bytes(), &report); err != nil || report.DeleteAfter.IsZero() {
			return fmt.Errorf("decode correction store line %d", lineNumber)
		}
		if report.DeleteAfter.After(now) {
			kept = append(kept, report)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan correction store near line %d: %w", lineNumber+1, err)
	}

	temporaryPath := store.path + ".tmp"
	temporary, err := os.OpenFile(temporaryPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create correction store temporary file: %w", err)
	}
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("restrict correction store temporary file permissions: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	for _, report := range kept {
		if err := encoder.Encode(report); err != nil {
			temporary.Close()
			return fmt.Errorf("rewrite correction store: %w", err)
		}
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close correction store temporary file: %w", err)
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		return fmt.Errorf("replace correction store: %w", err)
	}
	return nil
}

type MemoryStore struct {
	mu      sync.Mutex
	reports []Report
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (store *MemoryStore) Save(ctx context.Context, report Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.reports = append(store.reports, report)
	return nil
}

func (store *MemoryStore) Reports() []Report {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]Report(nil), store.reports...)
}
