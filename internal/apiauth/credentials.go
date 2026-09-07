// Package apiauth authorizes explicitly provisioned, read-only API clients.
// It does not authenticate the human using a bot; the bot must do that separately.
package apiauth

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

const (
	ReadScope           = "facilities:read"
	TokenBytes          = 32
	maxClients          = 32
	maxConfigBytes      = 32 * 1024
	maxIdentifierLength = 64
)

type Credential struct {
	ClientID    string    `json:"clientId"`
	OwnerID     string    `json:"ownerId"`
	TokenSHA256 string    `json:"tokenSha256"`
	Scope       string    `json:"scope"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Revoked     bool      `json:"revoked"`
}

type credentialDigest struct {
	digest    [sha256.Size]byte
	expiresAt time.Time
	revoked   bool
}

// Credentials is immutable. Revocation config takes effect on process replacement.
type Credentials struct {
	clients []credentialDigest
	now     func() time.Time
}

// Parse rejects incomplete or ambiguous configuration without echoing its contents.
func Parse(raw string, ownerID string, now func() time.Time) (*Credentials, error) {
	invalid := errors.New("invalid API client configuration")
	if !validIdentifier(ownerID) || len(raw) > maxConfigBytes {
		return nil, invalid
	}
	var configured []json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&configured); err != nil {
		return nil, invalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, invalid
	}
	if len(configured) == 0 || len(configured) > maxClients {
		return nil, invalid
	}
	if now == nil {
		now = time.Now
	}
	result := &Credentials{now: now}
	seenIDs := make(map[string]bool)
	seenDigests := make(map[string]bool)
	for _, rawCredential := range configured {
		credential, err := decodeCredential(rawCredential)
		if err != nil {
			return nil, invalid
		}
		if !validIdentifier(credential.ClientID) || credential.OwnerID != ownerID ||
			credential.Scope != ReadScope || credential.ExpiresAt.IsZero() ||
			seenIDs[credential.ClientID] || seenDigests[credential.TokenSHA256] {
			return nil, invalid
		}
		digest, err := decodeHex32(credential.TokenSHA256)
		if err != nil {
			return nil, invalid
		}
		seenIDs[credential.ClientID] = true
		seenDigests[credential.TokenSHA256] = true
		result.clients = append(result.clients, credentialDigest{digest: digest, expiresAt: credential.ExpiresAt, revoked: credential.Revoked})
	}
	return result, nil
}

// Configuration errors must not silently enable a client (e.g. revoked:null or
// duplicate revoked fields). Field spelling is exact, unlike Go's default decoder.
func decodeCredential(raw json.RawMessage) (Credential, error) {
	var credential Credential
	invalid := errors.New("invalid credential record")
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return credential, invalid
	}
	seen := make(map[string]bool)
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] {
			return credential, invalid
		}
		seen[key] = true
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return credential, invalid
		}
		switch key {
		case "clientId":
			err = json.Unmarshal(value, &credential.ClientID)
		case "ownerId":
			err = json.Unmarshal(value, &credential.OwnerID)
		case "tokenSha256":
			err = json.Unmarshal(value, &credential.TokenSHA256)
		case "scope":
			err = json.Unmarshal(value, &credential.Scope)
		case "expiresAt":
			err = json.Unmarshal(value, &credential.ExpiresAt)
		case "revoked":
			err = json.Unmarshal(value, &credential.Revoked)
		default:
			return credential, invalid
		}
		if err != nil {
			return credential, invalid
		}
	}
	if token, err := decoder.Token(); err != nil || token != json.Delim('}') {
		return credential, invalid
	}
	return credential, nil
}

// Authorize accepts only an Authorization header value, never query or cookie data.
func (credentials *Credentials) Authorize(header string) bool {
	if credentials == nil {
		return false
	}
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return false
	}
	if _, err := decodeHex32(token); err != nil {
		return false
	}
	digest := sha256.Sum256([]byte(token))
	allowed := false
	now := credentials.now()
	for _, client := range credentials.clients {
		matches := subtle.ConstantTimeCompare(digest[:], client.digest[:]) == 1
		if matches && !client.revoked && now.Before(client.expiresAt) {
			allowed = true
		}
	}
	return allowed
}

func decodeHex32(value string) ([TokenBytes]byte, error) {
	var result [TokenBytes]byte
	if len(value) != hex.EncodedLen(TokenBytes) || value != strings.ToLower(value) {
		return result, errors.New("invalid digest format")
	}
	_, err := hex.Decode(result[:], []byte(value))
	return result, err
}

func validIdentifier(value string) bool {
	if len(value) == 0 || len(value) > maxIdentifierLength {
		return false
	}
	for _, ch := range value {
		if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' && ch != '_' {
			return false
		}
	}
	return true
}
