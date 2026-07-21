package upload

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"testing"
	"time"
)

// TestHMACCredentialsIntercept guards against a transcription error when
// porting the signing algorithm off the SpecterOps SDK: it independently
// recomputes the expected signature (keyed off the RequestDate intercept
// actually emitted, to avoid flaking on wall-clock) and checks it matches.
func TestHMACCredentialsIntercept(t *testing.T) {
	creds := newHMACCredentials("test-token-key", "test-token-id")

	body := []byte(`{"hello":"world"}`)
	req, err := http.NewRequest(http.MethodPost, "https://example.com/api/v2/test/resource?x=1", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	if err := creds.intercept(context.Background(), req); err != nil {
		t.Fatalf("intercept returned error: %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "bhesignature test-token-id" {
		t.Errorf("Authorization = %q", got)
	}
	if got := req.Header.Get("User-Agent"); got != "bhe-go-sdk 0001" {
		t.Errorf("User-Agent = %q", got)
	}

	requestDate := req.Header.Get("RequestDate")
	if _, err := time.Parse(time.RFC3339, requestDate); err != nil {
		t.Fatalf("RequestDate not RFC3339: %q (%v)", requestDate, err)
	}

	digester := hmac.New(sha256.New, []byte("test-token-key"))
	digester.Write([]byte(req.Method + req.URL.RequestURI()))
	digester = hmac.New(sha256.New, digester.Sum(nil))
	digester.Write([]byte(requestDate[:13]))
	digester = hmac.New(sha256.New, digester.Sum(nil))
	digester.Write(body)
	want := base64.StdEncoding.EncodeToString(digester.Sum(nil))

	if got := req.Header.Get("Signature"); got != want {
		t.Errorf("Signature = %q, want %q", got, want)
	}

	replayed, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read replayed body: %v", err)
	}
	if !bytes.Equal(replayed, body) {
		t.Errorf("body not preserved after signing: got %q, want %q", replayed, body)
	}
}
