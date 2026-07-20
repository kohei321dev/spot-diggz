package facility

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type candidateLedger struct {
	AsOf        string   `json:"asOf"`
	Scope       []string `json:"scope"`
	Publication struct {
		PublishedCatalog string `json:"publishedCatalog"`
		PublishedCount   int    `json:"publishedCount"`
		Policy           string `json:"policy"`
	} `json:"publication"`
	HeldCandidates []struct {
		CandidateID       string   `json:"candidateId"`
		Name              string   `json:"name"`
		Prefecture        string   `json:"prefecture"`
		Municipality      string   `json:"municipality"`
		Existence         string   `json:"existence"`
		PublicationStatus string   `json:"publicationStatus"`
		Blockers          []string `json:"blockers"`
		SourceURLs        []string `json:"sourceUrls"`
	} `json:"heldCandidates"`
}

func TestCandidateLedgerMatchesPublishedCatalogAndHasEvidence(t *testing.T) {
	root := repositoryRoot(t)
	ledgerFile, err := os.Open(filepath.Join(root, "data", "facility-candidates.json"))
	if err != nil {
		t.Fatalf("open candidate ledger: %v", err)
	}
	defer ledgerFile.Close()

	var ledger candidateLedger
	decoder := json.NewDecoder(ledgerFile)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ledger); err != nil {
		t.Fatalf("decode candidate ledger: %v", err)
	}
	if ledger.AsOf != "2026-07-19" {
		t.Fatalf("asOf = %q, want 2026-07-19", ledger.AsOf)
	}
	if len(ledger.Scope) != 5 {
		t.Fatalf("scope count = %d, want 5", len(ledger.Scope))
	}
	if len(ledger.HeldCandidates) != 13 {
		t.Fatalf("held candidate count = %d, want 13", len(ledger.HeldCandidates))
	}

	catalog, err := LoadCatalogFile(filepath.Join(root, ledger.Publication.PublishedCatalog))
	if err != nil {
		t.Fatalf("LoadCatalogFile() error = %v", err)
	}
	if publishedCount := len(catalog.List("")); publishedCount != ledger.Publication.PublishedCount {
		t.Fatalf("published count = %d, ledger says %d", publishedCount, ledger.Publication.PublishedCount)
	}

	seenIDs := make(map[string]bool, len(ledger.HeldCandidates))
	for _, candidate := range ledger.HeldCandidates {
		if candidate.CandidateID == "" || candidate.Name == "" || candidate.Prefecture == "" || candidate.Municipality == "" {
			t.Fatalf("candidate has a missing identity field: %#v", candidate)
		}
		if seenIDs[candidate.CandidateID] {
			t.Fatalf("duplicate candidate ID %s", candidate.CandidateID)
		}
		seenIDs[candidate.CandidateID] = true
		if candidate.Existence != "confirmed" || candidate.PublicationStatus != "held" || len(candidate.Blockers) == 0 || len(candidate.SourceURLs) == 0 {
			t.Fatalf("candidate %s lacks publication evidence", candidate.CandidateID)
		}
		for _, sourceURL := range candidate.SourceURLs {
			parsed, err := url.ParseRequestURI(sourceURL)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
				t.Fatalf("candidate %s has invalid source URL %q", candidate.CandidateID, sourceURL)
			}
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not resolve the test file")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..")
}
