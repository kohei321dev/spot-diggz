package spot

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var SdzErrNotFound = errors.New("spot not found")

type SdzStore interface {
	Create(ctx context.Context, input SdzCreateSpotInput) (SdzSpot, error)
	Get(ctx context.Context, spotID string) (SdzSpot, error)
	List(ctx context.Context, filter SdzListFilter) ([]SdzSpot, error)
	Update(ctx context.Context, spotID string, input SdzUpdateSpotInput) (SdzSpot, error)
	Delete(ctx context.Context, spotID string) error
}

type SdzMemoryStore struct {
	mu    sync.RWMutex
	spots map[string]SdzSpot
	now   func() time.Time
}

func NewSdzMemoryStore() *SdzMemoryStore {
	return &SdzMemoryStore{
		spots: map[string]SdzSpot{},
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *SdzMemoryStore) Create(_ context.Context, input SdzCreateSpotInput) (SdzSpot, error) {
	id, err := NewSdzID()
	if err != nil {
		return SdzSpot{}, err
	}
	now := s.now()
	created, err := NewSdzSpot(id, input, now)
	if err != nil {
		return SdzSpot{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.spots[created.SdzSpotID] = created
	return created, nil
}

func (s *SdzMemoryStore) Get(_ context.Context, spotID string) (SdzSpot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	found, ok := s.spots[spotID]
	if !ok || found.DeletedAt != nil {
		return SdzSpot{}, SdzErrNotFound
	}
	return found, nil
}

func (s *SdzMemoryStore) List(_ context.Context, filter SdzListFilter) ([]SdzSpot, error) {
	tags := normalizeTags(filter.Tags)

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SdzSpot, 0, len(s.spots))
	for _, item := range s.spots {
		if item.DeletedAt != nil {
			continue
		}
		if filter.SdzVisibility != nil && item.SdzVisibility != *filter.SdzVisibility {
			continue
		}
		if filter.SdzBBox != nil && !filter.SdzBBox.Contains(item.SdzLocation) {
			continue
		}
		if len(tags) > 0 && !hasAllTags(item.Tags, tags) {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *SdzMemoryStore) Update(_ context.Context, spotID string, input SdzUpdateSpotInput) (SdzSpot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.spots[spotID]
	if !ok || current.DeletedAt != nil {
		return SdzSpot{}, SdzErrNotFound
	}
	updated, err := SdzApplyUpdate(current, input, s.now())
	if err != nil {
		return SdzSpot{}, err
	}
	s.spots[spotID] = updated
	return updated, nil
}

func (s *SdzMemoryStore) Delete(_ context.Context, spotID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.spots[spotID]
	if !ok || current.DeletedAt != nil {
		return SdzErrNotFound
	}
	now := s.now()
	current.DeletedAt = &now
	current.UpdatedAt = now
	s.spots[spotID] = current
	return nil
}

func hasAllTags(candidate []string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, tag := range normalizeTags(candidate) {
		set[stringsKey(tag)] = struct{}{}
	}
	for _, tag := range required {
		if _, ok := set[stringsKey(tag)]; !ok {
			return false
		}
	}
	return true
}

func stringsKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
