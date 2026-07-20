package facility

import (
	"testing"
	"time"
)

func TestIsDynamicInformationFresh(t *testing.T) {
	asOf := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		verifiedAt time.Time
		asOf       time.Time
		want       bool
	}{
		{name: "current", verifiedAt: asOf, asOf: asOf, want: true},
		{name: "within 30 days", verifiedAt: asOf.Add(-29 * 24 * time.Hour), asOf: asOf, want: true},
		{name: "exactly 30 days", verifiedAt: asOf.Add(-DynamicInformationFreshnessWindow), asOf: asOf, want: true},
		{name: "one nanosecond beyond 30 days", verifiedAt: asOf.Add(-DynamicInformationFreshnessWindow - time.Nanosecond), asOf: asOf, want: false},
		{name: "future verification", verifiedAt: asOf.Add(time.Nanosecond), asOf: asOf, want: false},
		{name: "missing verification", verifiedAt: time.Time{}, asOf: asOf, want: false},
		{name: "missing reference", verifiedAt: asOf, asOf: time.Time{}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsDynamicInformationFresh(test.verifiedAt, test.asOf); got != test.want {
				t.Fatalf("IsDynamicInformationFresh() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestIsStableInformationFresh(t *testing.T) {
	asOf := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		verifiedAt time.Time
		want       bool
	}{
		{name: "within 180 days", verifiedAt: asOf.Add(-179 * 24 * time.Hour), want: true},
		{name: "exactly 180 days", verifiedAt: asOf.Add(-StableInformationFreshnessWindow), want: true},
		{name: "one nanosecond beyond 180 days", verifiedAt: asOf.Add(-StableInformationFreshnessWindow - time.Nanosecond), want: false},
		{name: "future verification", verifiedAt: asOf.Add(time.Nanosecond), want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsStableInformationFresh(test.verifiedAt, asOf); got != test.want {
				t.Fatalf("IsStableInformationFresh() = %t, want %t", got, test.want)
			}
		})
	}
}
