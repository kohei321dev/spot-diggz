package chatbot

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type SlackRequestStatus string

const (
	SlackRequestGenerating SlackRequestStatus = "generating"
	SlackRequestDelivered  SlackRequestStatus = "delivered"
	SlackRequestFailed     SlackRequestStatus = "failed"
)

var (
	ErrSlackRequestNotFound    = errors.New("Slack recommendation request not found")
	ErrSlackRequestConflict    = errors.New("Slack recommendation request state conflict")
	ErrSlackRequestStoreClosed = errors.New("Slack recommendation request store is closed")
)

type SlackRecommendationRequest struct {
	RequestID      string
	SourceEventKey string
	Status         SlackRequestStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ExpiresAt      time.Time
}

type SlackRequestStore interface {
	Begin(context.Context, SlackRecommendationRequest) (bool, error)
	MarkDelivered(context.Context, string, time.Time) error
	MarkFailed(context.Context, string, time.Time) error
	PurgeExpired(context.Context, time.Time) error
	Close() error
}

type MemorySlackRequestStore struct {
	mu       sync.Mutex
	requests map[string]SlackRecommendationRequest
	sources  map[string]string
	closed   bool
}

func NewMemorySlackRequestStore() *MemorySlackRequestStore {
	return &MemorySlackRequestStore{
		requests: make(map[string]SlackRecommendationRequest),
		sources:  make(map[string]string),
	}
}

func (store *MemorySlackRequestStore) Begin(ctx context.Context, request SlackRecommendationRequest) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.closed {
		return false, ErrSlackRequestStoreClosed
	}
	if _, exists := store.sources[request.SourceEventKey]; exists {
		return false, nil
	}
	request.Status = SlackRequestGenerating
	store.requests[request.RequestID] = request
	store.sources[request.SourceEventKey] = request.RequestID
	return true, nil
}

func (store *MemorySlackRequestStore) MarkDelivered(ctx context.Context, requestID string, at time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	request, err := store.requestForUpdate(requestID)
	if err != nil {
		return err
	}
	if request.Status != SlackRequestGenerating {
		return ErrSlackRequestConflict
	}
	request.Status = SlackRequestDelivered
	request.UpdatedAt = at.UTC()
	store.requests[requestID] = request
	return nil
}

func (store *MemorySlackRequestStore) MarkFailed(ctx context.Context, requestID string, at time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	request, err := store.requestForUpdate(requestID)
	if err != nil {
		return err
	}
	if request.Status != SlackRequestGenerating {
		return ErrSlackRequestConflict
	}
	request.Status = SlackRequestFailed
	request.UpdatedAt = at.UTC()
	store.requests[requestID] = request
	return nil
}

func (store *MemorySlackRequestStore) PurgeExpired(ctx context.Context, at time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.closed {
		return ErrSlackRequestStoreClosed
	}
	for requestID, request := range store.requests {
		if !request.ExpiresAt.After(at.UTC()) {
			delete(store.requests, requestID)
			delete(store.sources, request.SourceEventKey)
		}
	}
	return nil
}

func (store *MemorySlackRequestStore) Close() error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.closed = true
	return nil
}

func (store *MemorySlackRequestStore) requestForUpdate(requestID string) (SlackRecommendationRequest, error) {
	if store.closed {
		return SlackRecommendationRequest{}, ErrSlackRequestStoreClosed
	}
	request, exists := store.requests[requestID]
	if !exists {
		return SlackRecommendationRequest{}, fmt.Errorf("%w: %s", ErrSlackRequestNotFound, requestID)
	}
	return request, nil
}
