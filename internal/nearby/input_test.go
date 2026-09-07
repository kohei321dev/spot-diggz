package nearby

import (
	"strings"
	"testing"
)

func TestDecodeInputAppliesDefaultsAndIndependentBounds(t *testing.T) {
	input, err := DecodeInput([]byte(`{"query":"  テスト駅  "}`))
	if err != nil || input.Query != "テスト駅" || input.Limit != DefaultLimit || input.RadiusKm != DefaultRadiusKm || input.Sort != DistanceSort || input.Genre != "" {
		t.Fatalf("defaults: %+v %v", input, err)
	}
	for _, body := range []string{
		`{"query":"駅","genre":"street","limit":1,"radiusKm":0.1,"sort":"distance"}`,
		`{"query":"駅","genre":"skatepark","limit":10,"radiusKm":50}`,
	} {
		if _, err := DecodeInput([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDecodeInputRejectsInvalidAndAmbiguousJSON(t *testing.T) {
	for _, body := range []string{
		`{}`, `null`, `[]`, `{"query":null}`, `{"query":" "}`, `{"query":123}`,
		`{"query":"駅","query":"別駅"}`, `{"query":"駅","unknown":1}`, `{"Query":"駅"}`,
		`{"query":"駅","limit":0}`, `{"query":"駅","limit":11}`, `{"query":"駅","limit":1.5}`,
		`{"query":"駅","limit":"3"}`, `{"query":"駅","limit":null}`,
		`{"query":"駅","radiusKm":0.09}`, `{"query":"駅","radiusKm":50.1}`, `{"query":"駅","radiusKm":1e999}`,
		`{"query":"駅","genre":"stree"}`, `{"query":"駅","genre":""}`, `{"query":"駅","sort":"price"}`,
		`{"query":"駅"} {}`, `{"query":"駅\n名"}`, `{"query":"\n駅"}`, "{\"query\":\"\xff\"}", `{"query":"` + strings.Repeat("駅", 121) + `"}`,
	} {
		t.Run(body, func(t *testing.T) {
			if _, err := DecodeInput([]byte(body)); err == nil {
				t.Fatal("invalid JSON accepted")
			}
		})
	}
}
