package chatbot

import (
	"context"
	"testing"
	"time"
)

func TestMemorySlackRequestStoreDeduplicatesAndPurgesDeliveredRequests(t *testing.T) {
	store := NewMemorySlackRequestStore()
	at := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC)
	request := SlackRecommendationRequest{
		RequestID: "request-a", SourceEventKey: "source-a",
		CreatedAt: at, UpdatedAt: at, ExpiresAt: at.Add(time.Hour),
	}
	created, err := store.Begin(context.Background(), request)
	if err != nil || !created {
		t.Fatalf("first Begin() = (%v, %v), want (true, nil)", created, err)
	}
	request.RequestID = "request-retry"
	created, err = store.Begin(context.Background(), request)
	if err != nil || created {
		t.Fatalf("retry Begin() = (%v, %v), want (false, nil)", created, err)
	}
	if err := store.MarkDelivered(context.Background(), "request-a", at); err != nil {
		t.Fatalf("MarkDelivered() error = %v", err)
	}
	if status := store.requests["request-a"].Status; status != SlackRequestDelivered {
		t.Fatalf("request status = %q, want %q", status, SlackRequestDelivered)
	}
	if err := store.PurgeExpired(context.Background(), at.Add(time.Hour)); err != nil {
		t.Fatalf("PurgeExpired() error = %v", err)
	}
	if len(store.requests) != 0 || len(store.sources) != 0 {
		t.Fatalf("expired request state was not purged: requests=%d sources=%d", len(store.requests), len(store.sources))
	}
}
