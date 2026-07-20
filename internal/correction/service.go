package correction

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	RetentionDays   = 90
	maxDetailsRunes = 1000
	minDetailsRunes = 10
	maxURLLength    = 500
	maxContactRunes = 254
)

var (
	ErrInvalidSubmission = errors.New("invalid correction submission")
	ErrStoreUnavailable  = errors.New("correction store unavailable")
	facilityIDPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,63}$`)
)

type Category string

const (
	CategoryHours   Category = "hours"
	CategoryPrice   Category = "price"
	CategoryRules   Category = "rules"
	CategoryClosure Category = "closure"
	CategoryAccess  Category = "access"
	CategoryOther   Category = "other"
)

type Submission struct {
	FacilityID     string   `json:"facilityId"`
	Category       Category `json:"category"`
	Details        string   `json:"details"`
	EvidenceURL    string   `json:"evidenceUrl,omitempty"`
	Contact        string   `json:"contact,omitempty"`
	ContactConsent bool     `json:"contactConsent,omitempty"`
}

type Receipt struct {
	ReportID   string    `json:"reportId"`
	ReceivedAt time.Time `json:"receivedAt"`
	NextAction string    `json:"nextAction"`
}

type Report struct {
	ReportID       string    `json:"reportId"`
	FacilityID     string    `json:"facilityId"`
	Category       Category  `json:"category"`
	Details        string    `json:"details"`
	EvidenceURL    string    `json:"evidenceUrl,omitempty"`
	Contact        string    `json:"contact,omitempty"`
	ContactConsent bool      `json:"contactConsent,omitempty"`
	ReceivedAt     time.Time `json:"receivedAt"`
	DeleteAfter    time.Time `json:"deleteAfter"`
}

type Store interface {
	Save(context.Context, Report) error
}

// RetentionStore is a correction store that can enforce the report retention deadline.
type RetentionStore interface {
	Store
	PurgeExpired(time.Time) error
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store, now func() time.Time) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: store is required", ErrStoreUnavailable)
	}
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, now: now}, nil
}

func (service *Service) Submit(ctx context.Context, submission Submission) (Receipt, error) {
	normalized, err := normalizeSubmission(submission)
	if err != nil {
		return Receipt{}, err
	}
	receivedAt := service.now().UTC().Truncate(time.Second)
	reportID, err := newReportID()
	if err != nil {
		return Receipt{}, fmt.Errorf("%w: generate report id", ErrStoreUnavailable)
	}
	report := Report{
		ReportID:       reportID,
		FacilityID:     normalized.FacilityID,
		Category:       normalized.Category,
		Details:        normalized.Details,
		EvidenceURL:    normalized.EvidenceURL,
		Contact:        normalized.Contact,
		ContactConsent: normalized.ContactConsent,
		ReceivedAt:     receivedAt,
		DeleteAfter:    receivedAt.AddDate(0, 0, RetentionDays),
	}
	if err := service.store.Save(ctx, report); err != nil {
		return Receipt{}, fmt.Errorf("%w: save report", ErrStoreUnavailable)
	}
	return Receipt{
		ReportID:   reportID,
		ReceivedAt: receivedAt,
		NextAction: "施設の公式情報と照合し、必要な場合はカタログを更新します。",
	}, nil
}

func normalizeSubmission(submission Submission) (Submission, error) {
	submission.FacilityID = strings.TrimSpace(submission.FacilityID)
	submission.Details = strings.TrimSpace(submission.Details)
	submission.EvidenceURL = strings.TrimSpace(submission.EvidenceURL)
	submission.Contact = strings.TrimSpace(submission.Contact)

	if !facilityIDPattern.MatchString(submission.FacilityID) {
		return Submission{}, invalidField("facilityId")
	}
	if !validCategory(submission.Category) {
		return Submission{}, invalidField("category")
	}
	detailsLength := utf8.RuneCountInString(submission.Details)
	if detailsLength < minDetailsRunes || detailsLength > maxDetailsRunes || strings.ContainsRune(submission.Details, '\x00') {
		return Submission{}, invalidField("details")
	}
	if submission.EvidenceURL != "" {
		if len(submission.EvidenceURL) > maxURLLength {
			return Submission{}, invalidField("evidenceUrl")
		}
		parsedURL, err := url.ParseRequestURI(submission.EvidenceURL)
		if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
			return Submission{}, invalidField("evidenceUrl")
		}
	}
	if submission.Contact != "" {
		if utf8.RuneCountInString(submission.Contact) > maxContactRunes || !submission.ContactConsent {
			return Submission{}, invalidField("contact")
		}
		parsedAddress, err := mail.ParseAddress(submission.Contact)
		if err != nil || !strings.EqualFold(parsedAddress.Address, submission.Contact) {
			return Submission{}, invalidField("contact")
		}
	} else {
		submission.ContactConsent = false
	}
	return submission, nil
}

func validCategory(category Category) bool {
	switch category {
	case CategoryHours, CategoryPrice, CategoryRules, CategoryClosure, CategoryAccess, CategoryOther:
		return true
	default:
		return false
	}
}

func invalidField(field string) error {
	return fmt.Errorf("%w: %s", ErrInvalidSubmission, field)
}

func newReportID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "COR-" + hex.EncodeToString(value[:]), nil
}
