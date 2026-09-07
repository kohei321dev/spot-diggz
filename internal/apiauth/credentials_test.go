package apiauth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCredentialsAuthorizeOnlyActiveReadClients(t *testing.T) {
	now := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	// Deterministic synthetic credential, never used outside this test.
	tokenBytes := sha256.Sum256([]byte(t.Name()))
	token := hex.EncodeToString(tokenBytes[:])
	digest := sha256.Sum256([]byte(token))
	base := Credential{ClientID: "test-bot", OwnerID: "owner", Scope: ReadScope, TokenSHA256: hex.EncodeToString(digest[:]), ExpiresAt: now.Add(time.Hour)}
	for _, test := range []struct {
		name    string
		change  func(*Credential)
		allowed bool
	}{
		{"active", func(*Credential) {}, true},
		{"expired", func(c *Credential) { c.ExpiresAt = now.Add(-time.Second) }, false},
		{"expiry boundary", func(c *Credential) { c.ExpiresAt = now }, false},
		{"revoked", func(c *Credential) { c.Revoked = true }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := base
			test.change(&client)
			encoded, _ := json.Marshal([]Credential{client})
			credentials, err := Parse(string(encoded), "owner", func() time.Time { return now })
			if err != nil {
				t.Fatal(err)
			}
			if credentials.Authorize("Bearer "+token) != test.allowed {
				t.Fatal("authorization mismatch")
			}
			unknown := sha256.Sum256([]byte("unknown " + t.Name()))
			if credentials.Authorize("Bearer " + hex.EncodeToString(unknown[:])) {
				t.Fatal("unknown token accepted")
			}
			for _, header := range []string{"", token, "Basic " + token, "Bearer  " + token, "Bearer " + token + " ", "Bearer " + strings.ToUpper(token), "Bearer " + token[:62]} {
				if credentials.Authorize(header) {
					t.Fatal("malformed authorization accepted")
				}
			}
		})
	}
	for _, change := range []func(*Credential){
		func(c *Credential) { c.OwnerID = "someone-else" },
		func(c *Credential) { c.Scope = "facilities:write" },
		func(c *Credential) { c.ExpiresAt = time.Time{} },
		func(c *Credential) { c.TokenSHA256 = "not-a-digest" },
		func(c *Credential) { c.ClientID = "" },
	} {
		client := base
		change(&client)
		encoded, _ := json.Marshal([]Credential{client})
		if _, err := Parse(string(encoded), "owner", nil); err == nil {
			t.Fatal("invalid client configuration accepted")
		}
	}
	duplicate, _ := json.Marshal([]Credential{base, base})
	for _, raw := range []string{"", "null", "[]", "{}", string(duplicate), "[] {}", strings.Repeat(" ", maxConfigBytes+1)} {
		if _, err := Parse(raw, "owner", nil); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
	valid, _ := json.Marshal([]Credential{base})
	for _, change := range []string{`"revoked":null`, `"revoked":true,"revoked":false`, `"Revoked":false`, `"revoked":false,"unknown":1`} {
		raw := strings.Replace(string(valid), `"revoked":false`, change, 1)
		if _, err := Parse(raw, "owner", nil); err == nil {
			t.Fatal("ambiguous credential record accepted")
		}
	}
	var absent *Credentials
	if absent.Authorize("Bearer " + token) {
		t.Fatal("nil credentials authorized")
	}
}

func TestRevokingOneClientDoesNotRevokeAnother(t *testing.T) {
	now := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	var clients []Credential
	var tokens []string
	for _, name := range []string{"slack-test", "discord-test"} {
		seed := sha256.Sum256([]byte(t.Name() + name))
		token := hex.EncodeToString(seed[:])
		digest := sha256.Sum256([]byte(token))
		tokens = append(tokens, token)
		clients = append(clients, Credential{ClientID: name, OwnerID: "owner", Scope: ReadScope, TokenSHA256: hex.EncodeToString(digest[:]), ExpiresAt: now.Add(time.Hour)})
	}
	clients[0].Revoked = true
	raw, _ := json.Marshal(clients)
	credentials, err := Parse(string(raw), "owner", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if credentials.Authorize("Bearer "+tokens[0]) || !credentials.Authorize("Bearer "+tokens[1]) {
		t.Fatal("per-client revocation failed")
	}
}
