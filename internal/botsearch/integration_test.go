package botsearch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/apiauth"
	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/httpapi"
	"github.com/kohei321dev/spot-diggz/internal/nearby"
)

func TestClientIntegrationAuthenticationPrecedesProvider(t *testing.T) {
	for _, test := range []struct {
		name       string
		credential func(*apiauth.Credential)
		wrongToken bool
		wantError  error
		wantCalls  int32
	}{
		{name: "valid", wantCalls: 1},
		{name: "unrecognized token", wrongToken: true, wantError: ErrUnauthorized},
		{name: "expired", credential: func(c *apiauth.Credential) { c.ExpiresAt = integrationNow().Add(-time.Second) }, wantError: ErrUnauthorized},
		{name: "expiry boundary", credential: func(c *apiauth.Credential) { c.ExpiresAt = integrationNow() }, wantError: ErrUnauthorized},
		{name: "revoked", credential: func(c *apiauth.Credential) { c.Revoked = true }, wantError: ErrUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			spot := integrationFacility("integration-park")
			provider := &integrationGeocoder{results: []geocoding.Result{{Label: "Resolved private marker", Location: spot.Location}}}
			fixture := integrationServer(t, []facility.Facility{spot}, provider, test.credential)
			client, token := fixture.client, fixture.token
			if test.wrongToken {
				token = integrationToken(t.Name() + "/unrecognized")
				client = fixture.newClient(t, token)
			}
			input := integrationInput(t, `{"query":"Private authentication input"}`)
			response, err := client.Search(context.Background(), input)
			if !errors.Is(err, test.wantError) {
				t.Fatal("unexpected authentication outcome")
			}
			if got := provider.calls.Load(); got != test.wantCalls {
				t.Fatalf("provider calls = %d, want %d", got, test.wantCalls)
			}
			if test.wantError == nil && (response.Status != nearby.StatusOK || len(response.Matches) != 1) {
				t.Fatal("authenticated HTTP search did not return the catalog spot")
			}
			if test.wantError != nil && (response.Status != "" || len(response.Matches) != 0) {
				t.Fatal("failed authentication returned search data")
			}
			integrationAssertPrivate(t, fixture, response, err, token, input.Query, "Resolved private marker")
		})
	}
}

func TestClientIntegrationPreservesAllSearchStatuses(t *testing.T) {
	for _, test := range []struct {
		name       string
		query      string
		places     int
		classify   bool
		wantStatus string
		wantCount  int
	}{
		{name: "matched", query: `{"query":"Private status input"}`, places: 1, classify: true, wantStatus: nearby.StatusOK, wantCount: 1},
		{name: "genre does not match", query: `{"query":"Private status input","genre":"street"}`, places: 1, classify: true, wantStatus: nearby.StatusNoMatches},
		{name: "unclassified catalog", query: `{"query":"Private status input"}`, places: 1, wantStatus: nearby.StatusDataUnavailable},
		{name: "location not found", query: `{"query":"Private status input"}`, classify: true, wantStatus: nearby.StatusLocationNotFound},
		{name: "ambiguous location", query: `{"query":"Private status input"}`, places: 2, classify: true, wantStatus: nearby.StatusLocationAmbiguous},
	} {
		t.Run(test.name, func(t *testing.T) {
			spot := integrationFacility("integration-park")
			if !test.classify {
				spot.Genre = ""
			}
			provider := &integrationGeocoder{}
			for index := 0; index < test.places; index++ {
				label := "確認候補A"
				if index == 1 {
					label = "確認候補B"
				}
				provider.results = append(provider.results, geocoding.Result{Label: label, Location: spot.Location})
			}
			fixture := integrationServer(t, []facility.Facility{spot}, provider, nil)
			input := integrationInput(t, test.query)
			response, err := fixture.client.Search(context.Background(), input)
			if err != nil || response.Status != test.wantStatus || response.Matches == nil || len(response.Matches) != test.wantCount {
				t.Fatalf("unexpected status contract: status=%s count=%d", response.Status, len(response.Matches))
			}
			if response.Limit != nearby.DefaultLimit || response.RadiusKm != nearby.DefaultRadiusKm || response.Sort != nearby.DistanceSort || response.DistanceKind != nearby.DistanceKind || response.AvailabilityNote == "" {
				t.Fatal("search defaults or safety note were lost")
			}
			if provider.calls.Load() != 1 || provider.lastQuery() != input.Query {
				t.Fatal("structured input did not reach the real API provider once")
			}
			if test.wantStatus == nearby.StatusLocationAmbiguous {
				if len(response.LocationCandidates) != 2 || response.LocationCandidates[0].Label != "確認候補A" || response.LocationCandidates[1].Label != "確認候補B" {
					t.Fatal("ambiguous labels were lost or automatically selected")
				}
			} else if len(response.LocationCandidates) != 0 {
				t.Fatal("non-ambiguous response exposed confirmation labels")
			}
			integrationAssertPrivate(t, fixture, response, err, input.Query)
		})
	}
}

func TestClientIntegrationAppliesGenreLimitRadiusAndStableDistanceOrder(t *testing.T) {
	base := integrationFacility("integration-origin")
	spots := []facility.Facility{}
	for _, spec := range []struct {
		id       string
		latitude float64
		genre    facility.Genre
	}{
		{id: "park-b", latitude: 0.003, genre: facility.GenreSkatepark},
		{id: "park-a", latitude: 0.003, genre: facility.GenreSkatepark},
		{id: "park-c", latitude: 0.008, genre: facility.GenreSkatepark},
		{id: "park-outside", latitude: 0.04, genre: facility.GenreSkatepark},
		{id: "street-nearest", latitude: 0.001, genre: facility.GenreStreet},
	} {
		spot := integrationFacility(spec.id)
		spot.Location.Latitude += spec.latitude
		spot.Genre = spec.genre
		spots = append(spots, spot)
	}
	provider := &integrationGeocoder{results: []geocoding.Result{{Label: "Private ordering label", Location: base.Location}}}
	fixture := integrationServer(t, spots, provider, nil)
	for _, test := range []struct {
		name string
		body string
		ids  []string
	}{
		{name: "limit and tie break", body: `{"query":"Private ordering input","genre":"skatepark","limit":2,"radiusKm":2,"sort":"distance"}`, ids: []string{"park-a", "park-b"}},
		{name: "radius independent from limit", body: `{"query":"Private ordering input","genre":"skatepark","limit":10,"radiusKm":0.6}`, ids: []string{"park-a", "park-b"}},
		{name: "larger radius", body: `{"query":"Private ordering input","genre":"skatepark","limit":10,"radiusKm":2}`, ids: []string{"park-a", "park-b", "park-c"}},
		{name: "both genres", body: `{"query":"Private ordering input","limit":10,"radiusKm":2}`, ids: []string{"street-nearest", "park-a", "park-b", "park-c"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := integrationInput(t, test.body)
			response, err := fixture.client.Search(context.Background(), input)
			if err != nil || response.Status != nearby.StatusOK || response.Limit != input.Limit || response.RadiusKm != input.RadiusKm {
				t.Fatal("search options were rejected or changed")
			}
			ids := make([]string, 0, len(response.Matches))
			previousDistance := -1.0
			for _, match := range response.Matches {
				ids = append(ids, match.Facility.ID)
				if match.DistanceKm < previousDistance || match.DistanceKm < 0 || match.DistanceKm > input.RadiusKm {
					t.Fatal("match distance violates order or radius")
				}
				previousDistance = match.DistanceKm
			}
			if !slices.Equal(ids, test.ids) {
				t.Fatal("genre, limit, radius or facility ID tie ordering changed")
			}
			integrationAssertPrivate(t, fixture, response, err, input.Query, "Private ordering label")
		})
	}
	if provider.calls.Load() != 4 {
		t.Fatal("client automatically retried or skipped an HTTP search")
	}
}

func TestClientIntegrationPreservesNonTimetabledStreetEvidence(t *testing.T) {
	for _, hoursStatus := range []facility.HoursStatus{facility.HoursNotApplicable, facility.HoursUnknown} {
		t.Run(string(hoursStatus), func(t *testing.T) {
			spot := integrationFacility("integration-street")
			spot.Genre, spot.HoursStatus = facility.GenreStreet, hoursStatus
			spot.Hours, spot.HoursBasis = nil, ""
			spot.HoursSourceURL = "https://example.com/integration-hours-policy"
			spot.GeneralUseStatus = facility.GeneralUseLimited
			spot.AvailabilityNote = "営業時間制度は非該当です。利用ルールを確認してください。"
			spot.EnglishTranslation.AvailabilityNote = "No opening-hours system applies. Check the usage rules."
			if hoursStatus == facility.HoursUnknown {
				spot.GeneralUseStatus = facility.GeneralUseScheduleCheckRequired
				spot.AvailabilityNote = "利用時間は未確認です。予定の確認が必要です。"
				spot.EnglishTranslation.AvailabilityNote = "Hours are unknown. Check the schedule."
			}
			provider := &integrationGeocoder{results: []geocoding.Result{{Label: "Private street label", Location: spot.Location}}}
			fixture := integrationServer(t, []facility.Facility{spot}, provider, nil)
			input := integrationInput(t, `{"query":"Private street input","genre":"street"}`)
			response, err := fixture.client.Search(context.Background(), input)
			if err != nil {
				t.Fatal("hours-state API response was rejected")
			}
			if hoursStatus == facility.HoursUnknown {
				if response.Status != nearby.StatusDataUnavailable || len(response.Matches) != 0 {
					t.Fatal("unknown hours became a search recommendation")
				}
			} else {
				if response.Status != nearby.StatusOK || len(response.Matches) != 1 {
					t.Fatal("evidenced non-timetabled street was lost")
				}
				got := response.Matches[0].Facility
				if got.HoursStatus != hoursStatus || got.HoursSourceURL != spot.HoursSourceURL || got.SkatingPermissionSourceURL != spot.SkatingPermissionSourceURL || got.AvailabilityNote != spot.AvailabilityNote || got.EnglishTranslation.AvailabilityNote != spot.EnglishTranslation.AvailabilityNote {
					t.Fatal("hours state, evidence or bilingual caution changed")
				}
				if len(got.Hours) != 0 || got.HoursBasis != "" {
					t.Fatal("client invented a timetable")
				}
			}
			integrationAssertPrivate(t, fixture, response, err, input.Query, "Private street label")
		})
	}
}

func TestClientIntegrationProviderFailureIsNotSuccessfulOrRetried(t *testing.T) {
	provider := &integrationGeocoder{err: errors.New("Synthetic private upstream diagnostic")}
	fixture := integrationServer(t, []facility.Facility{integrationFacility("integration-park")}, provider, nil)
	input := integrationInput(t, `{"query":"Private failure input"}`)
	response, err := fixture.client.Search(context.Background(), input)
	if !errors.Is(err, ErrUnavailable) || response.Status != "" || len(response.Matches) != 0 {
		t.Fatal("provider failure became success or usable search data")
	}
	if provider.calls.Load() != 1 {
		t.Fatal("provider failure triggered an automatic retry")
	}
	integrationAssertPrivate(t, fixture, response, err, input.Query, "Synthetic private upstream diagnostic")
}

type integrationFixture struct {
	client  *Client
	server  *httptest.Server
	logs    *integrationLogBuffer
	token   string
	pending sync.WaitGroup
}

func integrationServer(t *testing.T, spots []facility.Facility, provider geocoding.Provider, change func(*apiauth.Credential)) *integrationFixture {
	t.Helper()
	catalog, err := facility.NewCatalogAt(spots, integrationNow())
	if err != nil {
		t.Fatal("invalid synthetic catalog fixture")
	}
	token := integrationToken(t.Name())
	digest := sha256.Sum256([]byte(token))
	credential := apiauth.Credential{
		ClientID: "integration-bot", OwnerID: "integration-owner", Scope: apiauth.ReadScope,
		ExpiresAt: integrationNow().Add(time.Hour), TokenSHA256: hex.EncodeToString(digest[:]),
	}
	if change != nil {
		change(&credential)
	}
	raw, err := json.Marshal([]apiauth.Credential{credential})
	if err != nil {
		t.Fatal("could not encode synthetic credential fixture")
	}
	credentials, err := apiauth.Parse(string(raw), "integration-owner", integrationNow)
	if err != nil {
		t.Fatal("invalid synthetic credential fixture")
	}
	logs := &integrationLogBuffer{}
	fixture := &integrationFixture{logs: logs, token: token}
	handler := httpapi.NewReadAPI(catalog, credentials, provider, slog.New(slog.NewJSONHandler(logs, nil)), nil, integrationNow)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture.pending.Add(1)
		defer fixture.pending.Done()
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	fixture.server = server
	fixture.client = fixture.newClient(t, token)
	return fixture
}

func (fixture *integrationFixture) newClient(t *testing.T, token string) *Client {
	t.Helper()
	client, err := NewClient(fixture.server.URL, token)
	if err != nil {
		t.Fatal("could not create integration API client")
	}
	// Trust only the test certificate; keep the real client's transport and limits.
	roots := x509.NewCertPool()
	roots.AddCert(fixture.server.Certificate())
	client.httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: roots}
	t.Cleanup(client.CloseIdleConnections)
	return client
}

func integrationInput(t *testing.T, raw string) nearby.Input {
	t.Helper()
	input, err := nearby.DecodeInput([]byte(raw))
	if err != nil {
		t.Fatal("invalid structured input fixture")
	}
	return input
}

func integrationToken(seed string) string {
	digest := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(digest[:])
}

func integrationNow() time.Time {
	// Historical fixture time intentionally differs from the client's wall clock.
	return time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
}

func integrationFacility(id string) facility.Facility {
	verifiedAt := integrationNow().Add(-24 * time.Hour)
	return facility.Facility{
		ID: id, Name: "統合テスト施設", Address: "大阪府大阪市の合成テスト住所", Prefecture: "大阪府", Municipality: "大阪市",
		Genre: facility.GenreSkatepark, SkatingPermissionSourceURL: "https://example.com/integration-permission",
		Location: facility.Location{Latitude: 34.6937, Longitude: 135.5023}, Activities: []string{"skateboard"},
		Hours: []facility.OperatingHours{{Day: "daily", Opens: "09:00", Closes: "18:00"}}, GeneralUseStatus: facility.GeneralUseRegular,
		Price: "合成料金500円", Reservation: "合成テストの受付案内", Features: []string{"flat-area"}, Rules: []string{"ヘルメット必須"},
		Access: facility.Access{Notes: "合成テストのアクセス案内"}, ScheduleNotes: []string{"公式情報を確認してください。"},
		EnglishTranslation: facility.FacilityEnglishTranslation{
			Name: "Synthetic integration facility", Address: "Synthetic address, Osaka City, Osaka", Price: "Synthetic price JPY 500",
			Reservation: "Synthetic registration guidance", Rules: []string{"Helmet required"}, AccessNotes: "Synthetic access guidance",
			ScheduleNotes: []string{"Check the official source."},
		},
		SourceURL: "https://example.com/integration-source", SourceType: "official", Status: "verified", Confidence: "high",
		VerifiedAt: verifiedAt, DynamicVerifiedAt: verifiedAt, StableVerifiedAt: verifiedAt,
	}
}

func integrationAssertPrivate(t *testing.T, fixture *integrationFixture, response nearby.Response, searchError error, private ...string) {
	t.Helper()
	// A response may reach the client before the handler's access log is written.
	fixture.pending.Wait()
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal("could not inspect client result")
	}
	diagnostic := ""
	if searchError != nil {
		diagnostic = searchError.Error()
	}
	for _, value := range append(private, fixture.token) {
		if value != "" && (strings.Contains(string(encoded), value) || strings.Contains(diagnostic, value) || strings.Contains(fixture.logs.String(), value)) {
			t.Fatal("private input, credential or upstream diagnostic leaked")
		}
	}
}

type integrationGeocoder struct {
	results []geocoding.Result
	err     error
	calls   atomic.Int32
	mu      sync.Mutex
	query   string
}

func (provider *integrationGeocoder) Search(_ context.Context, query string) ([]geocoding.Result, error) {
	provider.calls.Add(1)
	provider.mu.Lock()
	provider.query = query
	provider.mu.Unlock()
	return provider.results, provider.err
}

func (provider *integrationGeocoder) lastQuery() string {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	return provider.query
}

type integrationLogBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (buffer *integrationLogBuffer) Write(value []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.Write(value)
}

func (buffer *integrationLogBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.String()
}
