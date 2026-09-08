package botsearch

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/nearby"
)

func unitOKResponse(t *testing.T) nearby.Response {
	t.Helper()
	catalog, err := facility.LoadCatalogFile("../../testdata/facilities.dev.json")
	if err != nil {
		t.Fatal("synthetic catalog unavailable")
	}
	spot := catalog.List("skateboard")[0]
	spot.Genre = facility.GenreSkatepark
	spot.SkatingPermissionSourceURL = "https://example.com/permission"
	result := unitEmptyResponse()
	result.Status = nearby.StatusOK
	result.Matches = []nearby.Match{{Facility: spot, DistanceKm: 0}}
	return result
}

func unitResponseJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal("synthetic response encoding failed")
	}
	return data
}

func TestResponsePreservesCompleteFacilityAndZeroDistance(t *testing.T) {
	want := unitOKResponse(t)
	got, err := decodeResponse(unitResponseJSON(t, want), unitInput())
	if err != nil || string(unitResponseJSON(t, got)) != string(unitResponseJSON(t, want)) {
		t.Fatal("valid facility facts or zero distance were changed")
	}
}

func TestResponseRejectsMissingFactsAndContradictoryCandidates(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(map[string]any, map[string]any, map[string]any)
	}{
		{"missing location", func(_, _, f map[string]any) { delete(f, "location") }},
		{"null location", func(_, _, f map[string]any) { f["location"] = nil }},
		{"missing latitude", func(_, _, f map[string]any) { delete(f["location"].(map[string]any), "latitude") }},
		{"null longitude", func(_, _, f map[string]any) { f["location"].(map[string]any)["longitude"] = nil }},
		{"missing boolean", func(_, _, f map[string]any) { delete(f, "beginnerFriendly") }},
		{"null schedule notes", func(_, _, f map[string]any) { f["englishTranslation"].(map[string]any)["scheduleNotes"] = nil }},
		{"null boolean", func(_, _, f map[string]any) { f["beginnerFriendly"] = nil }},
		{"missing distance", func(_, m, _ map[string]any) { delete(m, "distanceKm") }},
		{"null distance", func(_, m, _ map[string]any) { m["distanceKm"] = nil }},
		{"negative distance", func(_, m, _ map[string]any) { m["distanceKm"] = -0.1 }},
		{"outside radius", func(_, m, _ map[string]any) { m["distanceKm"] = 10.01 }},
		{"null facility", func(_, m, _ map[string]any) { m["facility"] = nil }},
		{"unknown field", func(_, _, f map[string]any) { f["query"] = "synthetic private input" }},
		{"non skate activity", func(_, _, f map[string]any) { f["activities"] = []string{"basketball"} }},
		{"unclassified", func(_, _, f map[string]any) { delete(f, "genre") }},
		{"unknown hours", func(_, _, f map[string]any) { f["hoursStatus"] = "unknown" }},
		{"schedule check", func(_, _, f map[string]any) { f["generalUseStatus"] = "schedule_check_required" }},
		{"unverified", func(_, _, f map[string]any) { f["status"] = "candidate" }},
		{"missing evidence", func(_, _, f map[string]any) { delete(f, "skatingPermissionSourceUrl") }},
		{"missing rules", func(_, _, f map[string]any) { delete(f, "rules") }},
		{"wrong limit", func(r, _, _ map[string]any) { r["limit"] = 3 }},
		{"wrong radius", func(r, _, _ map[string]any) { r["radiusKm"] = 11 }},
		{"wrong sort", func(r, _, _ map[string]any) { r["sort"] = "price" }},
		{"wrong distance kind", func(r, _, _ map[string]any) { r["distanceKind"] = "route" }},
		{"missing notice", func(r, _, _ map[string]any) { delete(r, "availabilityNote") }},
		{"unknown status", func(r, _, _ map[string]any) { r["status"] = "success" }},
		{"false empty state", func(r, _, _ map[string]any) { r["status"] = "data_unavailable" }},
		{"missing matches", func(r, _, _ map[string]any) { delete(r, "matches") }},
		{"null matches", func(r, _, _ map[string]any) { r["matches"] = nil }},
		{"excess matches", func(r, m, _ map[string]any) { r["matches"] = []any{m, m, m, m, m, m} }},
		{"duplicate facility", func(r, m, _ map[string]any) { r["matches"] = []any{m, m} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var raw map[string]any
			_ = json.Unmarshal(unitResponseJSON(t, unitOKResponse(t)), &raw)
			match := raw["matches"].([]any)[0].(map[string]any)
			spot := match["facility"].(map[string]any)
			test.edit(raw, match, spot)
			result, err := decodeResponse(unitResponseJSON(t, raw), unitInput())
			if !errors.Is(err, ErrInvalidResponse) || result.Status != "" || result.Matches != nil {
				t.Fatal("incomplete or inconsistent API response accepted")
			}
		})
	}
}

func TestResponseRejectsMalformedOrAmbiguousJSONKeys(t *testing.T) {
	valid := string(unitResponseJSON(t, unitEmptyResponse()))
	for _, body := range []string{
		"", "{}", "[]", "null", valid + "{}", valid[:len(valid)-1],
		strings.Replace(valid, `"status":`, `"query":"synthetic private input","status":`, 1),
		strings.Replace(valid, `"status":`, `"status":"ok","status":`, 1),
		strings.Replace(valid, `"status":`, `"Status":`, 1),
		strings.Replace(valid, `"status":`, `"ſtatus":`, 1),
		strings.Replace(valid, `"status":`, `"ſtatus":"ok","status":`, 1),
		strings.Replace(valid, `"status":`, `"\u0073tatus":"ok","status":`, 1),
		strings.Replace(valid, `"availabilityNote":`, `"availabilityNote":"`+string([]byte{0xff})+`","ignored":`, 1),
		strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34),
	} {
		if result, err := decodeResponse([]byte(body), unitInput()); !errors.Is(err, ErrInvalidResponse) || result.Status != "" {
			t.Fatal("malformed JSON or ambiguous field accepted")
		}
	}
	withFacility := string(unitResponseJSON(t, unitOKResponse(t)))
	duplicate := strings.Replace(withFacility, `"latitude":`, `"latitude":1,"latitude":`, 1)
	if _, err := decodeResponse([]byte(duplicate), unitInput()); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal("nested duplicate key accepted")
	}
}

func TestResponseValidatesOrderingGenreAndPlaceConfirmation(t *testing.T) {
	response := unitOKResponse(t)
	input := unitInput()
	input.Genre = facility.GenreStreet
	if _, err := decodeResponse(unitResponseJSON(t, response), input); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal("genre mismatch accepted")
	}
	first := response.Matches[0]
	first.Facility.ID = "facility-z"
	second := first
	second.Facility.ID = "facility-a"
	response.Matches = []nearby.Match{first, second}
	if _, err := decodeResponse(unitResponseJSON(t, response), unitInput()); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal("unstable ID order accepted")
	}
	response.Matches[0].DistanceKm = 2
	response.Matches[1].DistanceKm = 1
	if _, err := decodeResponse(unitResponseJSON(t, response), unitInput()); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal("descending distance accepted")
	}
	for _, candidates := range [][]nearby.PlaceCandidate{
		nil, {{Label: "Only one"}}, {{Label: ""}, {Label: "Test B"}},
		{{Label: "Test\nA"}, {Label: "Test B"}}, {{Label: strings.Repeat("A", 301)}, {Label: "Test B"}},
		{{Label: "A"}, {Label: "B"}, {Label: "C"}, {Label: "D"}, {Label: "E"}, {Label: "F"}},
	} {
		response := unitEmptyResponse()
		response.Status, response.LocationCandidates = nearby.StatusLocationAmbiguous, candidates
		if _, err := decodeResponse(unitResponseJSON(t, response), unitInput()); !errors.Is(err, ErrInvalidResponse) {
			t.Fatal("invalid confirmation candidates accepted")
		}
	}
}

func FuzzDecodeResponse(f *testing.F) {
	f.Add([]byte(`{"status":"data_unavailable","matches":[],"distanceKind":"straight_line","limit":5,"radiusKm":10,"sort":"distance","availabilityNote":"Check sources"}`))
	f.Add([]byte(`{"matches":[{"facility":null,"distanceKm":null}]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxResponseBytes {
			return
		}
		result, err := decodeResponse(data, unitInput())
		if err != nil && (err != ErrInvalidResponse || result.Status != "" || result.Matches != nil) {
			t.Fatal("invalid response leaked details or partial data")
		}
	})
}
